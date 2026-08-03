package main

import (
	_ "backendapi/docs"
	"backendapi/internal/config"
	"backendapi/internal/database"
	"backendapi/internal/handler"
	"backendapi/internal/model"
	"backendapi/internal/repository"
	"backendapi/internal/router"
	"backendapi/internal/security"
	"backendapi/internal/service"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

// @title Kobciye School API
// @version 1.0
// @description Backend API for Kobciye School.
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("load configuration: %v", err)
	}

	ctx := context.Background()
	db, err := database.NewPostgres(cfg.Postgres, cfg.App.Environment)
	if err != nil {
		log.Fatal(err)
	}
	if cfg.App.AutoMigrate {
		if err := db.AutoMigrate(&model.User{}); err != nil {
			log.Fatalf("run database migrations: %v", err)
		}
	}
	redisClient, err := database.NewRedis(ctx, cfg.Redis)
	if err != nil {
		log.Fatal(err)
	}
	defer redisClient.Close()

	jwtManager := security.NewJWTManager(cfg.JWT)
	userRepository := repository.NewUserRepository(db)
	authService := service.NewAuthService(userRepository, jwtManager)
	engine := router.New(handler.NewAuthHandler(authService), handler.NewHealthHandler(db, redisClient))

	server := &http.Server{
		Addr:              ":" + cfg.App.Port,
		Handler:           engine,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("API listening on http://localhost:%s", cfg.App.Port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("serve API: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("server shutdown: %v", err)
	}
}
