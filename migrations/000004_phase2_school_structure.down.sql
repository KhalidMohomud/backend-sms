BEGIN;

DELETE FROM permissions WHERE perm_name IN ('manage_lookups', 'manage_school_structure');

DROP TABLE IF EXISTS classes;
DROP TABLE IF EXISTS levels;
DROP TABLE IF EXISTS expense_types;
DROP TABLE IF EXISTS amount_types;
DROP TABLE IF EXISTS staff_status_types;
DROP TABLE IF EXISTS att_conditions;
DROP TABLE IF EXISTS attendance_status;
DROP TABLE IF EXISTS periods;
DROP TABLE IF EXISTS exams;
DROP TABLE IF EXISTS subjects;
DROP TABLE IF EXISTS decrees;
DROP TABLE IF EXISTS jobs;

COMMIT;
