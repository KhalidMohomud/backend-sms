BEGIN;

DROP TRIGGER IF EXISTS exam_results_validate_scope ON exam_results;
DROP FUNCTION IF EXISTS validate_exam_result_scope();
DROP TRIGGER IF EXISTS exam_registrations_validate_scope ON exam_registrations;
DROP FUNCTION IF EXISTS validate_exam_registration_scope();

DELETE FROM role_permissions
WHERE perm_no IN (SELECT perm_no FROM permissions WHERE perm_name IN (
    'manage_enrollments', 'manage_subject_assignments', 'manage_exams', 'enter_marks', 'view_results'
));
DELETE FROM permissions WHERE perm_name IN (
    'manage_enrollments', 'manage_subject_assignments', 'manage_exams', 'enter_marks', 'view_results'
);

DROP TABLE IF EXISTS exam_results;
DROP TABLE IF EXISTS exam_registrations;
DROP TABLE IF EXISTS subject_classes;
DROP TABLE IF EXISTS student_classes;

DROP INDEX IF EXISTS uq_classes_id_school;
DROP INDEX IF EXISTS uq_academic_years_id_school;

COMMIT;
