package database

import (
	"backendapi/internal/model"
	"context"
	"fmt"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MigrateAcademic maintains the Phase 3 academic schema in local development.
// Production deployments use migrations/000007_phase3_academic.up.sql instead.
func MigrateAcademic(db *gorm.DB) error {
	for _, statement := range []string{
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_academic_years_id_school ON academic_years(y_no, sch_no)",
		"CREATE UNIQUE INDEX IF NOT EXISTS uq_classes_id_school ON classes(cl_no, sch_no)",
	} {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("prepare academic composite key: %w", err)
		}
	}
	if err := db.AutoMigrate(
		&model.StudentClass{}, &model.SubjectClass{}, &model.ExamRegistration{}, &model.ExamResult{},
	); err != nil {
		return fmt.Errorf("migrate academic models: %w", err)
	}
	statements := []string{
		"CREATE INDEX IF NOT EXISTS ix_student_classes_class_year ON student_classes(cl_no, y_no)",
		"CREATE INDEX IF NOT EXISTS ix_subject_classes_class_year ON subject_classes(cl_no, y_no)",
		"CREATE INDEX IF NOT EXISTS ix_subject_classes_staff_year ON subject_classes(stf_no, y_no)",
		"CREATE INDEX IF NOT EXISTS ix_exam_registrations_year_dates ON exam_registrations(y_no, started, ended)",
		"CREATE INDEX IF NOT EXISTS ix_exam_results_registration ON exam_results(ex_r_no)",
		"CREATE INDEX IF NOT EXISTS ix_exam_results_enrollment ON exam_results(sc_no)",
		`CREATE OR REPLACE FUNCTION validate_exam_registration_scope() RETURNS TRIGGER
		LANGUAGE plpgsql AS $$
		DECLARE year_started DATE; year_ended DATE;
		BEGIN
			SELECT started, ended INTO STRICT year_started, year_ended
			FROM academic_years WHERE y_no = NEW.y_no AND sch_no = NEW.sch_no;
			IF NEW.started < year_started OR NEW.ended > year_ended THEN
				RAISE EXCEPTION 'exam dates must be inside the academic year' USING ERRCODE = '23514';
			END IF;
			RETURN NEW;
		END; $$`,
		"DROP TRIGGER IF EXISTS exam_registrations_validate_scope ON exam_registrations",
		`CREATE TRIGGER exam_registrations_validate_scope BEFORE INSERT OR UPDATE ON exam_registrations
		FOR EACH ROW EXECUTE FUNCTION validate_exam_registration_scope()`,
		`CREATE OR REPLACE FUNCTION validate_exam_result_scope() RETURNS TRIGGER
		LANGUAGE plpgsql AS $$
		DECLARE
			exam_year BIGINT; enrollment_year BIGINT; enrollment_class BIGINT;
			assignment_year BIGINT; assignment_class BIGINT; assignment_staff BIGINT;
			allowed_mark NUMERIC(5,2); recorder_role VARCHAR(40); recorder_staff BIGINT;
		BEGIN
			SELECT y_no INTO STRICT exam_year FROM exam_registrations WHERE ex_r_no = NEW.ex_r_no AND sch_no = NEW.sch_no;
			SELECT y_no, cl_no INTO STRICT enrollment_year, enrollment_class FROM student_classes WHERE sc_no = NEW.sc_no AND sch_no = NEW.sch_no;
			SELECT y_no, cl_no, stf_no, max_mark INTO STRICT assignment_year, assignment_class, assignment_staff, allowed_mark
			FROM subject_classes WHERE sb_cl_no = NEW.sb_cl_no AND sch_no = NEW.sch_no;
			IF exam_year <> enrollment_year OR exam_year <> assignment_year OR enrollment_class <> assignment_class THEN
				RAISE EXCEPTION 'exam, enrollment, and subject assignment must share the same class and academic year' USING ERRCODE = '23514';
			END IF;
			IF NEW.marks > allowed_mark THEN
				RAISE EXCEPTION 'marks exceed the assigned maximum mark' USING ERRCODE = '23514';
			END IF;
			SELECT r.role_name, u.stf_no INTO STRICT recorder_role, recorder_staff
			FROM users u JOIN roles r ON r.role_no = u.role_no WHERE u.usr_no = NEW.recorded_by;
			IF recorder_role = 'Teacher' AND (recorder_staff IS NULL OR recorder_staff <> assignment_staff) THEN
				RAISE EXCEPTION 'teacher is not assigned to this subject class' USING ERRCODE = '42501';
			END IF;
			RETURN NEW;
		END; $$`,
		"DROP TRIGGER IF EXISTS exam_results_validate_scope ON exam_results",
		`CREATE TRIGGER exam_results_validate_scope BEFORE INSERT OR UPDATE ON exam_results
		FOR EACH ROW EXECUTE FUNCTION validate_exam_result_scope()`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("apply academic constraint: %w", err)
		}
	}
	return nil
}

func SeedAcademic(ctx context.Context, db *gorm.DB) error {
	permissions := []model.Permission{
		{Name: model.PermissionManageEnrollments, Description: "Enroll students into classes for academic years"},
		{Name: model.PermissionManageAssignments, Description: "Assign subjects and teachers to classes"},
		{Name: model.PermissionManageExams, Description: "Schedule and manage school exams"},
		{Name: model.PermissionEnterMarks, Description: "Create, correct, and remove exam marks"},
		{Name: model.PermissionViewResults, Description: "View academic enrollments, assignments, exams, and results"},
	}
	return db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "perm_name"}}, DoNothing: true}).Create(&permissions).Error; err != nil {
			return fmt.Errorf("seed academic permissions: %w", err)
		}
		assignments := map[string][]string{
			model.RoleSuperAdmin:  {model.PermissionManageEnrollments, model.PermissionManageAssignments, model.PermissionManageExams, model.PermissionEnterMarks, model.PermissionViewResults},
			model.RoleSchoolAdmin: {model.PermissionManageEnrollments, model.PermissionManageAssignments, model.PermissionManageExams, model.PermissionEnterMarks, model.PermissionViewResults},
			model.RoleRegistrar:   {model.PermissionManageEnrollments, model.PermissionViewResults},
			model.RoleTeacher:     {model.PermissionEnterMarks, model.PermissionViewResults},
		}
		for roleName, permissionNames := range assignments {
			for _, permissionName := range permissionNames {
				if err := tx.Exec(`INSERT INTO role_permissions (role_no, perm_no, created_at)
					SELECT r.role_no, p.perm_no, NOW() FROM roles r CROSS JOIN permissions p
					WHERE r.role_name = ? AND p.perm_name = ?
					ON CONFLICT (role_no, perm_no) DO NOTHING`, roleName, permissionName).Error; err != nil {
					return fmt.Errorf("assign academic permission: %w", err)
				}
			}
		}
		return nil
	})
}

func ApplyAcademicRLS(db *gorm.DB) error {
	statements := []string{
		"GRANT SELECT, INSERT, UPDATE, DELETE ON student_classes, subject_classes, exam_registrations, exam_results TO kobciye_runtime",
		"GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO kobciye_runtime",
	}
	for _, table := range []string{"student_classes", "subject_classes", "exam_registrations", "exam_results"} {
		statements = append(statements,
			fmt.Sprintf("ALTER TABLE %s ENABLE ROW LEVEL SECURITY", table),
			fmt.Sprintf("DROP POLICY IF EXISTS %s_tenant ON %s", table, table),
			fmt.Sprintf("CREATE POLICY %s_tenant ON %s FOR ALL TO kobciye_runtime USING (app_is_superadmin() OR sch_no = app_current_school()) WITH CHECK (app_is_superadmin() OR sch_no = app_current_school())", table, table),
		)
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("apply academic RLS: %w", err)
		}
	}
	return nil
}
