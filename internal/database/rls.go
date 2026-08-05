package database

import (
	"fmt"

	"gorm.io/gorm"
)

func ApplyFoundationRLS(db *gorm.DB) error {
	statements := []string{
		"DO $$ BEGIN CREATE ROLE kobciye_runtime NOLOGIN; EXCEPTION WHEN duplicate_object THEN NULL; END $$",
		"GRANT kobciye_runtime TO CURRENT_USER",
		`CREATE OR REPLACE FUNCTION app_current_school() RETURNS BIGINT AS $$
			SELECT NULLIF(current_setting('app.current_school', true), '')::BIGINT
		$$ LANGUAGE SQL STABLE`,
		`CREATE OR REPLACE FUNCTION app_current_user() RETURNS BIGINT AS $$
			SELECT NULLIF(current_setting('app.current_user', true), '')::BIGINT
		$$ LANGUAGE SQL STABLE`,
		`CREATE OR REPLACE FUNCTION app_is_superadmin() RETURNS BOOLEAN AS $$
			SELECT COALESCE(NULLIF(current_setting('app.is_superadmin', true), '')::BOOLEAN, FALSE)
		$$ LANGUAGE SQL STABLE`,
		`CREATE OR REPLACE FUNCTION app_auth_lookup() RETURNS BOOLEAN AS $$
			SELECT COALESCE(NULLIF(current_setting('app.auth_lookup', true), '')::BOOLEAN, FALSE)
		$$ LANGUAGE SQL STABLE`,
		"GRANT USAGE ON SCHEMA public TO kobciye_runtime",
		"GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO kobciye_runtime",
		"GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO kobciye_runtime",
		"ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO kobciye_runtime",
		"ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO kobciye_runtime",

		"ALTER TABLE schools ENABLE ROW LEVEL SECURITY",
		"DROP POLICY IF EXISTS schools_select ON schools",
		"DROP POLICY IF EXISTS schools_write ON schools",
		"CREATE POLICY schools_select ON schools FOR SELECT TO kobciye_runtime USING (app_auth_lookup() OR app_is_superadmin() OR sch_no = app_current_school())",
		"CREATE POLICY schools_write ON schools FOR ALL TO kobciye_runtime USING (app_is_superadmin()) WITH CHECK (app_is_superadmin())",

		"ALTER TABLE academic_years ENABLE ROW LEVEL SECURITY",
		"DROP POLICY IF EXISTS academic_years_tenant ON academic_years",
		"CREATE POLICY academic_years_tenant ON academic_years FOR ALL TO kobciye_runtime USING (app_is_superadmin() OR sch_no = app_current_school()) WITH CHECK (app_is_superadmin() OR sch_no = app_current_school())",

		"ALTER TABLE users ENABLE ROW LEVEL SECURITY",
		"DROP POLICY IF EXISTS users_tenant ON users",
		"DROP POLICY IF EXISTS users_select ON users",
		"DROP POLICY IF EXISTS users_insert ON users",
		"DROP POLICY IF EXISTS users_update ON users",
		"DROP POLICY IF EXISTS users_delete ON users",
		"CREATE POLICY users_select ON users FOR SELECT TO kobciye_runtime USING (app_auth_lookup() OR app_is_superadmin() OR usr_no = app_current_user() OR sch_no = app_current_school())",
		"CREATE POLICY users_insert ON users FOR INSERT TO kobciye_runtime WITH CHECK (app_is_superadmin() OR sch_no = app_current_school())",
		"CREATE POLICY users_update ON users FOR UPDATE TO kobciye_runtime USING (app_is_superadmin() OR usr_no = app_current_user() OR sch_no = app_current_school()) WITH CHECK (app_is_superadmin() OR usr_no = app_current_user() OR sch_no = app_current_school())",
		"CREATE POLICY users_delete ON users FOR DELETE TO kobciye_runtime USING (app_is_superadmin() OR sch_no = app_current_school())",
		`CREATE OR REPLACE FUNCTION enforce_authentication_user_update() RETURNS trigger AS $$
		BEGIN
			IF app_auth_lookup() AND (
				NEW.sch_no IS DISTINCT FROM OLD.sch_no OR
				NEW.stf_no IS DISTINCT FROM OLD.stf_no OR
				NEW.username IS DISTINCT FROM OLD.username OR
				NEW.role_no IS DISTINCT FROM OLD.role_no OR
				NEW.created_at IS DISTINCT FROM OLD.created_at
			) THEN
				RAISE EXCEPTION 'authentication context cannot change user identity or authorization';
			END IF;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`,
		"DROP TRIGGER IF EXISTS users_authentication_update_guard ON users",
		`CREATE TRIGGER users_authentication_update_guard BEFORE UPDATE ON users
			FOR EACH ROW EXECUTE FUNCTION enforce_authentication_user_update()`,

		"ALTER TABLE audit_logs ENABLE ROW LEVEL SECURITY",
		"DROP POLICY IF EXISTS audit_logs_tenant ON audit_logs",
		"DROP POLICY IF EXISTS audit_logs_select ON audit_logs",
		"DROP POLICY IF EXISTS audit_logs_insert ON audit_logs",
		"CREATE POLICY audit_logs_select ON audit_logs FOR SELECT TO kobciye_runtime USING (app_is_superadmin() OR sch_no = app_current_school())",
		"CREATE POLICY audit_logs_insert ON audit_logs FOR INSERT TO kobciye_runtime WITH CHECK (app_auth_lookup() OR app_is_superadmin() OR sch_no = app_current_school())",

		"ALTER TABLE roles ENABLE ROW LEVEL SECURITY",
		"DROP POLICY IF EXISTS roles_read ON roles",
		"DROP POLICY IF EXISTS roles_write ON roles",
		"CREATE POLICY roles_read ON roles FOR SELECT TO kobciye_runtime USING (TRUE)",
		"CREATE POLICY roles_write ON roles FOR ALL TO kobciye_runtime USING (app_is_superadmin() AND role_name <> 'SuperAdmin') WITH CHECK (app_is_superadmin() AND role_name <> 'SuperAdmin')",

		"ALTER TABLE permissions ENABLE ROW LEVEL SECURITY",
		"DROP POLICY IF EXISTS permissions_read ON permissions",
		"CREATE POLICY permissions_read ON permissions FOR SELECT TO kobciye_runtime USING (TRUE)",

		"ALTER TABLE role_permissions ENABLE ROW LEVEL SECURITY",
		"DROP POLICY IF EXISTS role_permissions_read ON role_permissions",
		"DROP POLICY IF EXISTS role_permissions_write ON role_permissions",
		"CREATE POLICY role_permissions_read ON role_permissions FOR SELECT TO kobciye_runtime USING (TRUE)",
		`CREATE POLICY role_permissions_write ON role_permissions FOR ALL TO kobciye_runtime
			USING (app_is_superadmin() AND NOT EXISTS (SELECT 1 FROM roles WHERE roles.role_no = role_permissions.role_no AND roles.role_name = 'SuperAdmin'))
			WITH CHECK (app_is_superadmin() AND NOT EXISTS (SELECT 1 FROM roles WHERE roles.role_no = role_permissions.role_no AND roles.role_name = 'SuperAdmin'))`,
	}
	for _, statement := range statements {
		if err := db.Exec(statement).Error; err != nil {
			return fmt.Errorf("apply foundation RLS: %w", err)
		}
	}
	return nil
}
