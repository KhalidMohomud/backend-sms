package database

import (
	"backendapi/internal/model"
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func MigrateFoundation(db *gorm.DB) error {
	if err := ConfigureFoundationModels(db); err != nil {
		return err
	}
	if err := db.AutoMigrate(
		&model.School{},
		&model.AcademicYear{},
		&model.Role{},
		&model.Permission{},
		&model.RolePermission{},
		&model.User{},
		&model.AuditLog{},
	); err != nil {
		return err
	}
	statements := []string{
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_schools_name_ci ON schools (LOWER(sch_name))",
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_schools_email_ci ON schools (LOWER(email)) WHERE email IS NOT NULL AND email <> ''",
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_users_username_ci ON users (LOWER(username))",
		`CREATE OR REPLACE FUNCTION prevent_audit_log_mutation() RETURNS trigger AS $$
		BEGIN
			RAISE EXCEPTION 'audit logs are append-only';
		END;
		$$ LANGUAGE plpgsql`,
		"DROP TRIGGER IF EXISTS audit_logs_no_update_or_delete ON audit_logs",
		`CREATE TRIGGER audit_logs_no_update_or_delete
		BEFORE UPDATE OR DELETE ON audit_logs
		FOR EACH ROW EXECUTE FUNCTION prevent_audit_log_mutation()`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("apply foundation constraint: %w", err)
		}
	}
	return nil
}

func ConfigureFoundationModels(db *gorm.DB) error {
	if err := db.SetupJoinTable(&model.Role{}, "Permissions", &model.RolePermission{}); err != nil {
		return fmt.Errorf("configure role permissions: %w", err)
	}
	return nil
}

func SeedFoundation(ctx context.Context, db *gorm.DB) error {
	permissions := []model.Permission{
		{Name: model.PermissionManageSchools, Description: "Create and manage schools"},
		{Name: model.PermissionManageAcademicYears, Description: "Create and manage academic years"},
		{Name: model.PermissionManageUsers, Description: "Create and manage user accounts"},
		{Name: model.PermissionManageRoles, Description: "View and manage access-control definitions"},
		{Name: model.PermissionViewAuditLogs, Description: "View security and data audit events"},
		{Name: model.PermissionManageLookups, Description: "Create and manage global reference data"},
		{Name: model.PermissionManageStructure, Description: "Create and manage levels and classes"},
	}
	roles := []model.Role{
		{Name: model.RoleSuperAdmin, Description: "Platform administrator with access to all schools", Status: model.RoleStatusActive, IsSystem: true},
		{Name: model.RoleSchoolAdmin, Description: "Administrator restricted to one school", Status: model.RoleStatusActive, IsSystem: true},
		{Name: model.RoleRegistrar, Description: "Student registration and attendance operator", Status: model.RoleStatusActive, IsSystem: true},
		{Name: model.RoleFinance, Description: "School finance operator", Status: model.RoleStatusActive, IsSystem: true},
		{Name: model.RoleTeacher, Description: "Teacher restricted to assigned academic work", Status: model.RoleStatusActive, IsSystem: true},
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "perm_name"}}, DoNothing: true}).Create(&permissions).Error; err != nil {
			return fmt.Errorf("seed permissions: %w", err)
		}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "role_name"}}, DoNothing: true}).Create(&roles).Error; err != nil {
			return fmt.Errorf("seed roles: %w", err)
		}
		if err := tx.Model(&model.Role{}).Where("role_name IN ?", []string{
			model.RoleSuperAdmin, model.RoleSchoolAdmin, model.RoleRegistrar, model.RoleFinance, model.RoleTeacher,
		}).Updates(map[string]any{"is_system": true, "status": model.RoleStatusActive}).Error; err != nil {
			return fmt.Errorf("protect system roles: %w", err)
		}

		var savedPermissions []model.Permission
		var savedRoles []model.Role
		if err := tx.Find(&savedPermissions).Error; err != nil {
			return fmt.Errorf("load permissions: %w", err)
		}
		if err := tx.Find(&savedRoles).Error; err != nil {
			return fmt.Errorf("load roles: %w", err)
		}
		permissionIDs := make(map[string]uint64, len(savedPermissions))
		for _, permission := range savedPermissions {
			permissionIDs[permission.Name] = permission.ID
		}
		roleIDs := make(map[string]uint64, len(savedRoles))
		for _, role := range savedRoles {
			roleIDs[role.Name] = role.ID
		}

		assignments := map[string][]string{
			model.RoleSuperAdmin: {
				model.PermissionManageSchools, model.PermissionManageAcademicYears,
				model.PermissionManageUsers, model.PermissionManageRoles, model.PermissionViewAuditLogs,
				model.PermissionManageLookups, model.PermissionManageStructure,
			},
			model.RoleSchoolAdmin: {
				model.PermissionManageAcademicYears, model.PermissionManageUsers, model.PermissionViewAuditLogs,
				model.PermissionManageStructure,
			},
		}
		for roleName, names := range assignments {
			for _, name := range names {
				row := model.RolePermission{RoleID: roleIDs[roleName], PermissionID: permissionIDs[name]}
				if err := tx.Clauses(clause.OnConflict{
					Columns: []clause.Column{{Name: "role_no"}, {Name: "perm_no"}}, DoNothing: true,
				}).Create(&row).Error; err != nil {
					return fmt.Errorf("seed role permissions: %w", err)
				}
			}
		}
		return nil
	})
}
