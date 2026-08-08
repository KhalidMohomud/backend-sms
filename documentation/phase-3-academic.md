# Phase 3 — Academic

## Status

Implemented. Phase 3 connects existing students, classes, subjects, teachers, exams, and academic years into a secure academic workflow.

## Scope

The global `subjects` and `exams` lookups were prepared in Phase 2. Phase 3 adds these school-owned tables:

- `student_classes`: one student's class enrollment for one academic year.
- `subject_classes`: one subject and teacher assignment for a class and year.
- `exam_registrations`: an exam schedule inside a school's academic year.
- `exam_results`: one mark for an enrollment, subject assignment, and exam.

Production migration: `000007_phase3_academic`.

## Security design

The design document derives school ownership through related records for several academic tables. The implementation also stores `sch_no` directly on every academic table. This supports simple, enforceable PostgreSQL RLS and composite foreign keys.

Security is enforced in four layers:

1. JWT authentication reloads current user permissions from PostgreSQL on every request.
2. School middleware resolves the school from the account; only SuperAdmin can select it with `X-School-ID`.
3. Services validate permissions, school ownership, teacher assignment, class, year, dates, and marks.
4. PostgreSQL RLS and integrity triggers reject invalid or cross-school data if application validation is bypassed.

Academic responses expose student and teacher names but do not embed unrelated personal, guardian, address, or salary data.

## Permissions

| Permission | Default roles |
|---|---|
| `manage_enrollments` | SuperAdmin, SchoolAdmin, Registrar |
| `manage_subject_assignments` | SuperAdmin, SchoolAdmin |
| `manage_exams` | SuperAdmin, SchoolAdmin |
| `enter_marks` | SuperAdmin, SchoolAdmin, Teacher |
| `view_results` | SuperAdmin, SchoolAdmin, Registrar, Teacher |

Teachers are limited to their own `staff_id` assignments. A Teacher without a linked staff record cannot access teacher-scoped academic data.

## Business rules

- A student can have only one class enrollment per academic year.
- Only active students can be enrolled.
- Classes must be active.
- Student, class, teacher, academic year, assignment, exam, and result must belong to the same school.
- A subject can be assigned only once to a class in an academic year.
- Assigned staff must have `Active` as their latest employment status.
- `max_mark` defaults to `100` and must be between `0.01` and `999.99`.
- Exam dates use `YYYY-MM-DD`, must be ordered, and must remain inside the selected academic year.
- An exam result must use an enrollment and subject assignment for the same class and academic year as the exam.
- Marks must be between zero and the subject assignment's `max_mark`.
- The same exam, enrollment, and subject assignment cannot receive a duplicate result.
- Teachers may create, update, delete, and view results only for subject assignments linked to their staff record.
- Every successful mutation and mark correction is audited in the same request transaction.

## Safe update and deletion rules

- An enrollment cannot be updated or deleted after results reference it.
- A subject assignment, including its teacher or maximum mark, cannot change after results reference it.
- An exam registration cannot change or be deleted after results reference it.
- Exam-result identity is immutable; corrections update only `marks` and retain audit metadata containing old and new values.
- Deleting a result is permitted only to an account with `enter_marks`, remains teacher-scoped, and writes an audit event.

Future attendance tables will reference `student_classes`; their foreign key will automatically extend the enrollment deletion protection.

## Endpoints

| Method | Endpoint | Mutation permission |
|---|---|---|
| `GET`, `POST` | `/api/v1/student-classes` | `manage_enrollments` |
| `GET`, `PATCH`, `DELETE` | `/api/v1/student-classes/{id}` | `manage_enrollments` |
| `GET`, `POST` | `/api/v1/subject-classes` | `manage_subject_assignments` |
| `GET`, `PATCH`, `DELETE` | `/api/v1/subject-classes/{id}` | `manage_subject_assignments` |
| `GET`, `POST` | `/api/v1/exam-registrations` | `manage_exams` |
| `GET`, `PATCH`, `DELETE` | `/api/v1/exam-registrations/{id}` | `manage_exams` |
| `GET`, `POST` | `/api/v1/exam-results` | `enter_marks` |
| `GET`, `PATCH`, `DELETE` | `/api/v1/exam-results/{id}` | `enter_marks` |

Read endpoints accept an appropriate academic permission. Teacher reads are automatically restricted to their own assignments.

List filters include:

- `academic_year_id`
- `class_id`
- `student_id`
- `subject_id`
- `staff_id`
- `exam_id`
- `exam_registration_id`
- `student_class_id`
- `subject_class_id`
- `search`, `limit`, and `offset` where applicable

## Postman test order

Log in and place `access_token` in Authorization → Bearer Token. SuperAdmin must also send `X-School-ID` with the target school ID.

### 1. Obtain prerequisite IDs

```text
GET /api/v1/students?status=active
GET /api/v1/classes
GET /api/v1/academic-years
GET /api/v1/staff
GET /api/v1/lookups/subjects
GET /api/v1/lookups/exams
```

### 2. Enroll a student

```http
POST /api/v1/student-classes
Content-Type: application/json

{
  "student_id": 1,
  "class_id": 1,
  "academic_year_id": 1
}
```

Expected: `201 Created`. Repeating the same student and year must return `409 Conflict`.

### 3. Assign a subject and teacher

```http
POST /api/v1/subject-classes
Content-Type: application/json

{
  "subject_id": 1,
  "class_id": 1,
  "staff_id": 1,
  "academic_year_id": 1,
  "max_mark": 100
}
```

Expected: `201 Created`.

### 4. Schedule an exam

```http
POST /api/v1/exam-registrations
Content-Type: application/json

{
  "exam_id": 1,
  "academic_year_id": 1,
  "starts_on": "2026-10-01",
  "ends_on": "2026-10-07"
}
```

Dates must fall inside the actual academic-year dates in your database.

### 5. Enter a result

```http
POST /api/v1/exam-results
Content-Type: application/json

{
  "exam_registration_id": 1,
  "student_class_id": 1,
  "subject_class_id": 1,
  "marks": 85
}
```

Expected: `201 Created`. A mark greater than `max_mark`, duplicate result, different class, different year, different school, or unassigned Teacher must be rejected.

### 6. Correct a mark

```http
PATCH /api/v1/exam-results/1
Content-Type: application/json

{
  "marks": 88
}
```

The audit event records both the old and new mark.

## Verification commands

```bash
make test
make test-race
make test-integration
make swagger
make admin-verify
```

For production, start with a clean database, set `AUTO_MIGRATE=false`, run `make migrate-up`, and then start the API. Development databases created with `AUTO_MIGRATE=true` do not share SQL migration history and can show the SQL files as pending.
