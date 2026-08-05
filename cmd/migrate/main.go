package main

import (
	"backendapi/internal/config"
	"backendapi/internal/database"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type migration struct {
	version  string
	upPath   string
	downPath string
}

func main() {
	if len(os.Args) != 2 || (os.Args[1] != "up" && os.Args[1] != "down" && os.Args[1] != "status") {
		log.Fatal("usage: migrate up|down|status")
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := database.NewPostgres(cfg.Postgres, cfg.App.Environment)
	if err != nil {
		log.Fatal(err)
	}
	db.Logger = logger.Default.LogMode(logger.Silent)
	if err := db.Exec(`CREATE TABLE IF NOT EXISTS schema_migrations (
		version VARCHAR(255) PRIMARY KEY,
		applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
	)`).Error; err != nil {
		log.Fatal(err)
	}
	directory := os.Getenv("MIGRATIONS_DIR")
	if directory == "" {
		directory = "migrations"
		if _, err := os.Stat(directory); err != nil {
			directory = "/migrations"
		}
	}
	migrations, err := loadMigrations(directory)
	if err != nil {
		log.Fatal(err)
	}
	switch os.Args[1] {
	case "up":
		err = migrateUp(db, migrations)
	case "down":
		err = migrateDown(db, migrations)
	case "status":
		err = migrationStatus(db, migrations)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func loadMigrations(directory string) ([]migration, error) {
	paths, err := filepath.Glob(filepath.Join(directory, "*.up.sql"))
	if err != nil {
		return nil, err
	}
	sort.Strings(paths)
	result := make([]migration, 0, len(paths))
	for _, up := range paths {
		name := filepath.Base(up)
		version := strings.TrimSuffix(name, ".up.sql")
		down := filepath.Join(directory, version+".down.sql")
		if _, err := os.Stat(down); err != nil {
			return nil, fmt.Errorf("migration %s has no down file", version)
		}
		result = append(result, migration{version: version, upPath: up, downPath: down})
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("no migrations found in %s", directory)
	}
	return result, nil
}

func migrateUp(db *gorm.DB, migrations []migration) error {
	for _, item := range migrations {
		var count int64
		if err := db.Table("schema_migrations").Where("version = ?", item.version).Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			continue
		}
		if err := applyMigration(db, item.upPath, func(tx *gorm.DB) error {
			return tx.Exec("INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)", item.version, time.Now().UTC()).Error
		}); err != nil {
			return fmt.Errorf("apply %s: %w", item.version, err)
		}
		fmt.Printf("applied %s\n", item.version)
	}
	return nil
}

func migrateDown(db *gorm.DB, migrations []migration) error {
	var version string
	if err := db.Table("schema_migrations").Select("version").Order("version DESC").Limit(1).Scan(&version).Error; err != nil {
		return err
	}
	if version == "" {
		fmt.Println("no applied migrations")
		return nil
	}
	for _, item := range migrations {
		if item.version != version {
			continue
		}
		if err := applyMigration(db, item.downPath, func(tx *gorm.DB) error {
			return tx.Exec("DELETE FROM schema_migrations WHERE version = ?", item.version).Error
		}); err != nil {
			return fmt.Errorf("revert %s: %w", item.version, err)
		}
		fmt.Printf("reverted %s\n", item.version)
		return nil
	}
	return fmt.Errorf("down migration for %s was not found", version)
}

func migrationStatus(db *gorm.DB, migrations []migration) error {
	var applied []string
	if err := db.Table("schema_migrations").Order("version").Pluck("version", &applied).Error; err != nil {
		return err
	}
	set := make(map[string]struct{}, len(applied))
	for _, version := range applied {
		set[version] = struct{}{}
	}
	for _, item := range migrations {
		state := "pending"
		if _, ok := set[item.version]; ok {
			state = "applied"
		}
		fmt.Printf("%-50s %s\n", item.version, state)
	}
	return nil
}

func applyMigration(db *gorm.DB, path string, after func(*gorm.DB) error) error {
	content, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	sql := strings.TrimSpace(string(content))
	sql = strings.TrimSpace(strings.TrimPrefix(sql, "BEGIN;"))
	sql = strings.TrimSpace(strings.TrimSuffix(sql, "COMMIT;"))
	return db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec(sql).Error; err != nil {
			return err
		}
		return after(tx)
	})
}
