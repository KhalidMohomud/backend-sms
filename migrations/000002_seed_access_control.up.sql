BEGIN;

INSERT INTO permissions (perm_name, description) VALUES
    ('manage_schools', 'Create and manage schools'),
    ('manage_academic_years', 'Create and manage academic years'),
    ('manage_users', 'Create and manage user accounts'),
    ('manage_roles', 'View and manage access-control definitions'),
    ('view_audit_logs', 'View security and data audit events')
ON CONFLICT (perm_name) DO NOTHING;

INSERT INTO roles (role_name, description) VALUES
    ('SuperAdmin', 'Platform administrator with access to all schools'),
    ('SchoolAdmin', 'Administrator restricted to one school'),
    ('Registrar', 'Student registration and attendance operator'),
    ('Finance', 'School finance operator'),
    ('Teacher', 'Teacher restricted to assigned academic work')
ON CONFLICT (role_name) DO NOTHING;

INSERT INTO role_permissions (role_no, perm_no)
SELECT r.role_no, p.perm_no
FROM roles r
CROSS JOIN permissions p
WHERE r.role_name = 'SuperAdmin'
ON CONFLICT (role_no, perm_no) DO NOTHING;

INSERT INTO role_permissions (role_no, perm_no)
SELECT r.role_no, p.perm_no
FROM roles r
JOIN permissions p ON p.perm_name IN (
    'manage_academic_years',
    'manage_users',
    'view_audit_logs'
)
WHERE r.role_name = 'SchoolAdmin'
ON CONFLICT (role_no, perm_no) DO NOTHING;

COMMIT;
