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
	service := NewAuthService(users, manager, NewAuditWriter(audits), newMemorySessions())

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
		School: &model.School{ID: schoolID, Status: model.SchoolStatusActive},
		Role:   model.Role{Name: model.RoleRegistrar, Permissions: []model.Permission{{Name: "manage_students"}}},
	}}
	audits := &memoryAudits{}
	manager := security.NewJWTManager(config.JWTConfig{Secret: "test-secret-long-enough", Expiration: time.Hour, Issuer: "test"})
	service := NewAuthService(users, manager, NewAuditWriter(audits), newMemorySessions())

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

func TestLoginRejectsAccountFromInactiveSchool(t *testing.T) {
	hash, _ := security.HashPassword("correct-password")
	schoolID := uint64(8)
	users := &memoryUsers{user: &model.User{
		ID: 3, SchoolID: &schoolID, Username: "registrar", PasswordHash: hash,
		Status: model.UserStatusActive,
		School: &model.School{ID: schoolID, Status: model.SchoolStatusInactive},
		Role:   model.Role{Name: model.RoleRegistrar},
	}}
	manager := security.NewJWTManager(config.JWTConfig{Secret: "test-secret-long-enough", Expiration: time.Hour, Issuer: "test"})
	service := NewAuthService(users, manager, NewAuditWriter(&memoryAudits{}), newMemorySessions())

	_, err := service.Login(context.Background(), LoginInput{Username: "registrar", Password: "correct-password"}, RequestMetadata{})
	if !errors.Is(err, ErrAccountUnavailable) {
		t.Fatalf("inactive school login error = %v, want ErrAccountUnavailable", err)
	}
}

func TestRefreshRotatesTokenAndPasswordResetIsOneTime(t *testing.T) {
	hash, _ := security.HashPassword("old-password-123")
	users := &memoryUsers{user: &model.User{
		ID: 4, Username: "admin", PasswordHash: hash, Status: model.UserStatusActive,
		Role: model.Role{Name: model.RoleSuperAdmin, Status: model.RoleStatusActive},
	}}
	sessions := newMemorySessions()
	audits := &memoryAudits{}
	manager := security.NewJWTManager(config.JWTConfig{Secret: "test-secret-long-enough", Expiration: time.Hour, Issuer: "test"})
	service := NewAuthService(users, manager, NewAuditWriter(audits), sessions)

	refresh, _, _ := sessions.CreateRefresh(context.Background(), users.user.ID)
	result, err := service.Refresh(context.Background(), RefreshInput{RefreshToken: refresh}, RequestMetadata{})
	if err != nil || result.RefreshToken == refresh {
		t.Fatalf("Refresh() = %#v, %v", result, err)
	}
	if _, _, _, err := sessions.RotateRefresh(context.Background(), refresh); !errors.Is(err, security.ErrInvalidToken) {
		t.Fatalf("reused refresh token error = %v", err)
	}

	reset, _, _ := sessions.CreatePasswordReset(context.Background(), users.user.ID)
	input := ResetPasswordInput{ResetToken: reset, NewPassword: "new-password-123"}
	if err := service.ResetPassword(context.Background(), input, RequestMetadata{}); err != nil {
		t.Fatalf("ResetPassword() error = %v", err)
	}
	if security.CheckPassword(users.user.PasswordHash, input.NewPassword) != nil {
		t.Fatal("new password was not stored")
	}
	if err := service.ResetPassword(context.Background(), input, RequestMetadata{}); !errors.Is(err, security.ErrInvalidToken) {
		t.Fatalf("reused reset token error = %v", err)
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
func (m *memoryUsers) UpdateProfile(_ context.Context, user *model.User) error {
	m.user = user
	return nil
}
func (m *memoryUsers) UpdatePassword(_ context.Context, _ uint64, hash string) error {
	m.user.PasswordHash = hash
	if m.user.Status == model.UserStatusLocked {
		m.user.Status = model.UserStatusActive
	}
	m.user.FailedLogins = 0
	return nil
}
func (m *memoryUsers) UpdateStatus(_ context.Context, _ uint64, status model.UserStatus) error {
	m.user.Status = status
	return nil
}
func (m *memoryUsers) CountSuperAdmins(context.Context) (int64, error) {
	if m.user != nil && m.user.IsSuperAdmin() {
		return 1, nil
	}
	return 0, nil
}

type memoryAudits struct{ entries []model.AuditLog }

func (m *memoryAudits) Create(_ context.Context, entry *model.AuditLog) error {
	m.entries = append(m.entries, *entry)
	return nil
}
func (m *memoryAudits) List(context.Context, *uint64, int, int) ([]model.AuditLog, error) {
	return m.entries, nil
}

type memorySessions struct {
	refresh map[string]uint64
	reset   map[string]uint64
}

func newMemorySessions() *memorySessions {
	return &memorySessions{refresh: make(map[string]uint64), reset: make(map[string]uint64)}
}

func (m *memorySessions) AllowLogin(context.Context, string, string) error { return nil }
func (m *memorySessions) CreateRefresh(_ context.Context, userID uint64) (string, time.Time, error) {
	token := "refresh-token"
	m.refresh[token] = userID
	return token, time.Now().Add(time.Hour), nil
}
func (m *memorySessions) RotateRefresh(_ context.Context, token string) (uint64, string, time.Time, error) {
	userID, ok := m.refresh[token]
	if !ok {
		return 0, "", time.Time{}, security.ErrInvalidToken
	}
	delete(m.refresh, token)
	newToken := token + "-rotated"
	m.refresh[newToken] = userID
	return userID, newToken, time.Now().Add(time.Hour), nil
}
func (m *memorySessions) RevokeRefresh(_ context.Context, token string) error {
	delete(m.refresh, token)
	return nil
}
func (m *memorySessions) DenyAccess(context.Context, string, time.Time) error { return nil }
func (m *memorySessions) AccessDenied(context.Context, uint64, string, time.Time) (bool, error) {
	return false, nil
}
func (m *memorySessions) RevokeAll(_ context.Context, userID uint64) error {
	for token, id := range m.refresh {
		if id == userID {
			delete(m.refresh, token)
		}
	}
	return nil
}
func (m *memorySessions) CreatePasswordReset(_ context.Context, userID uint64) (string, time.Time, error) {
	token := "reset-token"
	m.reset[token] = userID
	return token, time.Now().Add(time.Minute), nil
}
func (m *memorySessions) ConsumePasswordReset(_ context.Context, token string) (uint64, error) {
	userID, ok := m.reset[token]
	if !ok {
		return 0, security.ErrInvalidToken
	}
	delete(m.reset, token)
	return userID, nil
}
