package service

import (
	"backendapi/internal/authz"
	"backendapi/internal/model"
	"backendapi/internal/repository"
	"context"
	"testing"
	"time"
)

func TestUpdateAndArchiveSchool(t *testing.T) {
	repo := &memoryFoundation{schools: map[uint64]*model.School{
		8: {ID: 8, Name: "Old name", Status: model.SchoolStatusActive},
	}}
	audits := &memoryAudits{}
	service := NewFoundationService(repo, &memoryUsers{}, audits, NewAuditWriter(audits), newMemorySessions())
	actor := authz.Principal{UserID: 1, Role: model.RoleSuperAdmin}
	newName := "  New name  "

	school, err := service.UpdateSchool(context.Background(), actor, 8, UpdateSchoolInput{Name: &newName}, RequestMetadata{})
	if err != nil {
		t.Fatalf("UpdateSchool() error = %v", err)
	}
	if school.Name != "New name" || audits.entries[len(audits.entries)-1].Action != "UPDATE" {
		t.Fatalf("unexpected school update: %#v, audits: %#v", school, audits.entries)
	}

	if err := service.ArchiveSchool(context.Background(), actor, 8, RequestMetadata{}); err != nil {
		t.Fatalf("ArchiveSchool() error = %v", err)
	}
	if repo.schools[8].Status != model.SchoolStatusInactive || audits.entries[len(audits.entries)-1].Action != "DELETE" {
		t.Fatalf("school was not safely deleted: %#v, audits: %#v", repo.schools[8], audits.entries)
	}
}

func TestUpdateAndDeleteAcademicYearAreSchoolScoped(t *testing.T) {
	schoolID := uint64(8)
	start := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2027, 6, 30, 0, 0, 0, 0, time.UTC)
	repo := &memoryFoundation{
		schools: map[uint64]*model.School{schoolID: {ID: schoolID, Status: model.SchoolStatusActive}},
		years:   map[uint64]*model.AcademicYear{4: {ID: 4, SchoolID: schoolID, YearName: "2026", StartsOn: start, EndsOn: end}},
	}
	audits := &memoryAudits{}
	service := NewFoundationService(repo, &memoryUsers{}, audits, NewAuditWriter(audits), newMemorySessions())
	actor := authz.Principal{
		UserID: 2, SchoolID: &schoolID, Role: model.RoleSchoolAdmin,
		Permissions: []string{model.PermissionManageAcademicYears},
	}
	newName := "2026-2027"

	year, err := service.UpdateAcademicYear(context.Background(), actor, schoolID, 4, UpdateAcademicYearInput{YearName: &newName}, RequestMetadata{})
	if err != nil || year.YearName != newName {
		t.Fatalf("UpdateAcademicYear() = %#v, %v", year, err)
	}
	if err := service.DeleteAcademicYear(context.Background(), actor, schoolID+1, 4, RequestMetadata{}); err != ErrInvalidScope {
		t.Fatalf("cross-school DeleteAcademicYear() error = %v, want ErrInvalidScope", err)
	}
	if err := service.DeleteAcademicYear(context.Background(), actor, schoolID, 4, RequestMetadata{}); err != nil {
		t.Fatalf("DeleteAcademicYear() error = %v", err)
	}
	if _, exists := repo.years[4]; exists || audits.entries[len(audits.entries)-1].Action != "DELETE" {
		t.Fatalf("academic year was not deleted or audited: %#v, audits: %#v", repo.years, audits.entries)
	}
}

func TestDeleteUserDisablesAccount(t *testing.T) {
	users := &memoryUsers{user: &model.User{ID: 9, Status: model.UserStatusActive, Role: model.Role{Name: model.RoleRegistrar}}}
	audits := &memoryAudits{}
	service := NewFoundationService(&memoryFoundation{}, users, audits, NewAuditWriter(audits), newMemorySessions())
	actor := authz.Principal{UserID: 1, Role: model.RoleSuperAdmin}
	username := "  Updated.User  "

	updated, err := service.UpdateUser(context.Background(), actor, 9, UpdateUserInput{Username: &username}, RequestMetadata{})
	if err != nil || updated.Username != "updated.user" {
		t.Fatalf("UpdateUser() = %#v, %v", updated, err)
	}

	if err := service.DisableUser(context.Background(), actor, 9, RequestMetadata{}); err != nil {
		t.Fatalf("DisableUser() error = %v", err)
	}
	if users.user.Status != model.UserStatusDisabled || audits.entries[len(audits.entries)-1].Action != "DELETE" {
		t.Fatalf("user was not safely deleted: %#v, audits: %#v", users.user, audits.entries)
	}
}

func TestRoleManagementProtectsSuperAdminAndAuditsAssignments(t *testing.T) {
	repo := &memoryFoundation{
		roles:       map[uint64]*model.Role{1: {ID: 1, Name: model.RoleSuperAdmin, Status: model.RoleStatusActive, IsSystem: true}},
		permissions: map[uint64]model.Permission{3: {ID: 3, Name: model.PermissionManageUsers}},
	}
	audits := &memoryAudits{}
	service := NewFoundationService(repo, &memoryUsers{}, audits, NewAuditWriter(audits), newMemorySessions())
	actor := authz.Principal{UserID: 1, Role: model.RoleSuperAdmin}
	name := "Changed"

	if _, err := service.UpdateRole(context.Background(), actor, 1, UpdateRoleInput{Name: &name}, RequestMetadata{}); err != ErrProtectedRecord {
		t.Fatalf("UpdateRole(SuperAdmin) error = %v, want ErrProtectedRecord", err)
	}
	role, err := service.CreateRole(context.Background(), actor, CreateRoleInput{Name: "Counselor", PermissionIDs: []uint64{3}}, RequestMetadata{})
	if err != nil {
		t.Fatalf("CreateRole() error = %v", err)
	}
	if role.Name != "Counselor" || len(role.Permissions) != 1 || audits.entries[len(audits.entries)-1].ResourceType != "roles" {
		t.Fatalf("unexpected custom role: %#v, audits: %#v", role, audits.entries)
	}
	if _, err := service.CreateRole(context.Background(), authz.Principal{}, CreateRoleInput{Name: "Forbidden"}, RequestMetadata{}); err != ErrForbidden {
		t.Fatalf("non-SuperAdmin CreateRole() error = %v, want ErrForbidden", err)
	}
}

type memoryFoundation struct {
	schools     map[uint64]*model.School
	years       map[uint64]*model.AcademicYear
	roles       map[uint64]*model.Role
	permissions map[uint64]model.Permission
}

func (m *memoryFoundation) CreateSchool(_ context.Context, school *model.School) error {
	if m.schools == nil {
		m.schools = make(map[uint64]*model.School)
	}
	m.schools[school.ID] = school
	return nil
}

func (m *memoryFoundation) ListSchools(context.Context) ([]model.School, error) { return nil, nil }

func (m *memoryFoundation) FindSchoolByID(_ context.Context, id uint64) (*model.School, error) {
	school, exists := m.schools[id]
	if !exists {
		return nil, repository.ErrNotFound
	}
	copy := *school
	return &copy, nil
}

func (m *memoryFoundation) UpdateSchool(_ context.Context, school *model.School) error {
	if _, exists := m.schools[school.ID]; !exists {
		return repository.ErrNotFound
	}
	copy := *school
	m.schools[school.ID] = &copy
	return nil
}

func (m *memoryFoundation) CreateAcademicYear(_ context.Context, year *model.AcademicYear) error {
	if m.years == nil {
		m.years = make(map[uint64]*model.AcademicYear)
	}
	m.years[year.ID] = year
	return nil
}

func (m *memoryFoundation) ListAcademicYears(context.Context, uint64) ([]model.AcademicYear, error) {
	return nil, nil
}

func (m *memoryFoundation) FindAcademicYearByID(_ context.Context, schoolID, id uint64) (*model.AcademicYear, error) {
	year, exists := m.years[id]
	if !exists || year.SchoolID != schoolID {
		return nil, repository.ErrNotFound
	}
	copy := *year
	return &copy, nil
}

func (m *memoryFoundation) UpdateAcademicYear(_ context.Context, year *model.AcademicYear) error {
	existing, exists := m.years[year.ID]
	if !exists || existing.SchoolID != year.SchoolID {
		return repository.ErrNotFound
	}
	copy := *year
	m.years[year.ID] = &copy
	return nil
}

func (m *memoryFoundation) DeleteAcademicYear(_ context.Context, schoolID, id uint64) error {
	year, exists := m.years[id]
	if !exists || year.SchoolID != schoolID {
		return repository.ErrNotFound
	}
	delete(m.years, id)
	return nil
}

func (m *memoryFoundation) FindRoleByID(_ context.Context, id uint64) (*model.Role, error) {
	role, ok := m.roles[id]
	if !ok {
		return nil, repository.ErrNotFound
	}
	copy := *role
	return &copy, nil
}

func (m *memoryFoundation) FindRoleByName(context.Context, string) (*model.Role, error) {
	return nil, repository.ErrNotFound
}

func (m *memoryFoundation) ListRoles(context.Context) ([]model.Role, error) { return nil, nil }

func (m *memoryFoundation) CreateRole(_ context.Context, role *model.Role) error {
	if m.roles == nil {
		m.roles = make(map[uint64]*model.Role)
	}
	if role.ID == 0 {
		role.ID = uint64(len(m.roles) + 1)
	}
	copy := *role
	m.roles[role.ID] = &copy
	return nil
}

func (m *memoryFoundation) UpdateRole(_ context.Context, role *model.Role) error {
	if _, ok := m.roles[role.ID]; !ok {
		return repository.ErrNotFound
	}
	copy := *role
	m.roles[role.ID] = &copy
	return nil
}

func (m *memoryFoundation) ReplaceRolePermissions(_ context.Context, roleID uint64, permissions []model.Permission) error {
	role, ok := m.roles[roleID]
	if !ok {
		return repository.ErrNotFound
	}
	role.Permissions = permissions
	return nil
}

func (m *memoryFoundation) FindPermissionsByIDs(_ context.Context, ids []uint64) ([]model.Permission, error) {
	result := make([]model.Permission, 0, len(ids))
	for _, id := range ids {
		permission, ok := m.permissions[id]
		if !ok {
			return nil, repository.ErrNotFound
		}
		result = append(result, permission)
	}
	return result, nil
}

func (m *memoryFoundation) ListPermissions(context.Context) ([]model.Permission, error) {
	return nil, nil
}

func (m *memoryFoundation) ListUsers(context.Context, *uint64) ([]model.User, error) {
	return nil, nil
}
