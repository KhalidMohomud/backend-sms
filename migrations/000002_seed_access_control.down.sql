BEGIN;

DELETE FROM role_permissions
WHERE role_no IN (SELECT role_no FROM roles WHERE role_name IN ('SuperAdmin', 'SchoolAdmin'))
  AND perm_no IN (SELECT perm_no FROM permissions WHERE perm_name IN (
      'manage_schools',
      'manage_academic_years',
      'manage_users',
      'manage_roles',
      'view_audit_logs'
  ));

DELETE FROM roles WHERE role_name IN ('SuperAdmin', 'SchoolAdmin', 'Registrar', 'Finance', 'Teacher');
DELETE FROM permissions WHERE perm_name IN (
    'manage_schools',
    'manage_academic_years',
    'manage_users',
    'manage_roles',
    'view_audit_logs'
);

COMMIT;
