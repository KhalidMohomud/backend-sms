BEGIN;

DROP TRIGGER IF EXISTS users_authentication_update_guard ON users;
DROP FUNCTION IF EXISTS enforce_authentication_user_update();

DROP POLICY IF EXISTS users_select ON users;
DROP POLICY IF EXISTS users_insert ON users;
DROP POLICY IF EXISTS users_update ON users;
DROP POLICY IF EXISTS users_delete ON users;
CREATE POLICY users_tenant ON users FOR ALL TO kobciye_runtime
USING (app_auth_lookup() OR app_is_superadmin() OR usr_no = app_current_user() OR sch_no = app_current_school())
WITH CHECK (app_auth_lookup() OR app_is_superadmin() OR usr_no = app_current_user() OR sch_no = app_current_school());

DROP POLICY IF EXISTS audit_logs_select ON audit_logs;
DROP POLICY IF EXISTS audit_logs_insert ON audit_logs;
CREATE POLICY audit_logs_tenant ON audit_logs FOR ALL TO kobciye_runtime
USING (app_auth_lookup() OR app_is_superadmin() OR sch_no = app_current_school())
WITH CHECK (app_auth_lookup() OR app_is_superadmin() OR sch_no = app_current_school());

COMMIT;
