package main

import (
	"backendapi/internal/authz"
	"backendapi/internal/config"
	"backendapi/internal/database"
	"backendapi/internal/model"
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

// closePostgres closes the underlying SQL database connection for the given gorm.DB instance. It logs any error encountered during the close operation.
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
	if err := database.ConfigureFoundationModels(db); err != nil {
		return err
	}

	switch os.Args[1] {
	case "create-superadmin":
		return createSuperAdmin(context.Background(), cfg, db, os.Args[2:])
	case "archive-legacy-users":
		return archiveLegacyUsers(context.Background(), db, os.Args[2:])
	case "verify-foundation":
		return verifyFoundation(context.Background(), db)
	default:
		return usageError()
	}
}

// ApplyFoundationRLS applies row-level security policies to the foundation tables in the database. It creates functions to retrieve the current school and user from the session, and grants appropriate permissions to the kobciye_runtime role.
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
	if err := database.ApplyFoundationRLS(db); err != nil {
		return err
	}
	requestContext, tx, err := database.BeginRequest(ctx, db, database.SecurityScope{Principal: authz.Principal{Role: model.RoleSuperAdmin}})
	if err != nil {
		return err
	}
	users := repository.NewUserRepository(db)
	foundation := repository.NewFoundationRepository(db)
	audit := service.NewAuditWriter(repository.NewAuditRepository(db))
	user, err := service.NewAdminService(users, foundation, audit).
		CreateInitialSuperAdmin(requestContext, *username, string(password))
	if err != nil {
		tx.Rollback()
		return err
	}
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("commit SuperAdmin creation: %w", err)
	}
	fmt.Printf("SuperAdmin %q created with ID %d\n", user.Username, user.ID)
	return nil
}

// ApplyFoundationRLS applies row-level security policies to the foundation tables in the database. It creates functions to retrieve the current school and user from the session, and grants appropriate permissions to the kobciye_runtime role.
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

// verifyFoundation checks that the foundation tables, roles, permissions, and audit trigger are correctly set up in the database. It returns an error if any required table is missing, if the expected roles and permissions are not present, or if the audit trigger is not installed.
func verifyFoundation(ctx context.Context, db *gorm.DB) error {
	requiredTables := []string{"schools", "academic_years", "roles", "permissions", "role_permissions", "users", "audit_logs"}
	for _, table := range requiredTables {
		if !db.Migrator().HasTable(table) {
			return fmt.Errorf("required table %s is missing", table)
		}
	}
	foundation := repository.NewFoundationRepository(db)
	roles, err := foundation.ListRoles(ctx)
	if err != nil {
		return fmt.Errorf("load roles and permissions: %w", err)
	}
	permissions, err := foundation.ListPermissions(ctx)
	if err != nil {
		return fmt.Errorf("load permissions: %w", err)
	}
	permissionCounts := make(map[string]int, len(roles))
	for _, role := range roles {
		permissionCounts[role.Name] = len(role.Permissions)
	}
	if len(roles) < 5 || len(permissions) != 5 || permissionCounts["SuperAdmin"] != 5 || permissionCounts["SchoolAdmin"] != 3 {
		return fmt.Errorf("unexpected access-control seed: roles=%d permissions=%d assignments=%v", len(roles), len(permissions), permissionCounts)
	}
	var triggerCount int64
	err = db.WithContext(ctx).Raw(`SELECT COUNT(*) FROM pg_trigger
		WHERE tgname = 'audit_logs_no_update_or_delete' AND NOT tgisinternal`).Scan(&triggerCount).Error
	if err != nil || triggerCount != 1 {
		return fmt.Errorf("append-only audit trigger is not installed")
	}
	fmt.Printf("Foundation verified: %d tables, %d roles, %d permissions, audit trigger active.\n", len(requiredTables), len(roles), len(permissions))
	return nil
}

// openDatabase opens a connection to the Postgres database using the provided configuration. It returns the gorm.DB instance, a function to close the database connection, and any error encountered during the process.
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

// usageError returns an error indicating the correct usage of the admin command-line tool. It specifies the available commands and their options.
func usageError() error {
	return errors.New("usage: admin <create-superadmin|archive-legacy-users|verify-foundation> [options]")
}
