BEGIN;

CREATE UNIQUE INDEX IF NOT EXISTS uq_academic_years_id_school ON academic_years(y_no, sch_no);
CREATE UNIQUE INDEX IF NOT EXISTS uq_classes_id_school ON classes(cl_no, sch_no);

CREATE TABLE student_classes (
    sc_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sch_no BIGINT NOT NULL REFERENCES schools(sch_no) ON DELETE RESTRICT,
    std_id BIGINT NOT NULL,
    cl_no BIGINT NOT NULL,
    y_no BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_student_classes_id_school UNIQUE (sc_no, sch_no),
    CONSTRAINT uq_student_classes_student_year UNIQUE (sch_no, std_id, y_no),
    CONSTRAINT fk_student_classes_student_school FOREIGN KEY (std_id, sch_no)
        REFERENCES students(std_id, sch_no) ON DELETE RESTRICT,
    CONSTRAINT fk_student_classes_class_school FOREIGN KEY (cl_no, sch_no)
        REFERENCES classes(cl_no, sch_no) ON DELETE RESTRICT,
    CONSTRAINT fk_student_classes_year_school FOREIGN KEY (y_no, sch_no)
        REFERENCES academic_years(y_no, sch_no) ON DELETE RESTRICT
);
CREATE INDEX ix_student_classes_school ON student_classes(sch_no);
CREATE INDEX ix_student_classes_class_year ON student_classes(cl_no, y_no);

CREATE TABLE subject_classes (
    sb_cl_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sch_no BIGINT NOT NULL REFERENCES schools(sch_no) ON DELETE RESTRICT,
    sub_no BIGINT NOT NULL REFERENCES subjects(sub_no) ON DELETE RESTRICT,
    cl_no BIGINT NOT NULL,
    stf_no BIGINT NOT NULL,
    y_no BIGINT NOT NULL,
    max_mark NUMERIC(5,2) NOT NULL DEFAULT 100 CHECK (max_mark > 0 AND max_mark <= 999.99),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_subject_classes_id_school UNIQUE (sb_cl_no, sch_no),
    CONSTRAINT uq_subject_classes_assignment UNIQUE (sch_no, sub_no, cl_no, y_no),
    CONSTRAINT fk_subject_classes_class_school FOREIGN KEY (cl_no, sch_no)
        REFERENCES classes(cl_no, sch_no) ON DELETE RESTRICT,
    CONSTRAINT fk_subject_classes_staff_school FOREIGN KEY (stf_no, sch_no)
        REFERENCES staff(stf_no, sch_no) ON DELETE RESTRICT,
    CONSTRAINT fk_subject_classes_year_school FOREIGN KEY (y_no, sch_no)
        REFERENCES academic_years(y_no, sch_no) ON DELETE RESTRICT
);
CREATE INDEX ix_subject_classes_school ON subject_classes(sch_no);
CREATE INDEX ix_subject_classes_class_year ON subject_classes(cl_no, y_no);
CREATE INDEX ix_subject_classes_staff_year ON subject_classes(stf_no, y_no);

CREATE TABLE exam_registrations (
    ex_r_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sch_no BIGINT NOT NULL REFERENCES schools(sch_no) ON DELETE RESTRICT,
    ex_no BIGINT NOT NULL REFERENCES exams(ex_no) ON DELETE RESTRICT,
    y_no BIGINT NOT NULL,
    started DATE NOT NULL,
    ended DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_exam_registrations_dates CHECK (ended >= started),
    CONSTRAINT uq_exam_registrations_id_school UNIQUE (ex_r_no, sch_no),
    CONSTRAINT uq_exam_registrations_schedule UNIQUE (sch_no, ex_no, y_no, started),
    CONSTRAINT fk_exam_registrations_year_school FOREIGN KEY (y_no, sch_no)
        REFERENCES academic_years(y_no, sch_no) ON DELETE RESTRICT
);
CREATE INDEX ix_exam_registrations_school ON exam_registrations(sch_no);
CREATE INDEX ix_exam_registrations_year_dates ON exam_registrations(y_no, started, ended);

CREATE TABLE exam_results (
    res_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sch_no BIGINT NOT NULL REFERENCES schools(sch_no) ON DELETE RESTRICT,
    ex_r_no BIGINT NOT NULL,
    sc_no BIGINT NOT NULL,
    sb_cl_no BIGINT NOT NULL,
    marks NUMERIC(5,2) NOT NULL CHECK (marks >= 0),
    recorded_by BIGINT NOT NULL REFERENCES users(usr_no) ON DELETE RESTRICT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_exam_results_entry UNIQUE (sch_no, ex_r_no, sc_no, sb_cl_no),
    CONSTRAINT fk_exam_results_registration_school FOREIGN KEY (ex_r_no, sch_no)
        REFERENCES exam_registrations(ex_r_no, sch_no) ON DELETE RESTRICT,
    CONSTRAINT fk_exam_results_enrollment_school FOREIGN KEY (sc_no, sch_no)
        REFERENCES student_classes(sc_no, sch_no) ON DELETE RESTRICT,
    CONSTRAINT fk_exam_results_assignment_school FOREIGN KEY (sb_cl_no, sch_no)
        REFERENCES subject_classes(sb_cl_no, sch_no) ON DELETE RESTRICT
);
CREATE INDEX ix_exam_results_school ON exam_results(sch_no);
CREATE INDEX ix_exam_results_registration ON exam_results(ex_r_no);
CREATE INDEX ix_exam_results_enrollment ON exam_results(sc_no);

CREATE OR REPLACE FUNCTION validate_exam_registration_scope() RETURNS TRIGGER
LANGUAGE plpgsql AS $$
DECLARE
    year_started DATE;
    year_ended DATE;
BEGIN
    SELECT started, ended INTO STRICT year_started, year_ended
    FROM academic_years WHERE y_no = NEW.y_no AND sch_no = NEW.sch_no;
    IF NEW.started < year_started OR NEW.ended > year_ended THEN
        RAISE EXCEPTION 'exam dates must be inside the academic year' USING ERRCODE = '23514';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER exam_registrations_validate_scope
BEFORE INSERT OR UPDATE ON exam_registrations
FOR EACH ROW EXECUTE FUNCTION validate_exam_registration_scope();

CREATE OR REPLACE FUNCTION validate_exam_result_scope() RETURNS TRIGGER
LANGUAGE plpgsql AS $$
DECLARE
    exam_year BIGINT;
    enrollment_year BIGINT;
    enrollment_class BIGINT;
    assignment_year BIGINT;
    assignment_class BIGINT;
    assignment_staff BIGINT;
    allowed_mark NUMERIC(5,2);
    recorder_role VARCHAR(40);
    recorder_staff BIGINT;
BEGIN
    SELECT y_no INTO STRICT exam_year FROM exam_registrations
        WHERE ex_r_no = NEW.ex_r_no AND sch_no = NEW.sch_no;
    SELECT y_no, cl_no INTO STRICT enrollment_year, enrollment_class FROM student_classes
        WHERE sc_no = NEW.sc_no AND sch_no = NEW.sch_no;
    SELECT y_no, cl_no, stf_no, max_mark
        INTO STRICT assignment_year, assignment_class, assignment_staff, allowed_mark
        FROM subject_classes WHERE sb_cl_no = NEW.sb_cl_no AND sch_no = NEW.sch_no;

    IF exam_year <> enrollment_year OR exam_year <> assignment_year OR enrollment_class <> assignment_class THEN
        RAISE EXCEPTION 'exam, enrollment, and subject assignment must share the same class and academic year'
            USING ERRCODE = '23514';
    END IF;
    IF NEW.marks > allowed_mark THEN
        RAISE EXCEPTION 'marks exceed the assigned maximum mark' USING ERRCODE = '23514';
    END IF;

    SELECT r.role_name, u.stf_no INTO STRICT recorder_role, recorder_staff
    FROM users u JOIN roles r ON r.role_no = u.role_no WHERE u.usr_no = NEW.recorded_by;
    IF recorder_role = 'Teacher' AND (recorder_staff IS NULL OR recorder_staff <> assignment_staff) THEN
        RAISE EXCEPTION 'teacher is not assigned to this subject class' USING ERRCODE = '42501';
    END IF;
    RETURN NEW;
END;
$$;
CREATE TRIGGER exam_results_validate_scope
BEFORE INSERT OR UPDATE ON exam_results
FOR EACH ROW EXECUTE FUNCTION validate_exam_result_scope();

GRANT SELECT, INSERT, UPDATE, DELETE ON student_classes, subject_classes, exam_registrations, exam_results TO kobciye_runtime;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO kobciye_runtime;

INSERT INTO permissions (perm_name, description) VALUES
    ('manage_enrollments', 'Enroll students into classes for academic years'),
    ('manage_subject_assignments', 'Assign subjects and teachers to classes'),
    ('manage_exams', 'Schedule and manage school exams'),
    ('enter_marks', 'Create, correct, and remove exam marks'),
    ('view_results', 'View academic enrollments, assignments, exams, and results')
ON CONFLICT (perm_name) DO NOTHING;

INSERT INTO role_permissions (role_no, perm_no, created_at)
SELECT r.role_no, p.perm_no, NOW() FROM roles r CROSS JOIN permissions p
WHERE r.role_name IN ('SuperAdmin', 'SchoolAdmin') AND p.perm_name IN (
    'manage_enrollments', 'manage_subject_assignments', 'manage_exams', 'enter_marks', 'view_results'
)
ON CONFLICT (role_no, perm_no) DO NOTHING;

INSERT INTO role_permissions (role_no, perm_no, created_at)
SELECT r.role_no, p.perm_no, NOW() FROM roles r CROSS JOIN permissions p
WHERE r.role_name = 'Registrar' AND p.perm_name IN ('manage_enrollments', 'view_results')
ON CONFLICT (role_no, perm_no) DO NOTHING;

INSERT INTO role_permissions (role_no, perm_no, created_at)
SELECT r.role_no, p.perm_no, NOW() FROM roles r CROSS JOIN permissions p
WHERE r.role_name = 'Teacher' AND p.perm_name IN ('enter_marks', 'view_results')
ON CONFLICT (role_no, perm_no) DO NOTHING;

ALTER TABLE student_classes ENABLE ROW LEVEL SECURITY;
CREATE POLICY student_classes_tenant ON student_classes FOR ALL TO kobciye_runtime
USING (app_is_superadmin() OR sch_no = app_current_school())
WITH CHECK (app_is_superadmin() OR sch_no = app_current_school());

ALTER TABLE subject_classes ENABLE ROW LEVEL SECURITY;
CREATE POLICY subject_classes_tenant ON subject_classes FOR ALL TO kobciye_runtime
USING (app_is_superadmin() OR sch_no = app_current_school())
WITH CHECK (app_is_superadmin() OR sch_no = app_current_school());

ALTER TABLE exam_registrations ENABLE ROW LEVEL SECURITY;
CREATE POLICY exam_registrations_tenant ON exam_registrations FOR ALL TO kobciye_runtime
USING (app_is_superadmin() OR sch_no = app_current_school())
WITH CHECK (app_is_superadmin() OR sch_no = app_current_school());

ALTER TABLE exam_results ENABLE ROW LEVEL SECURITY;
CREATE POLICY exam_results_tenant ON exam_results FOR ALL TO kobciye_runtime
USING (app_is_superadmin() OR sch_no = app_current_school())
WITH CHECK (app_is_superadmin() OR sch_no = app_current_school());

COMMIT;
