package security

import (
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

	token, err := manager.Generate(42)
	if err != nil {
		t.Fatalf("Generate() error = %v", err)
	}
	userID, err := manager.Parse(token)
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if userID != 42 {
		t.Fatalf("Parse() userID = %d, want 42", userID)
	}
}
