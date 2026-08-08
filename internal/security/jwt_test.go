package security

import (
	"backendapi/internal/authz"
	"backendapi/internal/config"
	"testing"
	"time"
)

func TestJWTGenerateAndParse(t *testing.T) {
	manager := NewJWTManager(config.JWTConfig{
		Secret:     "a-secret-long-enough-for-unit-tests",
		Expiration: time.Hour,
		Issuer:     "test",
	})

	schoolID := uint64(7)
	staffID := uint64(12)
	want := authz.Principal{UserID: 42, SchoolID: &schoolID, StaffID: &staffID, Role: "SchoolAdmin", Permissions: []string{"manage_users"}}
	token, _, err := manager.Generate(want)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	got, err := manager.Parse(token)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got.UserID != want.UserID || got.SchoolID == nil || *got.SchoolID != schoolID || got.StaffID == nil || *got.StaffID != staffID || got.Role != want.Role {
		t.Fatalf("Parse() principal = %#v, want %#v", got, want)
	}
}
