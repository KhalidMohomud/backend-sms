// Package app wires application dependencies and manages the HTTP server lifecycle.
package app

import (
	"backendapi/internal/config"
	"backendapi/internal/database"
	"backendapi/internal/handler"
	"backendapi/internal/repository"
	"backendapi/internal/router"
	"backendapi/internal/security"
	"backendapi/internal/service"
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type App struct {
	server *http.Server
	db     *gorm.DB
	redis  *redis.Client
}

func New(ctx context.Context, cfg config.Config) (*App, error) {
	db, err := database.NewPostgres(cfg.Postgres, cfg.App.Environment)
	if err != nil {
		return nil, err
	}
	if err := database.ConfigureFoundationModels(db); err != nil {
		closePostgres(db)
		return nil, err
	}
	legacyUsers, err := database.HasLegacyUserSchema(db)
	if err != nil {
		closePostgres(db)
		return nil, fmt.Errorf("inspect users schema: %w", err)
	}
	if legacyUsers {
		closePostgres(db)
		return nil, fmt.Errorf("legacy email-based users table detected; run 'make admin-archive-legacy-users' before starting Phase 1")
	}

	if cfg.App.AutoMigrate {
		if err := database.MigrateFoundation(db); err != nil {
			closePostgres(db)
			return nil, fmt.Errorf("run database migrations: %w", err)
		}
	}
	if err := database.SeedFoundation(ctx, db); err != nil {
		closePostgres(db)
		return nil, err
	}
	if err := database.ApplyFoundationRLS(db); err != nil {
		closePostgres(db)
		return nil, err
	}

	redisClient, err := database.NewRedis(ctx, cfg.Redis)
	if err != nil {
		closePostgres(db)
		return nil, err
	}

	jwtManager := security.NewJWTManager(cfg.JWT)
	sessionStore := security.NewSessionStore(redisClient)
	userRepository := repository.NewUserRepository(db)
	foundationRepository := repository.NewFoundationRepository(db)
	auditRepository := repository.NewAuditRepository(db)
	auditWriter := service.NewAuditWriter(auditRepository)
	authService := service.NewAuthService(userRepository, jwtManager, auditWriter, sessionStore)
	foundationService := service.NewFoundationService(foundationRepository, userRepository, auditRepository, auditWriter)
	if cfg.App.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	engine := router.New(
		handler.NewAuthHandler(authService),
		handler.NewFoundationHandler(foundationService),
		handler.NewHealthHandler(db, redisClient),
		jwtManager,
		userRepository,
		db,
		sessionStore,
	)

	return &App{
		server: &http.Server{
			Addr:              ":" + cfg.App.Port,
			Handler:           engine,
			ReadHeaderTimeout: 5 * time.Second,
		},
		db:    db,
		redis: redisClient,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	serverErr := make(chan error, 1)
	go func() {
		log.Printf("API listening on http://localhost%s", a.server.Addr)
		if err := a.server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		return fmt.Errorf("serve API: %w", err)
	case <-ctx.Done():
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.server.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown API: %w", err)
	}
	return nil
}

func (a *App) Close() error {
	var closeErrors []error
	if a.redis != nil {
		if err := a.redis.Close(); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("close redis: %w", err))
		}
	}
	if err := closePostgres(a.db); err != nil {
		closeErrors = append(closeErrors, err)
	}
	return errors.Join(closeErrors...)
}

func closePostgres(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get postgres connection pool: %w", err)
	}
	if err := sqlDB.Close(); err != nil {
		return fmt.Errorf("close postgres: %w", err)
	}
	return nil
}
