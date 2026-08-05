# Phase 2 — School Structure and Reference Data

## Status

Implemented. Runtime verification results are recorded in the final section.

## Scope

Phase 2 adds reusable reference data and each school's level/class hierarchy.

Global lookup tables:

- `jobs`
- `decrees`
- `subjects`
- `exams`
- `periods`
- `attendance_status`
- `att_conditions`
- `staff_status_types`
- `amount_types`
- `expense_types`

School-owned tables:

- `levels`
- `classes`

## Design and maintenance boundaries

- `StructureHandler` owns HTTP binding and response codes.
- `StructureService` owns authorization, tenant checks, validation, safe deletion, and audit decisions.
- `StructureRepository` owns database access.
- Lookup table identifiers are selected only from a fixed server-side allowlist. Client values are never used as SQL identifiers.
- Production schema changes use migration `000004_phase2_school_structure`.
- Local `AUTO_MIGRATE=true` creates the same models and indexes for development.

## Authorization

Two permissions are added:

- `manage_lookups`: assigned to `SuperAdmin` only.
- `manage_school_structure`: assigned to `SuperAdmin` and `SchoolAdmin`.

All authenticated users may read global lookup values. Only SuperAdmin may create, update, reactivate, or deactivate them. Levels and classes are readable inside the resolved school. Their mutations require `manage_school_structure`.

SuperAdmin must send `X-School-ID` for level and class endpoints. A normal user's school comes from the current database account; a client cannot select a different school.

## PostgreSQL security

- RLS is enabled on all twelve Phase 2 tables.
- `levels` and `classes` allow only the current school or an authenticated SuperAdmin context.
- Global lookup tables allow runtime reads but require the database `app.is_superadmin` context for writes.
- Class ownership is protected by a composite foreign key: `(lev_no, sch_no)` must reference a level in the same school.
- Names are unique case-insensitively: globally for lookup values and within a school for levels/classes.
- Level prices must be non-negative.
- Status values are restricted to `active` or `inactive`.

## Safe deletion

`DELETE` never physically removes a lookup, level, or class. It changes status to `inactive` and writes a `DELETE` audit event with `soft_delete: true`.

An active level cannot be deactivated while it has active classes. Deactivate its classes first. `PATCH` with `{"status":"active"}` restores an item when business rules allow it.

## Endpoints

### Global lookups

Valid `{type}` values:

```text
jobs
decrees
subjects
exams
periods
attendance-status
attendance-conditions
staff-status-types
amount-types
expense-types
```

| Method | Endpoint | Authorization |
|---|---|---|
| `GET` | `/api/v1/lookups/{type}` | Authenticated |
| `GET` | `/api/v1/lookups/{type}/{id}` | Authenticated |
| `POST` | `/api/v1/lookups/{type}` | SuperAdmin + `manage_lookups` |
| `PATCH` | `/api/v1/lookups/{type}/{id}` | SuperAdmin + `manage_lookups` |
| `DELETE` | `/api/v1/lookups/{type}/{id}` | SuperAdmin + `manage_lookups` |

Create example:

```json
{
  "name": "Biology"
}
```

### Levels

| Method | Endpoint | Authorization |
|---|---|---|
| `GET` | `/api/v1/levels` | Authenticated, school scoped |
| `GET` | `/api/v1/levels/{id}` | Authenticated, school scoped |
| `POST` | `/api/v1/levels` | `manage_school_structure` |
| `PATCH` | `/api/v1/levels/{id}` | `manage_school_structure` |
| `DELETE` | `/api/v1/levels/{id}` | `manage_school_structure` |

Create example:

```json
{
  "name": "Form 1",
  "price": 25,
  "status": "active"
}
```

### Classes

| Method | Endpoint | Authorization |
|---|---|---|
| `GET` | `/api/v1/classes` | Authenticated, school scoped |
| `GET` | `/api/v1/classes/{id}` | Authenticated, school scoped |
| `POST` | `/api/v1/classes` | `manage_school_structure` |
| `PATCH` | `/api/v1/classes/{id}` | `manage_school_structure` |
| `DELETE` | `/api/v1/classes/{id}` | `manage_school_structure` |

Create example:

```json
{
  "name": "Form 1-A",
  "level_id": 1,
  "status": "active"
}
```

## Seed data

Development startup idempotently seeds common jobs, decrees, subjects, exam types, periods, attendance values, staff status types, fee types, and expense types. Seeds can be renamed or deactivated through the API. Repeated startup does not create duplicates.

## Verification

```bash
make test
make test-race
make test-integration
make swagger
make admin-verify
docker compose ps
```

The integration suite proves:

- School A sees its own levels but not School B's levels.
- School A cannot insert a level for School B.
- SchoolAdmin cannot write a global lookup even when bypassing HTTP middleware.
- SuperAdmin RLS behavior remains available.
- Redis session security from Phase 1 remains operational.

