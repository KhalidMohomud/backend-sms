package database

import (
	"backendapi/internal/model"
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func MigrateFoundation(db *gorm.DB) error {
	return db.AutoMigrate(
		&model.School{},
		&model.AcademicYear{},
		&model.Role{},
		&model.Permission{},
		&model.RolePermission{},
		&model.User{},
		&model.AuditLog{},
	)
}

func SeedFoundation(ctx context.Context, db *gorm.DB) error {
	permissions := []model.Permission{
		{Name: model.PermissionManageSchools, Description: "Create and manage schools"},
		{Name: model.PermissionManageAcademicYears, Description: "Create and manage academic years"},
		{Name: model.PermissionManageUsers, Description: "Create and manage user accounts"},
		{Name: model.PermissionManageRoles, Description: "View and manage access-control definitions"},
		{Name: model.PermissionViewAuditLogs, Description: "View security and data audit events"},
	}
	roles := []model.Role{
		{Name: model.RoleSuperAdmin, Description: "Platform administrator with access to all schools"},
		{Name: model.RoleSchoolAdmin, Description: "Administrator restricted to one school"},
		{Name: model.RoleRegistrar, Description: "Student registration and attendance operator"},
		{Name: model.RoleFinance, Description: "School finance operator"},
		{Name: model.RoleTeacher, Description: "Teacher restricted to assigned academic work"},
	}

	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "perm_name"}}, DoNothing: true}).Create(&permissions).Error; err != nil {
			return fmt.Errorf("seed permissions: %w", err)
		}
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "role_name"}}, DoNothing: true}).Create(&roles).Error; err != nil {
			return fmt.Errorf("seed roles: %w", err)
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
			},
			model.RoleSchoolAdmin: {
				model.PermissionManageAcademicYears, model.PermissionManageUsers, model.PermissionViewAuditLogs,
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
