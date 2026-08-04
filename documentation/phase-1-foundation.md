# Phase 1 — Foundation

## Status

Implementation in progress. This document is updated together with the code and becomes the handoff and maintenance reference for Phase 1.

## Purpose

Phase 1 establishes the secure multi-school foundation used by every later module. It owns school identity, academic calendars, authentication, authorization, tenant isolation, and audit history.

## Architecture

```text
HTTP request
  → Gin router
  → JWT authentication middleware
  → account-state refresh from PostgreSQL
  → permission middleware
  → school-scope middleware
  → handler
  → service
  → repository
  → PostgreSQL
```

Handlers only translate HTTP input and output. Services enforce business and security rules. Repositories own database queries. Models describe persisted data. Middleware establishes authenticated user and school context.

## Phase 1 tables

### `schools`

Stores each tenant. School names and non-empty email addresses are unique without regard to letter case. Status is `active` or `inactive`.

### `academic_years`

Stores school-specific academic calendars. `(sch_no, year_name)` is unique, and the end date must be later than the start date.

### `roles`

Seeded roles:

- `SuperAdmin`
- `SchoolAdmin`
- `Registrar`
- `Finance`
- `Teacher`

### `permissions`

Phase 1 permissions:

- `manage_schools`
- `manage_academic_years`
- `manage_users`
- `manage_roles`
- `view_audit_logs`

### `role_permissions`

Many-to-many role and permission assignments. Duplicate assignments are blocked.

### `users`

Username-based login accounts. Passwords are stored only as bcrypt hashes. A user belongs to one school, except `SuperAdmin`, whose `sch_no` is `NULL`. Status is `active`, `locked`, or `disabled`.

Five consecutive failed password attempts change an active account to `locked`. A successful login clears the failure counter.

### `audit_logs`

Append-only security and data history. PostgreSQL rejects updates and deletes through a trigger. Records include actor, school, action, resource, record ID, IP address, timestamp, and JSON metadata.

## Tenant isolation rules

1. The client cannot choose its school freely.
2. A normal user's school comes from the current database account record after JWT validation.
3. A normal user cannot override that school with `X-School-ID`.
4. A `SuperAdmin` may select a school using `X-School-ID` for school-scoped operations.
5. Repository queries for school-owned data always include the resolved school ID.
6. An inactive school cannot be used for new school-scoped operations.

## Authentication rules

- JWT signing algorithm: HMAC SHA-256.
- Tokens include user ID, school ID, role, permissions, issuer, issued-at time, and expiration.
- Middleware reloads the user, role, permissions, and status from PostgreSQL for every authenticated request. This makes role changes, account disabling, and account locking effective immediately instead of waiting for an old token to expire.
- Login errors do not disclose whether a username exists.
- Password length is 12–72 bytes for newly created accounts.
- Production refuses the built-in development JWT secret.

## SuperAdmin bootstrap

Public registration is intentionally unavailable. The first `SuperAdmin` is created only when both bootstrap environment variables are present and no SuperAdmin exists:

```env
BOOTSTRAP_SUPERADMIN_USERNAME=superadmin
BOOTSTRAP_SUPERADMIN_PASSWORD=replace-with-a-strong-password
```

Remove these variables after the first account is created. Subsequent users are created through a protected endpoint requiring `manage_users`.

## Migrations

```text
migrations/000001_foundation.up.sql
migrations/000001_foundation.down.sql
migrations/000002_seed_access_control.up.sql
migrations/000002_seed_access_control.down.sql
```

The SQL migrations are the production source of truth. `AUTO_MIGRATE=true` remains available for local development only.

## Commands

Format and test:

```bash
make fmt
make test
```

Regenerate Swagger:

```bash
make swagger
```

Start the complete development stack:

```bash
make docker-up
```

Stop it:

```bash
make docker-down
```

## API documentation

When the stack is running, Swagger UI is available at:

```text
http://localhost:8081/swagger/index.html
```

The endpoint list will be finalized after the remaining Phase 1 handlers and middleware are implemented.

## Verification checklist

- [x] PostgreSQL-native Phase 1 schema written
- [x] GORM Phase 1 models written
- [x] Seed roles and permissions defined
- [ ] Authentication service completed
- [ ] Five-attempt account lock tested
- [ ] School middleware completed and tested
- [ ] Permission middleware completed and tested
- [ ] SuperAdmin bootstrap completed and tested
- [ ] Foundation endpoints completed
- [ ] Swagger regenerated
- [ ] Unit tests pass
- [ ] `go vet` passes
- [ ] Docker migration and runtime checks pass
