package main

import (
	"backendapi/internal/config"
	"backendapi/internal/database"
	"backendapi/internal/repository"
	"backendapi/internal/service"
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"golang.org/x/term"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if len(os.Args) < 2 || os.Args[1] != "create-superadmin" {
		return errors.New("usage: admin create-superadmin --username <username>")
	}
	flags := flag.NewFlagSet("create-superadmin", flag.ContinueOnError)
	username := flags.String("username", "", "SuperAdmin username")
	if err := flags.Parse(os.Args[2:]); err != nil {
		return err
	}
	if strings.TrimSpace(*username) == "" {
		return errors.New("--username is required")
	}

	fmt.Print("SuperAdmin password (12-72 characters): ")
	password, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if err != nil {
		return fmt.Errorf("read password: %w", err)
	}

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	db, err := database.NewPostgres(cfg.Postgres, cfg.App.Environment)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get postgres connection: %w", err)
	}
	defer sqlDB.Close()

	ctx := context.Background()
	if cfg.App.AutoMigrate {
		if err := database.MigrateFoundation(db); err != nil {
			return fmt.Errorf("migrate foundation: %w", err)
		}
	}
	if err := database.SeedFoundation(ctx, db); err != nil {
		return err
	}
	users := repository.NewUserRepository(db)
	foundation := repository.NewFoundationRepository(db)
	audit := service.NewAuditWriter(repository.NewAuditRepository(db))
	admin := service.NewAdminService(users, foundation, audit)
	user, err := admin.CreateInitialSuperAdmin(ctx, *username, string(password))
	if err != nil {
		return err
	}
	fmt.Printf("SuperAdmin %q created with ID %d\n", user.Username, user.ID)
	return nil
}
