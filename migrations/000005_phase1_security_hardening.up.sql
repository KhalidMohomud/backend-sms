BEGIN;

DROP POLICY IF EXISTS users_tenant ON users;
DROP POLICY IF EXISTS users_select ON users;
DROP POLICY IF EXISTS users_insert ON users;
DROP POLICY IF EXISTS users_update ON users;
DROP POLICY IF EXISTS users_delete ON users;

CREATE POLICY users_select ON users FOR SELECT TO kobciye_runtime
USING (
    app_auth_lookup()
    OR app_is_superadmin()
    OR usr_no = app_current_user()
    OR sch_no = app_current_school()
);
CREATE POLICY users_insert ON users FOR INSERT TO kobciye_runtime
WITH CHECK (app_is_superadmin() OR sch_no = app_current_school());
CREATE POLICY users_update ON users FOR UPDATE TO kobciye_runtime
USING (app_is_superadmin() OR usr_no = app_current_user() OR sch_no = app_current_school())
WITH CHECK (app_is_superadmin() OR usr_no = app_current_user() OR sch_no = app_current_school());
CREATE POLICY users_delete ON users FOR DELETE TO kobciye_runtime
USING (app_is_superadmin() OR sch_no = app_current_school());

CREATE OR REPLACE FUNCTION enforce_authentication_user_update() RETURNS trigger AS $$
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
$$ LANGUAGE plpgsql;
DROP TRIGGER IF EXISTS users_authentication_update_guard ON users;
CREATE TRIGGER users_authentication_update_guard
BEFORE UPDATE ON users
FOR EACH ROW EXECUTE FUNCTION enforce_authentication_user_update();

DROP POLICY IF EXISTS audit_logs_tenant ON audit_logs;
DROP POLICY IF EXISTS audit_logs_select ON audit_logs;
DROP POLICY IF EXISTS audit_logs_insert ON audit_logs;

CREATE POLICY audit_logs_select ON audit_logs FOR SELECT TO kobciye_runtime
USING (app_is_superadmin() OR sch_no = app_current_school());
CREATE POLICY audit_logs_insert ON audit_logs FOR INSERT TO kobciye_runtime
WITH CHECK (app_auth_lookup() OR app_is_superadmin() OR sch_no = app_current_school());

COMMIT;
