package service

import (
	"backendapi/internal/model"
	"backendapi/internal/repository"
	"backendapi/internal/security"
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

var (
	ErrEmailTaken         = errors.New("email is already registered")
	ErrInvalidCredentials = errors.New("invalid email or password")
)

type RegisterInput struct {
	Name     string `json:"name" binding:"required,min=2,max=120" example:"Amina Ali"`
	Email    string `json:"email" binding:"required,email" example:"amina@example.com"`
	Password string `json:"password" binding:"required,min=8,max=72" example:"password123"`
}

type LoginInput struct {
	Email    string `json:"email" binding:"required,email" example:"amina@example.com"`
	Password string `json:"password" binding:"required" example:"password123"`
}

type AuthResult struct {
	Token string      `json:"token"`
	User  *model.User `json:"user"`
}

type AuthService struct {
	users repository.UserRepository
	jwt   *security.JWTManager
}

func NewAuthService(users repository.UserRepository, jwt *security.JWTManager) *AuthService {
	return &AuthService{users: users, jwt: jwt}
}

func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if _, err := s.users.FindByEmail(ctx, email); err == nil {
		return nil, ErrEmailTaken
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, fmt.Errorf("check existing user: %w", err)
	}

	hash, err := security.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	user := &model.User{Name: strings.TrimSpace(input.Name), Email: email, PasswordHash: hash}
	if err := s.users.Create(ctx, user); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrEmailTaken
		}
		return nil, fmt.Errorf("create user: %w", err)
	}

	token, err := s.jwt.Generate(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}
	return &AuthResult{Token: token, User: user}, nil
}

func (s *AuthService) Login(ctx context.Context, input LoginInput) (*AuthResult, error) {
	user, err := s.users.FindByEmail(ctx, strings.ToLower(strings.TrimSpace(input.Email)))
	if errors.Is(err, repository.ErrNotFound) {
		return nil, ErrInvalidCredentials
	}
	if err != nil {
		return nil, fmt.Errorf("find user: %w", err)
	}
	if security.CheckPassword(user.PasswordHash, input.Password) != nil {
		return nil, ErrInvalidCredentials
	}
	token, err := s.jwt.Generate(user.ID)
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}
	return &AuthResult{Token: token, User: user}, nil
}
