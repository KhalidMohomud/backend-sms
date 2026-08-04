package service

import (
	"backendapi/internal/config"
	"backendapi/internal/model"
	"backendapi/internal/repository"
	"backendapi/internal/security"
	"context"
	"errors"
	"testing"
	"time"
)

func TestLoginLocksAccountAfterFiveFailedAttempts(t *testing.T) {
	hash, err := security.HashPassword("correct-password")
	if err != nil {
		t.Fatal(err)
	}
	users := &memoryUsers{user: &model.User{
		ID: 1, Username: "admin", PasswordHash: hash, Status: model.UserStatusActive,
		Role: model.Role{Name: model.RoleSchoolAdmin},
	}}
	audits := &memoryAudits{}
	manager := security.NewJWTManager(config.JWTConfig{Secret: "test-secret-long-enough", Expiration: time.Hour, Issuer: "test"})
	service := NewAuthService(users, manager, NewAuditWriter(audits))

	for attempt := 1; attempt <= 5; attempt++ {
		_, err := service.Login(context.Background(), LoginInput{Username: "admin", Password: "wrong-password"}, RequestMetadata{})
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("attempt %d error = %v, want ErrInvalidCredentials", attempt, err)
		}
	}
	if users.user.Status != model.UserStatusLocked || users.user.FailedLogins != 5 {
		t.Fatalf("user status = %s, failures = %d; want locked, 5", users.user.Status, users.user.FailedLogins)
	}
	_, err = service.Login(context.Background(), LoginInput{Username: "admin", Password: "correct-password"}, RequestMetadata{})
	if !errors.Is(err, ErrAccountUnavailable) {
		t.Fatalf("locked login error = %v, want ErrAccountUnavailable", err)
	}
}

func TestSuccessfulLoginResetsFailuresAndReturnsTenantClaims(t *testing.T) {
	hash, _ := security.HashPassword("correct-password")
	schoolID := uint64(8)
	users := &memoryUsers{user: &model.User{
		ID: 2, SchoolID: &schoolID, Username: "registrar", PasswordHash: hash,
		Status: model.UserStatusActive, FailedLogins: 3,
		Role: model.Role{Name: model.RoleRegistrar, Permissions: []model.Permission{{Name: "manage_students"}}},
	}}
	audits := &memoryAudits{}
	manager := security.NewJWTManager(config.JWTConfig{Secret: "test-secret-long-enough", Expiration: time.Hour, Issuer: "test"})
	service := NewAuthService(users, manager, NewAuditWriter(audits))

	result, err := service.Login(context.Background(), LoginInput{Username: "registrar", Password: "correct-password"}, RequestMetadata{})
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if users.user.FailedLogins != 0 || result.Principal.SchoolID == nil || *result.Principal.SchoolID != schoolID {
		t.Fatalf("unexpected successful login result: %#v", result)
	}
	parsed, err := manager.Parse(result.AccessToken)
	if err != nil || parsed.UserID != users.user.ID || parsed.Role != model.RoleRegistrar {
		t.Fatalf("parsed token = %#v, error = %v", parsed, err)
	}
}

type memoryUsers struct{ user *model.User }

func (m *memoryUsers) Create(_ context.Context, user *model.User) error { m.user = user; return nil }
func (m *memoryUsers) FindByUsername(_ context.Context, username string) (*model.User, error) {
	if m.user == nil || username != m.user.Username {
		return nil, repository.ErrNotFound
	}
	return m.user, nil
}
func (m *memoryUsers) FindByID(_ context.Context, id uint64) (*model.User, error) {
	if m.user == nil || id != m.user.ID {
		return nil, repository.ErrNotFound
	}
	return m.user, nil
}
func (m *memoryUsers) RecordFailedLogin(_ context.Context, _ uint64) error {
	if m.user.Status == model.UserStatusActive {
		m.user.FailedLogins++
		if m.user.FailedLogins >= 5 {
			m.user.FailedLogins = 5
			m.user.Status = model.UserStatusLocked
		}
	}
	return nil
}
func (m *memoryUsers) RecordSuccessfulLogin(_ context.Context, _ uint64, at time.Time) error {
	m.user.FailedLogins = 0
	m.user.LastLogin = &at
	return nil
}
func (m *memoryUsers) UpdateStatus(_ context.Context, _ uint64, status model.UserStatus) error {
	m.user.Status = status
	return nil
}
func (m *memoryUsers) CountSuperAdmins(context.Context) (int64, error) { return 0, nil }

type memoryAudits struct{ entries []model.AuditLog }

func (m *memoryAudits) Create(_ context.Context, entry *model.AuditLog) error {
	m.entries = append(m.entries, *entry)
	return nil
}
func (m *memoryAudits) List(context.Context, *uint64, int, int) ([]model.AuditLog, error) {
	return m.entries, nil
}
