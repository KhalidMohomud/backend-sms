# Kobciye School Backend API

Backend foundation built with Go 1.26, Gin, GORM, PostgreSQL, Redis, JWT, Swagger, and Docker. The code follows a handler → service → repository structure.

Implementation documentation: [Phase 1 — Foundation](documentation/phase-1-foundation.md).

## Project structure

```text
cmd/api/              Small executable entrypoint
cmd/admin/            Operator-only administrative command
docs/                 Generated Swagger files
internal/app/         Dependency wiring and server lifecycle
internal/config/      Environment configuration
internal/database/    PostgreSQL and Redis clients
internal/handler/     HTTP request and response handling
internal/middleware/  Gin authentication middleware
internal/model/       GORM database models
internal/repository/  Persistence interfaces and implementations
internal/router/      Route registration
internal/security/    JWT and password utilities
internal/service/     Application business logic
migrations/           Versioned PostgreSQL migrations
```

## Run with Docker

```sh
make docker-up
```

The repository already contains the local `.env` consumed by Docker Compose. For deployment, create an environment-specific `.env` and replace the development JWT secret.

The API is available at `http://localhost:8081`, health status at `/health`, and Swagger UI at `/swagger/index.html`. Set `API_HOST_PORT` to use another host port; the container always listens on port 8080 internally.

PostgreSQL and Redis are available to the API on Docker's private network and are not published to host ports, avoiding conflicts with locally installed database services.

## Run locally

Start PostgreSQL and Redis, configure `.env` values in your shell, then run:

```sh
make run
```

Environment variables are read from the process environment; `.env` is consumed automatically by Docker Compose.

## API endpoints

- `GET /health`
- `POST /api/v1/auth/login`
- `GET /api/v1/auth/me`
- `GET|POST /api/v1/schools`
- `PATCH|DELETE /api/v1/schools/:id`
- `GET|POST /api/v1/academic-years`
- `PATCH|DELETE /api/v1/academic-years/:id`
- `GET|POST /api/v1/users`
- `PATCH /api/v1/users/:id`
- `PATCH /api/v1/users/:id/status`
- `DELETE /api/v1/users/:id`
- `GET /api/v1/roles`
- `GET /api/v1/permissions`
- `GET /api/v1/audit-logs`

Except for health and login, these endpoints require a Bearer JWT and the documented permission. SuperAdmin must send `X-School-ID` on school-scoped academic-year routes.

School and user `DELETE` routes are safe deletes: they deactivate or disable the record so historical data remains valid. Academic-year deletion is permanent and returns a conflict when dependent records use the year. Roles and permissions are seeded security configuration, and audit logs are append-only, so Phase 1 intentionally exposes no update/delete routes for those resources.

## Create the first SuperAdmin

After the containers are running, execute the operator-only command. It prompts for the password without displaying it:

```sh
make admin-create USERNAME=superadmin
```

The command refuses to run when a SuperAdmin already exists. There is no public registration endpoint and no automatic default account.

## Development

```sh
make fmt
make test
make swagger
```
# backend-sms
