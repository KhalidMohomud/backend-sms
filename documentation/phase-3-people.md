# Phase 3 — People

## Status

Implemented. Phase 3 adds school-scoped student, guardian, address, staff, and employment-status management.

## Scope

Tables:

- `addresses`
- `responsibles` (exposed as guardians in the API)
- `students`
- `staff`
- `staff_status`

Production migration: `000006_phase3_people`.

For production, use a clean PostgreSQL database with `AUTO_MIGRATE=false` and apply `make migrate-up`. A local database created with `AUTO_MIGRATE=true` has the schema but no SQL migration history, so `make migrate-status` can correctly report the SQL files as pending for that development database.

## Security correction to the design document

The original table sketches omitted `sch_no` from addresses, guardians, and staff-status history. Those rows contain school-owned personal information, so Phase 3 deliberately adds `sch_no` to all five tables. PostgreSQL RLS can therefore reject cross-school reads and writes directly instead of depending on joins or application behavior.

Composite foreign keys enforce matching school ownership:

- Student address and guardian must belong to the student's school.
- Staff address must belong to the staff member's school.
- Staff status history must belong to the same school as the staff member.
- A user's optional staff link must belong to the user's school.

## Permissions

- `manage_students`: SuperAdmin, SchoolAdmin, and Registrar.
- `manage_staff`: SuperAdmin and SchoolAdmin.

SuperAdmin must send `X-School-ID` on every Phase 3 endpoint. Other users receive their school from the authenticated database account and cannot override it.

Addresses are available to an account holding either people permission because they may belong to a student or staff member. Guardians require `manage_students`.

## Business rules

- Names are trimmed and whitespace-only values are rejected.
- Student and staff dates use `YYYY-MM-DD`.
- Birth dates must be earlier than today.
- Registration, hire, and employment-status dates cannot be in the future.
- Sex is `M` or `F`.
- Salary cannot be negative.
- Jobs, decrees, and staff-status types must exist and be active.
- List endpoints support `search`, `limit`, and `offset`; the maximum page size is 100.
- Student lists also support `status=active|graduated|transferred|dropped`.
- Student and staff updates can remove an optional address with `{"clear_address": true}`; it cannot be combined with `address_id`.

## Deletion and history

- Student deletion is a safe delete: status becomes `dropped` and history remains available.
- Staff deletion appends a `Resigned` event to `staff_status`; the staff row is retained.
- Staff status is append-only through the API. Corrections are added as a newer event.
- An address or guardian can be physically removed only while no student or staff record references it; otherwise the API returns `409 Conflict`.
- Every successful mutation writes an audit row in the same PostgreSQL transaction.

Creating staff also creates the initial `Active` employment-status event atomically. A staff record cannot be created without its initial history.

## Endpoints

| Method | Endpoint | Authorization |
|---|---|---|
| `GET`, `POST` | `/api/v1/addresses` | `manage_students` or `manage_staff` |
| `GET`, `PATCH`, `DELETE` | `/api/v1/addresses/{id}` | `manage_students` or `manage_staff` |
| `GET`, `POST` | `/api/v1/guardians` | `manage_students` |
| `GET`, `PATCH`, `DELETE` | `/api/v1/guardians/{id}` | `manage_students` |
| `GET`, `POST` | `/api/v1/students` | `manage_students` |
| `GET`, `PATCH`, `DELETE` | `/api/v1/students/{id}` | `manage_students` |
| `GET`, `POST` | `/api/v1/staff` | `manage_staff` |
| `GET`, `PATCH`, `DELETE` | `/api/v1/staff/{id}` | `manage_staff` |
| `GET`, `POST` | `/api/v1/staff/{id}/statuses` | `manage_staff` |

### Suggested creation order

1. Create an address when required.
2. Create the guardian.
3. Create the student using `guardian_id` and optional `address_id`.

Staff uses an optional address plus existing job and decree lookup IDs.

Student example:

```json
{
  "name": "Amina Ahmed",
  "mother_name": "Hodan Ali",
  "sex": "F",
  "birth_date": "2012-04-10",
  "birth_place": "Mogadishu",
  "address_id": 1,
  "guardian_id": 1,
  "registered_on": "2026-08-05",
  "status": "active"
}
```

Staff example:

```json
{
  "name": "Mohamed Ali",
  "sex": "M",
  "job_id": 1,
  "decree_id": 1,
  "salary": 500,
  "hired_date": "2026-08-05",
  "description": "Mathematics teacher"
}
```

## Student images

The `image` field stores only a path or object-storage key. Phase 3 does not accept binary uploads. A future storage endpoint must rename files, verify content type, enforce a strict size limit, and store objects outside the API web root before saving the path.

## Verification

```bash
make test
make test-race
make test-integration
make swagger
make admin-verify
```

The integration suite proves that School A cannot read School B's addresses, guardians, students, staff, or staff-status history; cannot insert a people record for School B; and that SuperAdmin retains explicit cross-school access.

Verified on a clean database: migrations `000001` through `000006`, 24 required tables, 9 permissions, audit protection, and RLS policies all pass the administrative verifier.
