BEGIN;

CREATE TABLE schools (
    sch_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sch_name VARCHAR(100) NOT NULL,
    address VARCHAR(150),
    tell VARCHAR(20),
    email VARCHAR(254),
    logo VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'inactive')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_schools_name_ci ON schools (LOWER(sch_name));
CREATE UNIQUE INDEX uq_schools_email_ci ON schools (LOWER(email)) WHERE email IS NOT NULL;

CREATE TABLE academic_years (
    y_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sch_no BIGINT NOT NULL REFERENCES schools(sch_no) ON DELETE RESTRICT,
    year_name VARCHAR(20) NOT NULL,
    started DATE NOT NULL,
    ended DATE NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_academic_year_dates CHECK (ended > started),
    CONSTRAINT uq_academic_year_school_name UNIQUE (sch_no, year_name)
);

CREATE INDEX ix_academic_years_school ON academic_years(sch_no);

CREATE TABLE roles (
    role_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    role_name VARCHAR(40) NOT NULL UNIQUE,
    description VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE permissions (
    perm_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    perm_name VARCHAR(60) NOT NULL UNIQUE,
    description VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE role_permissions (
    rp_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    role_no BIGINT NOT NULL REFERENCES roles(role_no) ON DELETE CASCADE,
    perm_no BIGINT NOT NULL REFERENCES permissions(perm_no) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT uq_role_permission UNIQUE (role_no, perm_no)
);

CREATE TABLE users (
    usr_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    sch_no BIGINT REFERENCES schools(sch_no) ON DELETE RESTRICT,
    stf_no BIGINT,
    username VARCHAR(50) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    role_no BIGINT NOT NULL REFERENCES roles(role_no) ON DELETE RESTRICT,
    status VARCHAR(20) NOT NULL DEFAULT 'active'
        CHECK (status IN ('active', 'locked', 'disabled')),
    failed_logins SMALLINT NOT NULL DEFAULT 0
        CHECK (failed_logins BETWEEN 0 AND 5),
    last_login TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX uq_users_username_ci ON users (LOWER(username));
CREATE INDEX ix_users_school ON users(sch_no);
CREATE INDEX ix_users_role ON users(role_no);

CREATE TABLE audit_logs (
    log_no BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    usr_no BIGINT REFERENCES users(usr_no) ON DELETE SET NULL,
    sch_no BIGINT REFERENCES schools(sch_no) ON DELETE SET NULL,
    action VARCHAR(30) NOT NULL,
    table_name VARCHAR(60) NOT NULL,
    record_id BIGINT,
    ip_address VARCHAR(45),
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    log_time TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX ix_audit_logs_school_time ON audit_logs(sch_no, log_time DESC);
CREATE INDEX ix_audit_logs_user_time ON audit_logs(usr_no, log_time DESC);
CREATE INDEX ix_audit_logs_action ON audit_logs(action);

CREATE FUNCTION prevent_audit_log_mutation() RETURNS trigger AS $$
BEGIN
    RAISE EXCEPTION 'audit logs are append-only';
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER audit_logs_no_update_or_delete
    BEFORE UPDATE OR DELETE ON audit_logs
    FOR EACH ROW EXECUTE FUNCTION prevent_audit_log_mutation();

COMMIT;
