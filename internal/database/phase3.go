package database

import (
	"backendapi/internal/model"
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

func MigratePhase3(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.Address{}, &model.Responsible{}, &model.Student{}, &model.Staff{}, &model.StaffStatus{},
	); err != nil {
		return fmt.Errorf("migrate Phase 3 models: %w", err)
	}
	statements := []string{
		"CREATE INDEX IF NOT EXISTS ix_responsibles_phone ON responsibles (res_tell)",
		"CREATE INDEX IF NOT EXISTS ix_students_school_status ON students (sch_no, status)",
		"CREATE INDEX IF NOT EXISTS ix_students_name ON students (std_name)",
		"CREATE INDEX IF NOT EXISTS ix_staff_name ON staff (stf_name)",
		"CREATE INDEX IF NOT EXISTS ix_staff_status_staff_date ON staff_status (stf_no, st_date DESC, ss_no DESC)",
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_staff_status_event ON staff_status (stf_no, sst_no, st_date)",
		`DO $$ BEGIN
			ALTER TABLE users ADD CONSTRAINT fk_users_staff_school FOREIGN KEY (stf_no, sch_no)
			REFERENCES staff(stf_no, sch_no) ON DELETE RESTRICT;
		EXCEPTION WHEN duplicate_object THEN NULL; END $$`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("apply Phase 3 constraint: %w", err)
		}
	}
	return nil
}

func SeedPhase3(ctx context.Context, db *gorm.DB) error {
	permissions := []model.Permission{
		{Name: model.PermissionManageStudents, Description: "Create and manage student records, guardians, and addresses"},
		{Name: model.PermissionManageStaff, Description: "Create and manage staff records and employment status history"},
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "perm_name"}}, DoNothing: true}).Create(&permissions).Error; err != nil {
			return fmt.Errorf("seed Phase 3 permissions: %w", err)
		}
		assignments := map[string][]string{
			model.RoleSuperAdmin:  {model.PermissionManageStudents, model.PermissionManageStaff},
			model.RoleSchoolAdmin: {model.PermissionManageStudents, model.PermissionManageStaff},
			model.RoleRegistrar:   {model.PermissionManageStudents},
		}
		for roleName, permissionNames := range assignments {
			for _, permissionName := range permissionNames {
				if err := tx.Exec(`INSERT INTO role_permissions (role_no, perm_no, created_at)
					SELECT r.role_no, p.perm_no, NOW() FROM roles r CROSS JOIN permissions p
					WHERE r.role_name = ? AND p.perm_name = ?
					ON CONFLICT (role_no, perm_no) DO NOTHING`, roleName, permissionName).Error; err != nil {
					return fmt.Errorf("assign Phase 3 permission: %w", err)
				}
			}
		}
		return nil
	})
}

func ApplyPhase3RLS(db *gorm.DB) error {
	statements := []string{
		"GRANT SELECT, INSERT, UPDATE, DELETE ON addresses, responsibles, students, staff, staff_status TO kobciye_runtime",
		"GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO kobciye_runtime",
	}
	for _, table := range []string{"addresses", "responsibles", "students", "staff", "staff_status"} {
		statements = append(statements,
			fmt.Sprintf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY", table),
			fmt.Sprintf("DROP POLICY IF EXISTS %s_tenant ON %s", table, table),
			fmt.Sprintf("CREATE POLICY %s_tenant ON %s FOR ALL TO kobciye_runtime USING (app_is_superadmin() OR sch_no = app_current_school()) WITH CHECK (app_is_superadmin() OR sch_no = app_current_school())", table, table),
		)
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("apply Phase 3 RLS: %w", err)
		}
	}
	return nil
}
