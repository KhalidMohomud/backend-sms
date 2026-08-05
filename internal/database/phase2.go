package database

import (
	"backendapi/internal/model"
	"context"
	"fmt"

	"gorm.io/gorm"
)

type lookupSeed struct {
	table      string
	nameColumn string
	names      []string
}

var phase2LookupSeeds = []lookupSeed{
	{table: "jobs", nameColumn: "job_name", names: []string{"Teacher", "Registrar", "Accountant", "Administrator"}},
	{table: "decrees", nameColumn: "dec_name", names: []string{"Permanent", "Contract", "Temporary"}},
	{table: "subjects", nameColumn: "sub_name", names: []string{"Mathematics", "English", "Somali", "Science"}},
	{table: "exams", nameColumn: "ex_name", names: []string{"Monthly", "Midterm", "Final"}},
	{table: "periods", nameColumn: "per_name", names: []string{"Period 1", "Period 2", "Period 3", "Period 4"}},
	{table: "attendance_status", nameColumn: "ast_name", names: []string{"Present", "Absent", "Late", "Excused"}},
	{table: "att_conditions", nameColumn: "con_name", names: []string{"Check-In", "Check-Out"}},
	{table: "staff_status_types", nameColumn: "sst_name", names: []string{"Active", "Suspended", "Resigned", "Retired"}},
	{table: "amount_types", nameColumn: "am_name", names: []string{"Tuition", "Registration", "Exam Fee"}},
	{table: "expense_types", nameColumn: "exp_name", names: []string{"Rent", "Electricity", "Maintenance"}},
}

func MigratePhase2(db *gorm.DB) error {
	if err := db.AutoMigrate(
		&model.Job{}, &model.Decree{}, &model.Subject{}, &model.Exam{}, &model.Period{},
		&model.AttendanceStatus{}, &model.AttendanceCondition{}, &model.StaffStatusType{},
		&model.AmountType{}, &model.ExpenseType{}, &model.Level{}, &model.Class{},
	); err != nil {
		return fmt.Errorf("migrate Phase 2 models: %w", err)
	}
	indexes := []string{
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_jobs_name_ci ON jobs (LOWER(job_name))",
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_decrees_name_ci ON decrees (LOWER(dec_name))",
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_subjects_name_ci ON subjects (LOWER(sub_name))",
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_exams_name_ci ON exams (LOWER(ex_name))",
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_periods_name_ci ON periods (LOWER(per_name))",
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_attendance_status_name_ci ON attendance_status (LOWER(ast_name))",
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_att_conditions_name_ci ON att_conditions (LOWER(con_name))",
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_staff_status_types_name_ci ON staff_status_types (LOWER(sst_name))",
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_amount_types_name_ci ON amount_types (LOWER(am_name))",
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_expense_types_name_ci ON expense_types (LOWER(exp_name))",
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_levels_school_name_ci ON levels (sch_no, LOWER(lev_name))",
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_classes_school_name_ci ON classes (sch_no, LOWER(cl_name))",
	}
	for _, statement := range indexes {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("apply Phase 2 index: %w", err)
		}
	}
	return nil
}

func SeedPhase2(ctx context.Context, db *gorm.DB) error {
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, seed := range phase2LookupSeeds {
			for _, name := range seed.names {
				statement := fmt.Sprintf(
					"INSERT INTO %s (%s, status) SELECT ?, 'active' WHERE NOT EXISTS (SELECT 1 FROM %s WHERE LOWER(%s) = LOWER(?))",
					seed.table, seed.nameColumn, seed.table, seed.nameColumn,
				)
				if err := tx.Exec(statement, name, name).Error; err != nil {
					return fmt.Errorf("seed %s: %w", seed.table, err)
				}
			}
		}
		return nil
	})
}

func ApplyPhase2RLS(db *gorm.DB) error {
	statements := []string{
		"GRANT SELECT, INSERT, UPDATE, DELETE ON jobs, decrees, subjects, exams, periods, attendance_status, att_conditions, staff_status_types, amount_types, expense_types, levels, classes TO kobciye_runtime",
		"GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO kobciye_runtime",
		"ALTER TABLE levels ENABLE ROW LEVEL SECURITY",
		"DROP POLICY IF EXISTS levels_tenant ON levels",
		"CREATE POLICY levels_tenant ON levels FOR ALL TO kobciye_runtime USING (app_is_superadmin() OR sch_no = app_current_school()) WITH CHECK (app_is_superadmin() OR sch_no = app_current_school())",
		"ALTER TABLE classes ENABLE ROW LEVEL SECURITY",
		"DROP POLICY IF EXISTS classes_tenant ON classes",
		"CREATE POLICY classes_tenant ON classes FOR ALL TO kobciye_runtime USING (app_is_superadmin() OR sch_no = app_current_school()) WITH CHECK (app_is_superadmin() OR sch_no = app_current_school())",
	}
	for _, lookup := range phase2LookupSeeds {
		statements = append(statements,
			fmt.Sprintf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY", lookup.table),
			fmt.Sprintf("DROP POLICY IF EXISTS %s_read ON %s", lookup.table, lookup.table),
			fmt.Sprintf("DROP POLICY IF EXISTS %s_write ON %s", lookup.table, lookup.table),
			fmt.Sprintf("CREATE POLICY %s_read ON %s FOR SELECT TO kobciye_runtime USING (TRUE)", lookup.table, lookup.table),
			fmt.Sprintf("CREATE POLICY %s_write ON %s FOR ALL TO kobciye_runtime USING (app_is_superadmin()) WITH CHECK (app_is_superadmin())", lookup.table, lookup.table),
		)
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("apply Phase 2 RLS: %w", err)
		}
	}
	return nil
}
