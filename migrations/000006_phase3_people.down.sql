BEGIN;

ALTER TABLE users DROP CONSTRAINT IF EXISTS fk_users_staff_school;
DELETE FROM permissions WHERE perm_name IN ('manage_students', 'manage_staff');

DROP TABLE IF EXISTS staff_status;
DROP TABLE IF EXISTS staff;
DROP TABLE IF EXISTS students;
DROP TABLE IF EXISTS responsibles;
DROP TABLE IF EXISTS addresses;

COMMIT;
