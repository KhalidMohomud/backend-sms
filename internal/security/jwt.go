package security

import (
	"backendapi/internal/authz"
	"backendapi/internal/config"
	"fmt"
	"strconv"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type TokenClaims struct {
	SchoolID    *uint64  `json:"school_id,omitempty"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions,omitempty"`
	jwt.RegisteredClaims
}

type JWTManager struct {
	secret     []byte
	expiration time.Duration
	issuer     string
}

func NewJWTManager(cfg config.JWTConfig) *JWTManager {
	return &JWTManager{secret: []byte(cfg.Secret), expiration: cfg.Expiration, issuer: cfg.Issuer}
}

func (m *JWTManager) Generate(principal authz.Principal) (string, time.Time, error) {
	now := time.Now().UTC()
	expiresAt := now.Add(m.expiration)
	claims := TokenClaims{
		SchoolID:    principal.SchoolID,
		Role:        principal.Role,
		Permissions: principal.Permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   strconv.FormatUint(principal.UserID, 10),
			Issuer:    m.issuer,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
	}
	token, err := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(m.secret)
	return token, expiresAt, err
}

func (m *JWTManager) Parse(tokenString string) (authz.Principal, error) {
	token, err := jwt.ParseWithClaims(tokenString, &TokenClaims{}, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, fmt.Errorf("unexpected signing method: %s", token.Method.Alg())
		}
		return m.secret, nil
	}, jwt.WithIssuer(m.issuer), jwt.WithExpirationRequired(), jwt.WithIssuedAt())
	if err != nil || !token.Valid {
		return authz.Principal{}, fmt.Errorf("invalid token")
	}

	claims, ok := token.Claims.(*TokenClaims)
	if !ok {
		return authz.Principal{}, fmt.Errorf("invalid token claims")
	}
	userID, err := strconv.ParseUint(claims.Subject, 10, 64)
	if err != nil || userID == 0 {
		return authz.Principal{}, fmt.Errorf("invalid token subject")
	}
	return authz.Principal{
		UserID:      userID,
		SchoolID:    claims.SchoolID,
		Role:        claims.Role,
		Permissions: claims.Permissions,
	}, nil
}
