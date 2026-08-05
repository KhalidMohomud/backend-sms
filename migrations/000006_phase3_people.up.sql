BEGIN;

CREATE TABLE addresses (
    add_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sch_no BIGINT NOT NULL REFERENCES schools(sch_no) ON DELETE RESTRICT,
    district VARCHAR(60) NOT NULL,
    village VARCHAR(60),
    area VARCHAR(60),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_addresses_id_school UNIQUE (add_no, sch_no)
);
CREATE INDEX ix_addresses_school ON addresses(sch_no);

CREATE TABLE responsibles (
    res_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sch_no BIGINT NOT NULL REFERENCES schools(sch_no) ON DELETE RESTRICT,
    res_name VARCHAR(100) NOT NULL,
    res_tell VARCHAR(20) NOT NULL,
    relationship VARCHAR(40) NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_responsibles_id_school UNIQUE (res_no, sch_no)
);
CREATE INDEX ix_responsibles_school ON responsibles(sch_no);
CREATE INDEX ix_responsibles_phone ON responsibles(res_tell);

CREATE TABLE students (
    std_id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sch_no BIGINT NOT NULL REFERENCES schools(sch_no) ON DELETE RESTRICT,
    std_name VARCHAR(100) NOT NULL,
    mother_name VARCHAR(100) NOT NULL,
    sex CHAR(1) NOT NULL CHECK (sex IN ('M', 'F')),
    tell VARCHAR(20),
    b_date DATE,
    p_birth VARCHAR(60),
    add_no BIGINT,
    res_no BIGINT NOT NULL,
    image VARCHAR(255),
    reg_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'graduated', 'transferred', 'dropped')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_students_id_school UNIQUE (std_id, sch_no),
    CONSTRAINT fk_students_address_school FOREIGN KEY (add_no, sch_no)
        REFERENCES addresses(add_no, sch_no) ON DELETE RESTRICT,
    CONSTRAINT fk_students_guardian_school FOREIGN KEY (res_no, sch_no)
        REFERENCES responsibles(res_no, sch_no) ON DELETE RESTRICT
);
CREATE INDEX ix_students_school ON students(sch_no);
CREATE INDEX ix_students_school_status ON students(sch_no, status);
CREATE INDEX ix_students_name ON students(std_name);

CREATE TABLE staff (
    stf_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sch_no BIGINT NOT NULL REFERENCES schools(sch_no) ON DELETE RESTRICT,
    stf_name VARCHAR(100) NOT NULL,
    sex CHAR(1) NOT NULL CHECK (sex IN ('M', 'F')),
    tell VARCHAR(20),
    add_no BIGINT,
    job_no BIGINT NOT NULL REFERENCES jobs(job_no) ON DELETE RESTRICT,
    dec_no BIGINT NOT NULL REFERENCES decrees(dec_no) ON DELETE RESTRICT,
    salary NUMERIC(10,2) NOT NULL CHECK (salary >= 0),
    hired_date DATE NOT NULL,
    description VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_staff_id_school UNIQUE (stf_no, sch_no),
    CONSTRAINT fk_staff_address_school FOREIGN KEY (add_no, sch_no)
        REFERENCES addresses(add_no, sch_no) ON DELETE RESTRICT
);
CREATE INDEX ix_staff_school ON staff(sch_no);
CREATE INDEX ix_staff_name ON staff(stf_name);
CREATE INDEX ix_staff_job ON staff(job_no);

CREATE TABLE staff_status (
    ss_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sch_no BIGINT NOT NULL REFERENCES schools(sch_no) ON DELETE RESTRICT,
    stf_no BIGINT NOT NULL,
    sst_no BIGINT NOT NULL REFERENCES staff_status_types(sst_no) ON DELETE RESTRICT,
    description VARCHAR(255),
    st_date DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT fk_staff_status_staff_school FOREIGN KEY (stf_no, sch_no)
        REFERENCES staff(stf_no, sch_no) ON DELETE RESTRICT,
    CONSTRAINT uq_staff_status_event UNIQUE (stf_no, sst_no, st_date)
);
CREATE INDEX ix_staff_status_school ON staff_status(sch_no);
CREATE INDEX ix_staff_status_staff_date ON staff_status(stf_no, st_date DESC, ss_no DESC);

ALTER TABLE users
    ADD CONSTRAINT fk_users_staff_school FOREIGN KEY (stf_no, sch_no)
    REFERENCES staff(stf_no, sch_no) ON DELETE RESTRICT;

GRANT SELECT, INSERT, UPDATE, DELETE ON addresses, responsibles, students, staff, staff_status TO kobciye_runtime;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO kobciye_runtime;

INSERT INTO permissions (perm_name, description) VALUES
    ('manage_students', 'Create and manage student records, guardians, and addresses'),
    ('manage_staff', 'Create and manage staff records and employment status history')
ON CONFLICT (perm_name) DO NOTHING;

INSERT INTO role_permissions (role_no, perm_no, created_at)
SELECT r.role_no, p.perm_no, NOW() FROM roles r CROSS JOIN permissions p
WHERE r.role_name = 'SuperAdmin' AND p.perm_name IN ('manage_students', 'manage_staff')
ON CONFLICT (role_no, perm_no) DO NOTHING;

INSERT INTO role_permissions (role_no, perm_no, created_at)
SELECT r.role_no, p.perm_no, NOW() FROM roles r CROSS JOIN permissions p
WHERE r.role_name = 'SchoolAdmin' AND p.perm_name IN ('manage_students', 'manage_staff')
ON CONFLICT (role_no, perm_no) DO NOTHING;

INSERT INTO role_permissions (role_no, perm_no, created_at)
SELECT r.role_no, p.perm_no, NOW() FROM roles r CROSS JOIN permissions p
WHERE r.role_name = 'Registrar' AND p.perm_name = 'manage_students'
ON CONFLICT (role_no, perm_no) DO NOTHING;

ALTER TABLE addresses ENABLE ROW LEVEL SECURITY;
CREATE POLICY addresses_tenant ON addresses FOR ALL TO kobciye_runtime
USING (app_is_superadmin() OR sch_no = app_current_school())
WITH CHECK (app_is_superadmin() OR sch_no = app_current_school());

ALTER TABLE responsibles ENABLE ROW LEVEL SECURITY;
CREATE POLICY responsibles_tenant ON responsibles FOR ALL TO kobciye_runtime
USING (app_is_superadmin() OR sch_no = app_current_school())
WITH CHECK (app_is_superadmin() OR sch_no = app_current_school());

ALTER TABLE students ENABLE ROW LEVEL SECURITY;
CREATE POLICY students_tenant ON students FOR ALL TO kobciye_runtime
USING (app_is_superadmin() OR sch_no = app_current_school())
WITH CHECK (app_is_superadmin() OR sch_no = app_current_school());

ALTER TABLE staff ENABLE ROW LEVEL SECURITY;
CREATE POLICY staff_tenant ON staff FOR ALL TO kobciye_runtime
USING (app_is_superadmin() OR sch_no = app_current_school())
WITH CHECK (app_is_superadmin() OR sch_no = app_current_school());

ALTER TABLE staff_status ENABLE ROW LEVEL SECURITY;
CREATE POLICY staff_status_tenant ON staff_status FOR ALL TO kobciye_runtime
USING (app_is_superadmin() OR sch_no = app_current_school())
WITH CHECK (app_is_superadmin() OR sch_no = app_current_school());

COMMIT;
