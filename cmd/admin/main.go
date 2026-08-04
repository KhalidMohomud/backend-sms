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
	"gorm.io/gorm"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if len(os.Args) < 2 {
		return usageError()
	}
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}
	db, closeDB, err := openDatabase(cfg)
	if err != nil {
		return err
	}
	defer closeDB()

	switch os.Args[1] {
	case "create-superadmin":
		return createSuperAdmin(context.Background(), cfg, db, os.Args[2:])
	case "archive-legacy-users":
		return archiveLegacyUsers(context.Background(), db, os.Args[2:])
	default:
		return usageError()
	}
}

func createSuperAdmin(ctx context.Context, cfg config.Config, db *gorm.DB, args []string) error {
	flags := flag.NewFlagSet("create-superadmin", flag.ContinueOnError)
	username := flags.String("username", "", "SuperAdmin username")
	if err := flags.Parse(args); err != nil {
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
	legacy, err := database.HasLegacyUserSchema(db)
	if err != nil {
		return err
	}
	if legacy {
		return errors.New("legacy users must be archived first with 'make admin-archive-legacy-users'")
	}
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
	user, err := service.NewAdminService(users, foundation, audit).
		CreateInitialSuperAdmin(ctx, *username, string(password))
	if err != nil {
		return err
	}
	fmt.Printf("SuperAdmin %q created with ID %d\n", user.Username, user.ID)
	return nil
}

func archiveLegacyUsers(ctx context.Context, db *gorm.DB, args []string) error {
	flags := flag.NewFlagSet("archive-legacy-users", flag.ContinueOnError)
	confirmed := flags.Bool("confirm-archive", false, "confirm archive and table replacement")
	if err := flags.Parse(args); err != nil {
		return err
	}
	if !*confirmed {
		return errors.New("refusing to change the legacy table without --confirm-archive")
	}
	count, err := database.ArchiveLegacyUsers(ctx, db)
	if err != nil {
		return err
	}
	fmt.Printf("Archived %d legacy user row(s) in legacy_users_email_auth. The Phase 1 users table can now be created.\n", count)
	return nil
}

func openDatabase(cfg config.Config) (*gorm.DB, func(), error) {
	db, err := database.NewPostgres(cfg.Postgres, cfg.App.Environment)
	if err != nil {
		return nil, nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, nil, fmt.Errorf("get postgres connection: %w", err)
	}
	return db, func() { _ = sqlDB.Close() }, nil
}

func usageError() error {
	return errors.New("usage: admin <create-superadmin|archive-legacy-users> [options]")
}
