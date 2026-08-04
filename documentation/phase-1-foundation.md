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
- Accounts belonging to an inactive school are rejected at login and on every authenticated request, including requests using an older JWT.
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

### Upgrade from the original scaffold

The original scaffold used an incompatible email-based `users` table. Phase 1 never silently drops it. Startup detects that schema and instructs the operator to run:

```bash
make admin-archive-legacy-users
```

The command runs one PostgreSQL transaction: it copies every old row to `legacy_users_email_auth` and only then replaces the obsolete table. On this development database, one legacy row was preserved in that archive.

## Commands

```bash
make fmt          # Format Go files
make test         # Run tests
make swagger      # Regenerate API documentation
make docker-up    # Run API, PostgreSQL and Redis
make docker-down  # Stop the development stack
make admin-create USERNAME=superadmin  # Create the first SuperAdmin interactively
make admin-verify   # Verify tables, RBAC assignments, and audit trigger
```

Swagger UI is available at `http://localhost:8081/swagger/index.html` while Docker Compose is running.

## Environment variables

| Variable | Purpose |
|---|---|
| `APP_ENV` | `development` or `production` |
| `APP_PORT` | Internal API port |
| `API_HOST_PORT` | Docker host port |
| `AUTO_MIGRATE` | Local GORM migration switch; disable in production |
| `POSTGRES_HOST`, `POSTGRES_PORT` | PostgreSQL address |
| `POSTGRES_USER`, `POSTGRES_PASSWORD`, `POSTGRES_DB` | PostgreSQL credentials/database |
| `POSTGRES_SSLMODE` | Use an SSL mode appropriate for deployment |
| `REDIS_ADDR`, `REDIS_PASSWORD`, `REDIS_DB` | Redis connection |
| `JWT_SECRET` | JWT signing secret; at least 32 bytes in production |
| `JWT_EXPIRATION` | Access-token lifetime such as `24h` |
| `JWT_ISSUER` | Expected JWT issuer |

Secrets must be supplied by the deployment environment and must never be committed. The API does not trust forwarded client-IP headers until a deployment explicitly configures its reverse-proxy allowlist, preventing forged audit IP addresses.

## Endpoints and authorization

| Method | Endpoint | Authorization |
|---|---|---|
| `POST` | `/api/v1/auth/login` | Public; username and password |
| `GET` | `/api/v1/auth/me` | Authenticated |
| `GET`, `POST` | `/api/v1/schools` | `manage_schools` (SuperAdmin) |
| `PATCH`, `DELETE` | `/api/v1/schools/{id}` | `manage_schools` (SuperAdmin) |
| `GET` | `/api/v1/academic-years` | Authenticated, school scoped |
| `POST` | `/api/v1/academic-years` | `manage_academic_years`, school scoped |
| `PATCH`, `DELETE` | `/api/v1/academic-years/{id}` | `manage_academic_years`, school scoped |
| `GET`, `POST` | `/api/v1/users` | `manage_users` |
| `PATCH` | `/api/v1/users/{id}` | `manage_users`; username, staff link, or role |
| `PATCH` | `/api/v1/users/{id}/status` | `manage_users`; disable or unlock |
| `DELETE` | `/api/v1/users/{id}` | `manage_users`; safe account deletion |
| `GET` | `/api/v1/roles` | `manage_roles` or `manage_users` |
| `GET` | `/api/v1/permissions` | `manage_roles` |
| `GET` | `/api/v1/audit-logs` | `view_audit_logs` |

SchoolAdmin can only create, list, disable, or unlock lower-privilege users in its own school. It cannot manage itself, another SchoolAdmin, or any SuperAdmin. SuperAdmin can manage every school and may create another SuperAdmin only with a `NULL` school.

### Update and delete behavior

- `PATCH /schools/{id}` updates only the supplied fields. `DELETE /schools/{id}` is a safe delete: it changes the school status to `inactive`, preserves school history, and immediately blocks its users. A school can be restored by patching its status to `active`.
- `PATCH /academic-years/{id}` updates only the supplied name or dates and revalidates that the ending date follows the starting date. `DELETE /academic-years/{id}` permanently removes the row. PostgreSQL returns `409 Conflict` when a later phase has dependent records that protect the year.
- `PATCH /users/{id}` changes supplied username, staff link, or role fields. Tenant checks prevent a SchoolAdmin from editing accounts outside its school or assigning SchoolAdmin/SuperAdmin privileges. `PATCH /users/{id}/status` disables or unlocks users. `DELETE /users/{id}` safely disables the account instead of deleting ownership and audit history. A permitted administrator can restore it through the status endpoint.
- Roles and permissions are security configuration seeded by migrations in Phase 1, so they are read-only through the API. Future role-management work must include privilege-escalation protections before exposing write routes.
- Audit logs are intentionally append-only. PostgreSQL rejects both updates and deletes, so no update/delete API exists for them.

Every successful create, update, and delete operation writes an audit entry. Safe deletions use the `DELETE` action with `soft_delete: true` metadata.

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
- [x] Operator-only SuperAdmin command and test
- [x] Foundation endpoints
- [x] School update and safe delete endpoints
- [x] Academic-year update and protected delete endpoints
- [x] User safe delete endpoint
- [x] User profile/role update endpoint with privilege-escalation checks
- [x] Swagger regeneration
- [x] Unit tests
- [x] `go vet`
- [x] Docker runtime verification

Final runtime result:

```text
Foundation verifier: 7 tables, 5 roles, 5 permissions, audit trigger active
GET /health: 200 OK
GET /api/v1/schools without JWT: 401 Unauthorized
GET /swagger/index.html: 200 OK
PostgreSQL: healthy
Redis: healthy
API: running on host port 8081
```
