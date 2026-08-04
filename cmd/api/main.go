package main

import (
	_ "backendapi/docs"
	"backendapi/internal/app"
	"backendapi/internal/config"
	"context"
	"fmt"
	"log"
	"os/signal"
	"syscall"
)

// @title Kobciye School API
// @version 1.0
// @description Backend API for Kobciye School.
// @BasePath /api/v1
// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	application, err := app.New(ctx, cfg)
	if err != nil {
		return fmt.Errorf("initialize application: %w", err)
	}
	defer func() {
		if err := application.Close(); err != nil {
			log.Printf("close application: %v", err)
		}
	}()

	return application.Run(ctx)
}
