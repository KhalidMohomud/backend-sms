# Phase 1 — Foundation

## Status

Implementation complete; runtime verification status is recorded at the end of this document. This file is the maintenance reference for Phase 1.

## Scope

Phase 1 establishes school tenancy, academic calendars, authentication, role-based authorization, and auditing.

### Tables

- `schools`
- `academic_years`
- `roles`
- `permissions`
- `role_permissions`
- `users`
- `audit_logs`

## Request architecture

```text
Gin router
  → JWT authentication
  → current account reload from PostgreSQL
  → permission check
  → school-scope resolution
  → handler
  → service
  → repository
  → PostgreSQL
```

- Handlers translate HTTP input and output.
- Services enforce business and security rules.
- Repositories own database queries and tenant filters.
- Middleware establishes trusted user and school context.

## Tenant isolation

1. A normal user's school is loaded from their current database account, never trusted from request JSON.
2. Normal users cannot override their school using `X-School-ID`.
3. A SuperAdmin has `sch_no = NULL` and may select a school using `X-School-ID` for school-scoped operations.
4. School-owned repository queries include the resolved school ID.
5. Inactive schools cannot receive new school-owned records.

## Authentication and account security

- Usernames are unique without regard to letter case.
- Passwords are stored only as bcrypt hashes.
- New passwords must contain 12–72 bytes.
- Five consecutive failed logins lock an active account.
- Successful login clears the failed-login counter.
- JWT uses HMAC SHA-256 and contains the user ID, school ID, role, permissions, issuer, issue time, and expiration.
- Authenticated requests reload account status and permissions from PostgreSQL so disabling or changing an account takes effect immediately.
- Login errors do not disclose whether a username exists.
- Production refuses the development JWT secret.

## SuperAdmin

There is no public registration and no automatic startup bootstrap. The first SuperAdmin is created explicitly using an operator-only backend command. Subsequent accounts are created through an authenticated endpoint requiring `manage_users`.

```bash
make docker-up
make admin-create USERNAME=superadmin
```

The second command prompts for a 12–72 character password without echoing it. It refuses to create an account if a SuperAdmin already exists. An existing SuperAdmin can create additional SuperAdmins through the protected users endpoint.

## Seeded access control

Roles:

- `SuperAdmin`
- `SchoolAdmin`
- `Registrar`
- `Finance`
- `Teacher`

Phase 1 permissions:

- `manage_schools`
- `manage_academic_years`
- `manage_users`
- `manage_roles`
- `view_audit_logs`

SuperAdmin receives every permission. SchoolAdmin receives `manage_academic_years`, `manage_users`, and `view_audit_logs`, restricted to its own school.

## Audit security

Audit rows record actor, school, action, resource, record ID, IP address, timestamp, and JSON metadata. A PostgreSQL trigger rejects updates and deletes so audit history is append-only.

## Migrations

```text
migrations/000001_foundation.up.sql
migrations/000001_foundation.down.sql
migrations/000002_seed_access_control.up.sql
migrations/000002_seed_access_control.down.sql
```

SQL migrations are the production source of truth. `AUTO_MIGRATE=true` is intended only for local development.

## Commands

```bash
make fmt          # Format Go files
make test         # Run tests
make swagger      # Regenerate API documentation
make docker-up    # Run API, PostgreSQL and Redis
make docker-down  # Stop the development stack
make admin-create USERNAME=superadmin  # Create the first SuperAdmin interactively
```

Swagger UI is available at `http://localhost:8081/swagger/index.html` while Docker Compose is running.

## Endpoints and authorization

| Method | Endpoint | Authorization |
|---|---|---|
| `POST` | `/api/v1/auth/login` | Public; username and password |
| `GET` | `/api/v1/auth/me` | Authenticated |
| `GET`, `POST` | `/api/v1/schools` | `manage_schools` (SuperAdmin) |
| `GET` | `/api/v1/academic-years` | Authenticated, school scoped |
| `POST` | `/api/v1/academic-years` | `manage_academic_years`, school scoped |
| `GET`, `POST` | `/api/v1/users` | `manage_users` |
| `PATCH` | `/api/v1/users/{id}/status` | `manage_users`; disable or unlock |
| `GET` | `/api/v1/roles` | `manage_roles` or `manage_users` |
| `GET` | `/api/v1/permissions` | `manage_roles` |
| `GET` | `/api/v1/audit-logs` | `view_audit_logs` |

SchoolAdmin can only create, list, disable, or unlock lower-privilege users in its own school. It cannot manage itself, another SchoolAdmin, or any SuperAdmin. SuperAdmin can manage every school and may create another SuperAdmin only with a `NULL` school.

## Login example

```bash
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"superadmin","password":"your-password"}'
```

Use the returned token:

```bash
curl http://localhost:8081/api/v1/schools \
  -H 'Authorization: Bearer YOUR_TOKEN'
```

## Verification checklist

- [x] PostgreSQL-native schema
- [x] GORM models
- [x] Access-control seed
- [x] JWT authentication
- [x] Five-attempt account lock test
- [x] School middleware and tests
- [x] Permission middleware and tests
- [x] Operator-only SuperAdmin command
- [x] Foundation endpoints
- [x] Swagger regeneration
- [x] Unit tests
- [x] `go vet`
- [ ] Docker runtime verification
