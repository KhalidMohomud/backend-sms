package service

import (
	"backendapi/internal/authz"
	"backendapi/internal/model"
	"backendapi/internal/repository"
	"backendapi/internal/security"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
)

var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrAccountUnavailable = errors.New("account is locked or disabled")
)

type LoginInput struct {
	Username string `json:"username" binding:"required,min=3,max=50" example:"superadmin"`
	Password string `json:"password" binding:"required,max=72" example:"a-strong-password"`
}

type AuthResult struct {
	AccessToken      string          `json:"access_token"`
	RefreshToken     string          `json:"refresh_token"`
	TokenType        string          `json:"token_type" example:"Bearer"`
	ExpiresAt        time.Time       `json:"expires_at"`
	RefreshExpiresAt time.Time       `json:"refresh_expires_at"`
	User             *model.User     `json:"user"`
	Principal        authz.Principal `json:"authorization"`
}

type RefreshInput struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

type LogoutInput struct {
	RefreshToken string `json:"refresh_token"`
}

type ChangePasswordInput struct {
	CurrentPassword string `json:"current_password" binding:"required,max=72"`
	NewPassword     string `json:"new_password" binding:"required,min=12,max=72"`
}

type ResetPasswordInput struct {
	ResetToken  string `json:"reset_token" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=12,max=72"`
}

type AuthService struct {
	users    repository.UserRepository
	jwt      *security.JWTManager
	audit    *AuditWriter
	sessions security.SessionRepository
}

func NewAuthService(users repository.UserRepository, jwt *security.JWTManager, audit *AuditWriter, sessions security.SessionRepository) *AuthService {
	return &AuthService{users: users, jwt: jwt, audit: audit, sessions: sessions}
}

func (s *AuthService) Login(ctx context.Context, input LoginInput, request RequestMetadata) (*AuthResult, error) {
	username := strings.ToLower(strings.TrimSpace(input.Username))
	if err := s.sessions.AllowLogin(ctx, request.IPAddress, username); err != nil {
		if errors.Is(err, security.ErrRateLimited) {
			return nil, security.ErrRateLimited
		}
		return nil, err
	}
	user, err := s.users.FindByUsername(ctx, username)
	if errors.Is(err, repository.ErrNotFound) {
		// Perform bcrypt work even for an unknown username to reduce timing differences.
		_, _ = security.HashPassword(input.Password)
		if err := s.audit.Write(ctx, nil, nil, "LOGIN_FAILED", "users", nil, request, map[string]any{"username": username}); err != nil {
			return nil, err
		}
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("find login account: %w", err)
	}

	if security.CheckPassword(user.PasswordHash, input.Password) != nil {
		if user.Status == model.UserStatusActive {
			if err := s.users.RecordFailedLogin(ctx, user.ID); err != nil {
				return nil, fmt.Errorf("record failed login: %w", err)
			}
		}
		if err := s.audit.Write(ctx, &user.ID, user.SchoolID, "LOGIN_FAILED", "users", &user.ID, request, nil); err != nil {
			return nil, err
		}
		return nil, ErrInvalidCredentials
	}

	if !userAvailable(user) {
		if err := s.audit.Write(ctx, &user.ID, user.SchoolID, "LOGIN_BLOCKED", "users", &user.ID, request, map[string]any{"status": user.Status}); err != nil {
			return nil, err
		}
		return nil, ErrAccountUnavailable
	}

	now := time.Now().UTC()
	if err := s.users.RecordSuccessfulLogin(ctx, user.ID, now); err != nil {
		return nil, fmt.Errorf("record successful login: %w", err)
	}
	user.FailedLogins = 0
	user.LastLogin = &now
	if err := s.audit.Write(ctx, &user.ID, user.SchoolID, "LOGIN", "users", &user.ID, request, nil); err != nil {
		return nil, err
	}
	return s.issueTokens(ctx, user)
}

func (s *AuthService) Refresh(ctx context.Context, input RefreshInput) (*AuthResult, error) {
	userID, newRefresh, refreshExpiresAt, err := s.sessions.RotateRefresh(ctx, input.RefreshToken)
	if err != nil {
		return nil, err
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil || !userAvailable(user) {
		_ = s.sessions.RevokeRefresh(ctx, newRefresh)
		return nil, ErrAccountUnavailable
	}
	result, err := s.issueAccessToken(user, newRefresh, refreshExpiresAt)
	if err != nil {
		_ = s.sessions.RevokeRefresh(ctx, newRefresh)
	}
	return result, err
}

func (s *AuthService) Logout(ctx context.Context, accessToken string, input LogoutInput) error {
	identity, err := s.jwt.ParseIdentity(accessToken)
	if err != nil {
		return err
	}
	if err := s.sessions.DenyAccess(ctx, identity.JTI, identity.ExpiresAt); err != nil {
		return err
	}
	return s.sessions.RevokeRefresh(ctx, input.RefreshToken)
}

func (s *AuthService) LogoutAll(ctx context.Context, actor authz.Principal) error {
	return s.sessions.RevokeAll(ctx, actor.UserID)
}

func (s *AuthService) ChangePassword(ctx context.Context, actor authz.Principal, input ChangePasswordInput, request RequestMetadata) error {
	user, err := s.users.FindByID(ctx, actor.UserID)
	if err != nil {
		return err
	}
	if security.CheckPassword(user.PasswordHash, input.CurrentPassword) != nil {
		return ErrInvalidCredentials
	}
	return s.setPassword(ctx, user, input.NewPassword, &actor.UserID, request)
}

func (s *AuthService) ResetPassword(ctx context.Context, input ResetPasswordInput, request RequestMetadata) error {
	userID, err := s.sessions.ConsumePasswordReset(ctx, input.ResetToken)
	if err != nil {
		return err
	}
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return err
	}
	return s.setPassword(ctx, user, input.NewPassword, &userID, request)
}

func (s *AuthService) setPassword(ctx context.Context, user *model.User, password string, actorID *uint64, request RequestMetadata) error {
	hash, err := security.HashPassword(password)
	if err != nil {
		return fmt.Errorf("hash password: %w", err)
	}
	if err := s.users.UpdatePassword(ctx, user.ID, hash); err != nil {
		return err
	}
	if err := s.sessions.RevokeAll(ctx, user.ID); err != nil {
		return err
	}
	return s.audit.Write(ctx, actorID, user.SchoolID, "PASSWORD_CHANGE", "users", &user.ID, request, nil)
}

func (s *AuthService) issueTokens(ctx context.Context, user *model.User) (*AuthResult, error) {
	refresh, refreshExpiresAt, err := s.sessions.CreateRefresh(ctx, user.ID)
	if err != nil {
		return nil, err
	}
	result, err := s.issueAccessToken(user, refresh, refreshExpiresAt)
	if err != nil {
		_ = s.sessions.RevokeRefresh(ctx, refresh)
	}
	return result, err
}

func (s *AuthService) issueAccessToken(user *model.User, refresh string, refreshExpiresAt time.Time) (*AuthResult, error) {
	principal := authz.FromUser(user)
	token, expiresAt, err := s.jwt.Generate(principal)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}
	return &AuthResult{AccessToken: token, RefreshToken: refresh, TokenType: "Bearer", ExpiresAt: expiresAt, RefreshExpiresAt: refreshExpiresAt, User: user, Principal: principal}, nil
}

func userAvailable(user *model.User) bool {
	return user != nil && user.Status == model.UserStatusActive && user.Role.Status != model.RoleStatusInactive && (user.SchoolID == nil || (user.School != nil && user.School.Status == model.SchoolStatusActive))
}
