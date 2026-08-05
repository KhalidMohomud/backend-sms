//go:build integration

package security

import (
	"context"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
)

func TestRedisSessionLifecycle(t *testing.T) {
	address := os.Getenv("TEST_REDIS_ADDR")
	if address == "" {
		t.Skip("TEST_REDIS_ADDR is required for the Redis integration test")
	}
	client := redis.NewClient(&redis.Options{Addr: address})
	defer client.Close()
	ctx := context.Background()
	store := NewSessionStore(client)
	userID := uint64(time.Now().UTC().UnixNano())

	refresh, _, err := store.CreateRefresh(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	rotatedUser, rotated, _, err := store.RotateRefresh(ctx, refresh)
	if err != nil || rotatedUser != userID || rotated == refresh {
		t.Fatalf("RotateRefresh() = %d, %q, %v", rotatedUser, rotated, err)
	}
	if _, _, _, err := store.RotateRefresh(ctx, refresh); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("reused refresh error = %v", err)
	}
	if err := store.RevokeAll(ctx, userID); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := store.RotateRefresh(ctx, rotated); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("logout-all refresh error = %v", err)
	}

	reset, _, err := store.CreatePasswordReset(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := store.ConsumePasswordReset(ctx, reset); err != nil || got != userID {
		t.Fatalf("ConsumePasswordReset() = %d, %v", got, err)
	}
	if _, err := store.ConsumePasswordReset(ctx, reset); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("reused reset error = %v", err)
	}

	unique := fmt.Sprintf("%d", userID)
	for attempt := 1; attempt <= 20; attempt++ {
		if err := store.AllowLogin(ctx, "integration-ip-"+unique, "integration-user-"+unique); err != nil {
			if attempt <= 10 {
				t.Fatalf("rate limit attempt %d: %v", attempt, err)
			}
			break
		}
	}
	if err := store.AllowLogin(ctx, "integration-ip-"+unique, "integration-user-"+unique); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("rate limit error = %v, want ErrRateLimited", err)
	}
}
