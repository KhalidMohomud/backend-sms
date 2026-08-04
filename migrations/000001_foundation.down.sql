BEGIN;

DROP TRIGGER IF EXISTS audit_logs_no_update_or_delete ON audit_logs;
DROP FUNCTION IF EXISTS prevent_audit_log_mutation();
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS role_permissions;
DROP TABLE IF EXISTS permissions;
DROP TABLE IF EXISTS roles;
DROP TABLE IF EXISTS academic_years;
DROP TABLE IF EXISTS schools;

COMMIT;
