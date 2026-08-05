package security

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrRateLimited  = errors.New("too many authentication attempts")
	ErrInvalidToken = errors.New("invalid or expired session token")
)

const (
	refreshTTL = 7 * 24 * time.Hour
	resetTTL   = 15 * time.Minute
)

type SessionStore struct{ redis *redis.Client }

type SessionRepository interface {
	AllowLogin(context.Context, string, string) error
	CreateRefresh(context.Context, uint64) (string, time.Time, error)
	RotateRefresh(context.Context, string) (uint64, string, time.Time, error)
	RevokeRefresh(context.Context, string) error
	DenyAccess(context.Context, string, time.Time) error
	AccessDenied(context.Context, uint64, string, time.Time) (bool, error)
	RevokeAll(context.Context, uint64) error
	CreatePasswordReset(context.Context, uint64) (string, time.Time, error)
	ConsumePasswordReset(context.Context, string) (uint64, error)
}

func NewSessionStore(client *redis.Client) *SessionStore { return &SessionStore{redis: client} }

func (s *SessionStore) AllowLogin(ctx context.Context, ipAddress, username string) error {
	checks := []struct {
		key   string
		limit int64
	}{
		{key: "auth:rate:ip:" + hashValue(ipAddress), limit: 20},
		{key: "auth:rate:user:" + hashValue(username), limit: 10},
	}
	for _, check := range checks {
		count, err := s.redis.Incr(ctx, check.key).Result()
		if err != nil {
			return fmt.Errorf("increment authentication rate limit: %w", err)
		}
		if count == 1 {
			if err := s.redis.Expire(ctx, check.key, time.Minute).Err(); err != nil {
				return fmt.Errorf("expire authentication rate limit: %w", err)
			}
		}
		if count > check.limit {
			return ErrRateLimited
		}
	}
	return nil
}

func (s *SessionStore) CreateRefresh(ctx context.Context, userID uint64) (string, time.Time, error) {
	token, err := randomToken()
	if err != nil {
		return "", time.Time{}, err
	}
	hash := hashValue(token)
	expiresAt := time.Now().UTC().Add(refreshTTL)
	userKey := "auth:refresh:user:" + strconv.FormatUint(userID, 10)
	pipe := s.redis.TxPipeline()
	pipe.Set(ctx, "auth:refresh:"+hash, userID, refreshTTL)
	pipe.SAdd(ctx, userKey, hash)
	pipe.Expire(ctx, userKey, refreshTTL)
	if _, err := pipe.Exec(ctx); err != nil {
		return "", time.Time{}, fmt.Errorf("store refresh session: %w", err)
	}
	return token, expiresAt, nil
}

func (s *SessionStore) RotateRefresh(ctx context.Context, token string) (uint64, string, time.Time, error) {
	hash := hashValue(token)
	value, err := s.redis.GetDel(ctx, "auth:refresh:"+hash).Result()
	if errors.Is(err, redis.Nil) {
		return 0, "", time.Time{}, ErrInvalidToken
	}
	if err != nil {
		return 0, "", time.Time{}, fmt.Errorf("consume refresh session: %w", err)
	}
	userID, err := strconv.ParseUint(value, 10, 64)
	if err != nil || userID == 0 {
		return 0, "", time.Time{}, ErrInvalidToken
	}
	_ = s.redis.SRem(ctx, "auth:refresh:user:"+value, hash).Err()
	newToken, expiresAt, err := s.CreateRefresh(ctx, userID)
	return userID, newToken, expiresAt, err
}

func (s *SessionStore) RevokeRefresh(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	hash := hashValue(token)
	value, err := s.redis.GetDel(ctx, "auth:refresh:"+hash).Result()
	if errors.Is(err, redis.Nil) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.redis.SRem(ctx, "auth:refresh:user:"+value, hash).Err()
}

func (s *SessionStore) DenyAccess(ctx context.Context, jti string, expiresAt time.Time) error {
	if jti == "" || !expiresAt.After(time.Now()) {
		return nil
	}
	return s.redis.Set(ctx, "auth:deny:"+jti, "1", time.Until(expiresAt)).Err()
}

func (s *SessionStore) AccessDenied(ctx context.Context, userID uint64, jti string, issuedAt time.Time) (bool, error) {
	denied, err := s.redis.Exists(ctx, "auth:deny:"+jti).Result()
	if err != nil || denied > 0 {
		return denied > 0, err
	}
	revokedAt, err := s.redis.Get(ctx, "auth:revoked-before:"+strconv.FormatUint(userID, 10)).Int64()
	if errors.Is(err, redis.Nil) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return issuedAt.UnixNano() <= revokedAt, nil
}

func (s *SessionStore) RevokeAll(ctx context.Context, userID uint64) error {
	value := strconv.FormatUint(userID, 10)
	userKey := "auth:refresh:user:" + value
	hashes, err := s.redis.SMembers(ctx, userKey).Result()
	if err != nil {
		return err
	}
	pipe := s.redis.TxPipeline()
	for _, hash := range hashes {
		pipe.Del(ctx, "auth:refresh:"+hash)
	}
	pipe.Del(ctx, userKey)
	pipe.Set(ctx, "auth:revoked-before:"+value, time.Now().UTC().UnixNano(), refreshTTL)
	_, err = pipe.Exec(ctx)
	return err
}

func (s *SessionStore) CreatePasswordReset(ctx context.Context, userID uint64) (string, time.Time, error) {
	token, err := randomToken()
	if err != nil {
		return "", time.Time{}, err
	}
	expiresAt := time.Now().UTC().Add(resetTTL)
	if err := s.redis.Set(ctx, "auth:reset:"+hashValue(token), userID, resetTTL).Err(); err != nil {
		return "", time.Time{}, err
	}
	return token, expiresAt, nil
}

func (s *SessionStore) ConsumePasswordReset(ctx context.Context, token string) (uint64, error) {
	value, err := s.redis.GetDel(ctx, "auth:reset:"+hashValue(token)).Result()
	if errors.Is(err, redis.Nil) {
		return 0, ErrInvalidToken
	}
	if err != nil {
		return 0, err
	}
	userID, err := strconv.ParseUint(value, 10, 64)
	if err != nil || userID == 0 {
		return 0, ErrInvalidToken
	}
	return userID, nil
}

func randomToken() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", fmt.Errorf("generate secure token: %w", err)
	}
	return hex.EncodeToString(buffer), nil
}

func hashValue(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
