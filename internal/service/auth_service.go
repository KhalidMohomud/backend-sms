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
	AccessToken string          `json:"access_token"`
	TokenType   string          `json:"token_type" example:"Bearer"`
	ExpiresAt   time.Time       `json:"expires_at"`
	User        *model.User     `json:"user"`
	Principal   authz.Principal `json:"authorization"`
}

type AuthService struct {
	users repository.UserRepository
	jwt   *security.JWTManager
	audit *AuditWriter
}

func NewAuthService(users repository.UserRepository, jwt *security.JWTManager, audit *AuditWriter) *AuthService {
	return &AuthService{users: users, jwt: jwt, audit: audit}
}

func (s *AuthService) Login(ctx context.Context, input LoginInput, request RequestMetadata) (*AuthResult, error) {
	username := strings.ToLower(strings.TrimSpace(input.Username))
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

	if user.Status != model.UserStatusActive || (user.SchoolID != nil && (user.School == nil || user.School.Status != model.SchoolStatusActive)) {
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
	principal := authz.FromUser(user)
	token, expiresAt, err := s.jwt.Generate(principal)
	if err != nil {
		return nil, fmt.Errorf("generate access token: %w", err)
	}
	if err := s.audit.Write(ctx, &user.ID, user.SchoolID, "LOGIN", "users", &user.ID, request, nil); err != nil {
		return nil, err
	}

	return &AuthResult{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresAt:   expiresAt,
		User:        user,
		Principal:   principal,
	}, nil
}
