package service

import (
	"backendapi/internal/model"
	"backendapi/internal/repository"
	"backendapi/internal/security"
	"context"
	"errors"
	"fmt"
	"strings"
)

var ErrSuperAdminExists = errors.New("a SuperAdmin account already exists")

type AdminService struct {
	users      repository.UserRepository
	foundation repository.FoundationRepository
	audit      *AuditWriter
}

func NewAdminService(users repository.UserRepository, foundation repository.FoundationRepository, audit *AuditWriter) *AdminService {
	return &AdminService{users: users, foundation: foundation, audit: audit}
}

func (s *AdminService) CreateInitialSuperAdmin(ctx context.Context, username, password string) (*model.User, error) {
	username = strings.ToLower(strings.TrimSpace(username))
	if len(username) < 3 || len(username) > 50 {
		return nil, fmt.Errorf("username must contain 3 to 50 characters")
	}
	if len(password) < 12 || len(password) > 72 {
		return nil, fmt.Errorf("password must contain 12 to 72 bytes")
	}
	count, err := s.users.CountSuperAdmins(ctx)
	if err != nil {
		return nil, fmt.Errorf("count SuperAdmin accounts: %w", err)
	}
	if count != 0 {
		return nil, ErrSuperAdminExists
	}
	role, err := s.foundation.FindRoleByName(ctx, model.RoleSuperAdmin)
	if err != nil {
		return nil, fmt.Errorf("find SuperAdmin role: %w", err)
	}
	hash, err := security.HashPassword(password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	user := &model.User{
		Username: username, PasswordHash: hash, RoleID: role.ID, Role: *role, Status: model.UserStatusActive,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("create SuperAdmin: %w", err)
	}
	if err := s.audit.Write(ctx, &user.ID, nil, "INSERT", "users", &user.ID, RequestMetadata{IPAddress: "operator-cli"}, map[string]any{"role": model.RoleSuperAdmin}); err != nil {
		return nil, err
	}
	return user, nil
}
