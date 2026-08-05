BEGIN;

DROP POLICY IF EXISTS role_permissions_write ON role_permissions;
DROP POLICY IF EXISTS role_permissions_read ON role_permissions;
DROP POLICY IF EXISTS permissions_read ON permissions;
DROP POLICY IF EXISTS roles_write ON roles;
DROP POLICY IF EXISTS roles_read ON roles;
DROP POLICY IF EXISTS audit_logs_tenant ON audit_logs;
DROP POLICY IF EXISTS users_tenant ON users;
DROP POLICY IF EXISTS academic_years_tenant ON academic_years;
DROP POLICY IF EXISTS schools_write ON schools;
DROP POLICY IF EXISTS schools_select ON schools;

ALTER TABLE role_permissions DISABLE ROW LEVEL SECURITY;
ALTER TABLE permissions DISABLE ROW LEVEL SECURITY;
ALTER TABLE roles DISABLE ROW LEVEL SECURITY;
ALTER TABLE audit_logs DISABLE ROW LEVEL SECURITY;
ALTER TABLE users DISABLE ROW LEVEL SECURITY;
ALTER TABLE academic_years DISABLE ROW LEVEL SECURITY;
ALTER TABLE schools DISABLE ROW LEVEL SECURITY;

DROP FUNCTION IF EXISTS app_auth_lookup();
DROP FUNCTION IF EXISTS app_is_superadmin();
DROP FUNCTION IF EXISTS app_current_user();
DROP FUNCTION IF EXISTS app_current_school();
ALTER TABLE roles DROP COLUMN IF EXISTS is_system;
ALTER TABLE roles DROP COLUMN IF EXISTS status;

COMMIT;
