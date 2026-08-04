package database

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"
)

const legacyUsersArchiveTable = "legacy_users_email_auth"

func HasLegacyUserSchema(db *gorm.DB) (bool, error) {
	if !db.Migrator().HasTable("users") {
		return false, nil
	}
	return db.Migrator().HasColumn("users", "email") && db.Migrator().HasColumn("users", "name"), nil
}

func ArchiveLegacyUsers(ctx context.Context, db *gorm.DB) (int64, error) {
	legacy, err := HasLegacyUserSchema(db)
	if err != nil {
		return 0, err
	}
	if !legacy {
		return 0, errors.New("legacy email-based users table was not found")
	}
	if db.Migrator().HasTable(legacyUsersArchiveTable) {
		return 0, fmt.Errorf("archive table %s already exists; manual review is required", legacyUsersArchiveTable)
	}

	var count int64
	err = db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Table("users").Count(&count).Error; err != nil {
			return fmt.Errorf("count legacy users: %w", err)
		}
		if err := tx.Exec("CREATE TABLE " + legacyUsersArchiveTable + " AS TABLE users WITH DATA").Error; err != nil {
			return fmt.Errorf("archive legacy users: %w", err)
		}
		if err := tx.Exec("DROP TABLE users CASCADE").Error; err != nil {
			return fmt.Errorf("replace legacy users table: %w", err)
		}
		return nil
	})
	return count, err
}
