BEGIN;

CREATE TABLE jobs (
    job_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    job_name VARCHAR(60) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uq_jobs_name_ci ON jobs (LOWER(job_name));

CREATE TABLE decrees (
    dec_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    dec_name VARCHAR(60) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uq_decrees_name_ci ON decrees (LOWER(dec_name));

CREATE TABLE subjects (
    sub_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sub_name VARCHAR(60) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uq_subjects_name_ci ON subjects (LOWER(sub_name));

CREATE TABLE exams (
    ex_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    ex_name VARCHAR(60) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uq_exams_name_ci ON exams (LOWER(ex_name));

CREATE TABLE periods (
    per_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    per_name VARCHAR(30) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uq_periods_name_ci ON periods (LOWER(per_name));

CREATE TABLE attendance_status (
    ast_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    ast_name VARCHAR(30) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uq_attendance_status_name_ci ON attendance_status (LOWER(ast_name));

CREATE TABLE att_conditions (
    con_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    con_name VARCHAR(30) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uq_att_conditions_name_ci ON att_conditions (LOWER(con_name));

CREATE TABLE staff_status_types (
    sst_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sst_name VARCHAR(40) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uq_staff_status_types_name_ci ON staff_status_types (LOWER(sst_name));

CREATE TABLE amount_types (
    am_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    am_name VARCHAR(60) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uq_amount_types_name_ci ON amount_types (LOWER(am_name));

CREATE TABLE expense_types (
    exp_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    exp_name VARCHAR(60) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE UNIQUE INDEX uq_expense_types_name_ci ON expense_types (LOWER(exp_name));

CREATE TABLE levels (
    lev_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sch_no BIGINT NOT NULL REFERENCES schools(sch_no) ON DELETE RESTRICT,
    lev_name VARCHAR(40) NOT NULL,
    price NUMERIC(10,2) NOT NULL DEFAULT 0 CHECK (price >= 0),
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_levels_id_school UNIQUE (lev_no, sch_no)
);
CREATE UNIQUE INDEX uq_levels_school_name_ci ON levels (sch_no, LOWER(lev_name));
CREATE INDEX ix_levels_school ON levels(sch_no);

CREATE TABLE classes (
    cl_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sch_no BIGINT NOT NULL REFERENCES schools(sch_no) ON DELETE RESTRICT,
    lev_no BIGINT NOT NULL,
    cl_name VARCHAR(40) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active' CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(), updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_classes_level_school FOREIGN KEY (lev_no, sch_no)
        REFERENCES levels(lev_no, sch_no) ON DELETE RESTRICT
);
CREATE UNIQUE INDEX uq_classes_school_name_ci ON classes (sch_no, LOWER(cl_name));
CREATE INDEX ix_classes_school ON classes(sch_no);
CREATE INDEX ix_classes_level ON classes(lev_no);

GRANT SELECT, INSERT, UPDATE, DELETE ON
    jobs, decrees, subjects, exams, periods, attendance_status, att_conditions,
    staff_status_types, amount_types, expense_types, levels, classes
TO kobciye_runtime;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO kobciye_runtime;

INSERT INTO permissions (perm_name, description) VALUES
    ('manage_lookups', 'Create and manage global reference data'),
    ('manage_school_structure', 'Create and manage levels and classes')
ON CONFLICT (perm_name) DO NOTHING;

INSERT INTO role_permissions (role_no, perm_no)
SELECT r.role_no, p.perm_no FROM roles r CROSS JOIN permissions p
WHERE r.role_name = 'SuperAdmin' AND p.perm_name IN ('manage_lookups', 'manage_school_structure')
ON CONFLICT (role_no, perm_no) DO NOTHING;

INSERT INTO role_permissions (role_no, perm_no)
SELECT r.role_no, p.perm_no FROM roles r CROSS JOIN permissions p
WHERE r.role_name = 'SchoolAdmin' AND p.perm_name = 'manage_school_structure'
ON CONFLICT (role_no, perm_no) DO NOTHING;

ALTER TABLE levels ENABLE ROW LEVEL SECURITY;
CREATE POLICY levels_tenant ON levels FOR ALL TO kobciye_runtime
USING (app_is_superadmin() OR sch_no = app_current_school())
WITH CHECK (app_is_superadmin() OR sch_no = app_current_school());

ALTER TABLE classes ENABLE ROW LEVEL SECURITY;
CREATE POLICY classes_tenant ON classes FOR ALL TO kobciye_runtime
USING (app_is_superadmin() OR sch_no = app_current_school())
WITH CHECK (app_is_superadmin() OR sch_no = app_current_school());

DO $$
DECLARE table_name TEXT;
BEGIN
    FOREACH table_name IN ARRAY ARRAY['jobs','decrees','subjects','exams','periods','attendance_status','att_conditions','staff_status_types','amount_types','expense_types']
    LOOP
        EXECUTE format('ALTER TABLE %I ENABLE ROW LEVEL SECURITY', table_name);
        EXECUTE format('CREATE POLICY %I ON %I FOR SELECT TO kobciye_runtime USING (TRUE)', table_name || '_read', table_name);
        EXECUTE format('CREATE POLICY %I ON %I FOR ALL TO kobciye_runtime USING (app_is_superadmin()) WITH CHECK (app_is_superadmin())', table_name || '_write', table_name);
    END LOOP;
END $$;

COMMIT;
