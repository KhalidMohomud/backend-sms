// Package config contains application configuration.
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	App      AppConfig
	Postgres PostgresConfig
	Redis    RedisConfig
	JWT      JWTConfig
}

type AppConfig struct {
	Environment    string
	Port           string
	AutoMigrate    bool
	AllowedOrigins []string
	MaxBodyBytes   int64
}

type PostgresConfig struct {
	Host     string
	Port     string
	User     string
	Password string
	Database string
	SSLMode  string
}

func (c PostgresConfig) DSN() string {
	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s TimeZone=UTC",
		c.Host, c.Port, c.User, c.Password, c.Database, c.SSLMode)
}

type RedisConfig struct {
	Address  string
	Password string
	DB       int
}

type JWTConfig struct {
	Secret     string
	Expiration time.Duration
	Issuer     string
}

func Load() (Config, error) {
	expiration, err := time.ParseDuration(getEnv("JWT_EXPIRATION", "15m"))
	if err != nil {
		return Config{}, fmt.Errorf("parse JWT_EXPIRATION: %w", err)
	}

	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return Config{}, fmt.Errorf("parse REDIS_DB: %w", err)
	}
	maxBodyBytes, err := strconv.ParseInt(getEnv("MAX_BODY_BYTES", "1048576"), 10, 64)
	if err != nil || maxBodyBytes < 1024 {
		return Config{}, fmt.Errorf("MAX_BODY_BYTES must be an integer of at least 1024")
	}

	cfg := Config{
		App: AppConfig{
			Environment:    getEnv("APP_ENV", "development"),
			Port:           getEnv("APP_PORT", "8080"),
			AutoMigrate:    getEnv("AUTO_MIGRATE", "true") == "true",
			AllowedOrigins: splitCSV(getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:3000,http://localhost:8081")),
			MaxBodyBytes:   maxBodyBytes,
		},
		Postgres: PostgresConfig{
			Host:     getEnv("POSTGRES_HOST", "localhost"),
			Port:     getEnv("POSTGRES_PORT", "5432"),
			User:     getEnv("POSTGRES_USER", "kobciye"),
			Password: getEnv("POSTGRES_PASSWORD", "kobciye"),
			Database: getEnv("POSTGRES_DB", "kobciye"),
			SSLMode:  getEnv("POSTGRES_SSLMODE", "disable"),
		},
		Redis: RedisConfig{
			Address:  getEnv("REDIS_ADDR", "localhost:6379"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       redisDB,
		},
		JWT: JWTConfig{
			Secret:     getEnv("JWT_SECRET", "local-development-secret-change-me"),
			Expiration: expiration,
			Issuer:     getEnv("JWT_ISSUER", "kobciye-api"),
		},
	}

	if cfg.App.Environment == "production" && cfg.JWT.Secret == "local-development-secret-change-me" {
		return Config{}, fmt.Errorf("JWT_SECRET must be set in production")
	}
	if cfg.App.Environment == "production" && len(cfg.JWT.Secret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must contain at least 32 bytes in production")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}

func splitCSV(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			result = append(result, part)
		}
	}
	return result
}
