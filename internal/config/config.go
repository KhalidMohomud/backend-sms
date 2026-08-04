// Package config contains application configuration.
package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

type Config struct {
	App       AppConfig
	Postgres  PostgresConfig
	Redis     RedisConfig
	JWT       JWTConfig
	Bootstrap BootstrapConfig
}

type AppConfig struct {
	Environment string
	Port        string
	AutoMigrate bool
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

type BootstrapConfig struct {
	SuperAdminUsername string
	SuperAdminPassword string
}

func Load() (Config, error) {
	expiration, err := time.ParseDuration(getEnv("JWT_EXPIRATION", "24h"))
	if err != nil {
		return Config{}, fmt.Errorf("parse JWT_EXPIRATION: %w", err)
	}

	redisDB, err := strconv.Atoi(getEnv("REDIS_DB", "0"))
	if err != nil {
		return Config{}, fmt.Errorf("parse REDIS_DB: %w", err)
	}

	cfg := Config{
		App: AppConfig{
			Environment: getEnv("APP_ENV", "development"),
			Port:        getEnv("APP_PORT", "8080"),
			AutoMigrate: getEnv("AUTO_MIGRATE", "true") == "true",
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
		Bootstrap: BootstrapConfig{
			SuperAdminUsername: strings.TrimSpace(os.Getenv("BOOTSTRAP_SUPERADMIN_USERNAME")),
			SuperAdminPassword: os.Getenv("BOOTSTRAP_SUPERADMIN_PASSWORD"),
		},
	}

	if cfg.App.Environment == "production" && cfg.JWT.Secret == "local-development-secret-change-me" {
		return Config{}, fmt.Errorf("JWT_SECRET must be set in production")
	}
	if (cfg.Bootstrap.SuperAdminUsername == "") != (cfg.Bootstrap.SuperAdminPassword == "") {
		return Config{}, fmt.Errorf("both BOOTSTRAP_SUPERADMIN_USERNAME and BOOTSTRAP_SUPERADMIN_PASSWORD must be set together")
	}
	if cfg.Bootstrap.SuperAdminPassword != "" && len(cfg.Bootstrap.SuperAdminPassword) < 12 {
		return Config{}, fmt.Errorf("BOOTSTRAP_SUPERADMIN_PASSWORD must contain at least 12 characters")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}
