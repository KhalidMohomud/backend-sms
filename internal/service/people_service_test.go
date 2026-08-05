package service

import (
	"backendapi/internal/authz"
	"backendapi/internal/model"
	"backendapi/internal/repository"
	"context"
	"testing"
	"time"
)

func TestPeopleServiceEnforcesSchoolScope(t *testing.T) {
	schoolID := uint64(8)
	service := newPeopleTestService(&peopleMemory{})
	actor := authz.Principal{SchoolID: &schoolID, Role: model.RoleRegistrar, Permissions: []string{model.PermissionManageStudents}}
	if _, err := service.ListStudents(context.Background(), actor, schoolID+1, repository.PeopleListOptions{}); err != ErrForbidden {
		t.Fatalf("cross-school ListStudents() error = %v, want ErrForbidden", err)
	}
}

func TestCreateStudentRejectsGuardianFromAnotherSchool(t *testing.T) {
	schoolID := uint64(8)
	memory := &peopleMemory{guardians: map[uint64]*model.Responsible{3: {ID: 3, SchoolID: schoolID + 1}}}
	service := newPeopleTestService(memory)
	actor := authz.Principal{UserID: 2, SchoolID: &schoolID, Role: model.RoleRegistrar, Permissions: []string{model.PermissionManageStudents}}
	_, err := service.CreateStudent(context.Background(), actor, schoolID, CreateStudentInput{
		Name: "Amina Ahmed", MotherName: "Hodan Ali", Sex: model.SexFemale,
		GuardianID: 3, RegisteredOn: todayUTC().Format("2006-01-02"),
	}, RequestMetadata{})
	if err != repository.ErrNotFound {
		t.Fatalf("CreateStudent(cross-school guardian) error = %v, want ErrNotFound", err)
	}
}

func TestArchiveStudentIsSoftDeleteAndAudited(t *testing.T) {
	schoolID := uint64(8)
	memory := &peopleMemory{students: map[uint64]*model.Student{4: {ID: 4, SchoolID: schoolID, Status: model.StudentStatusActive}}}
	audits := &memoryAudits{}
	service := NewPeopleService(memory, activeSchoolFoundation(schoolID), NewAuditWriter(audits))
	actor := authz.Principal{UserID: 2, SchoolID: &schoolID, Role: model.RoleRegistrar, Permissions: []string{model.PermissionManageStudents}}
	if err := service.ArchiveStudent(context.Background(), actor, schoolID, 4, RequestMetadata{}); err != nil {
		t.Fatal(err)
	}
	if memory.students[4].Status != model.StudentStatusDropped || audits.entries[len(audits.entries)-1].Action != "DELETE" {
		t.Fatalf("student = %#v, audits = %#v", memory.students[4], audits.entries)
	}
}

func TestCreateStaffAddsInitialActiveStatusAtomically(t *testing.T) {
	schoolID := uint64(8)
	memory := &peopleMemory{
		jobs:        map[uint64]*model.Job{1: {ID: 1, Status: model.RecordStatusActive}},
		decrees:     map[uint64]*model.Decree{2: {ID: 2, Status: model.RecordStatusActive}},
		statusTypes: map[uint64]*model.StaffStatusType{3: {ID: 3, Name: "Active", Status: model.RecordStatusActive}},
	}
	audits := &memoryAudits{}
	service := NewPeopleService(memory, activeSchoolFoundation(schoolID), NewAuditWriter(audits))
	actor := authz.Principal{UserID: 2, SchoolID: &schoolID, Role: model.RoleSchoolAdmin, Permissions: []string{model.PermissionManageStaff}}
	staff, err := service.CreateStaff(context.Background(), actor, schoolID, CreateStaffInput{
		Name: "Mohamed Ali", Sex: model.SexMale, JobID: 1, DecreeID: 2, Salary: 500, HiredDate: todayUTC().Format("2006-01-02"),
	}, RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	if staff.ID == 0 || len(memory.statuses) != 1 || memory.statuses[0].StaffStatusTypeID != 3 || len(audits.entries) != 2 {
		t.Fatalf("staff = %#v, statuses = %#v, audits = %#v", staff, memory.statuses, audits.entries)
	}
}

func TestDeleteReferencedAddressReturnsConflict(t *testing.T) {
	schoolID := uint64(8)
	memory := &peopleMemory{addresses: map[uint64]*model.Address{5: {ID: 5, SchoolID: schoolID}}, addressReferences: 1}
	service := newPeopleTestService(memory)
	actor := authz.Principal{SchoolID: &schoolID, Role: model.RoleSchoolAdmin, Permissions: []string{model.PermissionManageStaff}}
	if err := service.DeleteAddress(context.Background(), actor, schoolID, 5, RequestMetadata{}); err != ErrConflict {
		t.Fatalf("DeleteAddress() error = %v, want ErrConflict", err)
	}
}

func TestCreateStudentRejectsFutureRegistrationDate(t *testing.T) {
	schoolID := uint64(8)
	service := newPeopleTestService(&peopleMemory{})
	actor := authz.Principal{SchoolID: &schoolID, Permissions: []string{model.PermissionManageStudents}}

	_, err := service.CreateStudent(context.Background(), actor, schoolID, CreateStudentInput{
		Name:         "Amina Ahmed",
		MotherName:   "Hodan Ali",
		Sex:          model.SexFemale,
		GuardianID:   1,
		RegisteredOn: todayUTC().AddDate(0, 0, 1).Format(time.DateOnly),
	}, RequestMetadata{})
	if err != ErrInvalidDate {
		t.Fatalf("CreateStudent(future registration date) error = %v, want ErrInvalidDate", err)
	}
}

func TestUpdateStudentCanClearAddress(t *testing.T) {
	schoolID := uint64(8)
	addressID := uint64(5)
	memory := &peopleMemory{students: map[uint64]*model.Student{
		4: {ID: 4, SchoolID: schoolID, AddressID: &addressID, ResponsibleID: 1},
	}}
	service := newPeopleTestService(memory)
	actor := authz.Principal{SchoolID: &schoolID, Permissions: []string{model.PermissionManageStudents}}

	student, err := service.UpdateStudent(context.Background(), actor, schoolID, 4, UpdateStudentInput{ClearAddress: true}, RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	if student.AddressID != nil {
		t.Fatalf("UpdateStudent(clear address) address_id = %v, want nil", student.AddressID)
	}
}

func activeSchoolFoundation(schoolID uint64) *memoryFoundation {
	return &memoryFoundation{schools: map[uint64]*model.School{schoolID: {ID: schoolID, Status: model.SchoolStatusActive}}}
}

func newPeopleTestService(memory *peopleMemory) *PeopleService {
	audits := &memoryAudits{}
	return NewPeopleService(memory, activeSchoolFoundation(8), NewAuditWriter(audits))
}

type peopleMemory struct {
	addresses         map[uint64]*model.Address
	guardians         map[uint64]*model.Responsible
	students          map[uint64]*model.Student
	staff             map[uint64]*model.Staff
	statuses          []model.StaffStatus
	jobs              map[uint64]*model.Job
	decrees           map[uint64]*model.Decree
	statusTypes       map[uint64]*model.StaffStatusType
	addressReferences int64
}

func (m *peopleMemory) CreateAddress(_ context.Context, row *model.Address) error {
	row.ID = 1
	return nil
}
func (m *peopleMemory) ListAddresses(context.Context, uint64, repository.PeopleListOptions) ([]model.Address, error) {
	return nil, nil
}
func (m *peopleMemory) FindAddressByID(_ context.Context, schoolID, id uint64) (*model.Address, error) {
	row := m.addresses[id]
	if row == nil || row.SchoolID != schoolID {
		return nil, repository.ErrNotFound
	}
	copy := *row
	return &copy, nil
}
func (m *peopleMemory) UpdateAddress(context.Context, *model.Address) error { return nil }
func (m *peopleMemory) DeleteAddress(context.Context, uint64, uint64) error { return nil }
func (m *peopleMemory) CountAddressReferences(context.Context, uint64, uint64) (int64, error) {
	return m.addressReferences, nil
}
func (m *peopleMemory) CreateResponsible(_ context.Context, row *model.Responsible) error {
	row.ID = 1
	return nil
}
func (m *peopleMemory) ListResponsibles(context.Context, uint64, repository.PeopleListOptions) ([]model.Responsible, error) {
	return nil, nil
}
func (m *peopleMemory) FindResponsibleByID(_ context.Context, schoolID, id uint64) (*model.Responsible, error) {
	row := m.guardians[id]
	if row == nil || row.SchoolID != schoolID {
		return nil, repository.ErrNotFound
	}
	copy := *row
	return &copy, nil
}
func (m *peopleMemory) UpdateResponsible(context.Context, *model.Responsible) error { return nil }
func (m *peopleMemory) DeleteResponsible(context.Context, uint64, uint64) error     { return nil }
func (m *peopleMemory) CountResponsibleReferences(context.Context, uint64, uint64) (int64, error) {
	return 0, nil
}
func (m *peopleMemory) CreateStudent(_ context.Context, row *model.Student) error {
	row.ID = 1
	if m.students == nil {
		m.students = map[uint64]*model.Student{}
	}
	m.students[row.ID] = row
	return nil
}
func (m *peopleMemory) ListStudents(context.Context, uint64, repository.PeopleListOptions) ([]model.Student, error) {
	return nil, nil
}
func (m *peopleMemory) FindStudentByID(_ context.Context, schoolID, id uint64) (*model.Student, error) {
	row := m.students[id]
	if row == nil || row.SchoolID != schoolID {
		return nil, repository.ErrNotFound
	}
	copy := *row
	return &copy, nil
}
func (m *peopleMemory) UpdateStudent(_ context.Context, row *model.Student) error {
	copy := *row
	m.students[row.ID] = &copy
	return nil
}
func (m *peopleMemory) CreateStaff(_ context.Context, row *model.Staff) error {
	row.ID = 1
	if m.staff == nil {
		m.staff = map[uint64]*model.Staff{}
	}
	copy := *row
	m.staff[row.ID] = &copy
	return nil
}
func (m *peopleMemory) ListStaff(context.Context, uint64, repository.PeopleListOptions) ([]model.Staff, error) {
	return nil, nil
}
func (m *peopleMemory) FindStaffByID(_ context.Context, schoolID, id uint64) (*model.Staff, error) {
	row := m.staff[id]
	if row == nil || row.SchoolID != schoolID {
		return nil, repository.ErrNotFound
	}
	copy := *row
	return &copy, nil
}
func (m *peopleMemory) UpdateStaff(context.Context, *model.Staff) error { return nil }
func (m *peopleMemory) FindJobByID(_ context.Context, id uint64) (*model.Job, error) {
	row := m.jobs[id]
	if row == nil {
		return nil, repository.ErrNotFound
	}
	copy := *row
	return &copy, nil
}
func (m *peopleMemory) FindDecreeByID(_ context.Context, id uint64) (*model.Decree, error) {
	row := m.decrees[id]
	if row == nil {
		return nil, repository.ErrNotFound
	}
	copy := *row
	return &copy, nil
}
func (m *peopleMemory) CreateStaffStatus(_ context.Context, row *model.StaffStatus) error {
	row.ID = uint64(len(m.statuses) + 1)
	m.statuses = append(m.statuses, *row)
	return nil
}
func (m *peopleMemory) ListStaffStatuses(context.Context, uint64, uint64) ([]model.StaffStatus, error) {
	return m.statuses, nil
}
func (m *peopleMemory) FindStaffStatusTypeByID(_ context.Context, id uint64) (*model.StaffStatusType, error) {
	row := m.statusTypes[id]
	if row == nil {
		return nil, repository.ErrNotFound
	}
	copy := *row
	return &copy, nil
}
func (m *peopleMemory) FindStaffStatusTypeByName(_ context.Context, name string) (*model.StaffStatusType, error) {
	for _, row := range m.statusTypes {
		if row.Name == name {
			copy := *row
			return &copy, nil
		}
	}
	return nil, repository.ErrNotFound
}
func (m *peopleMemory) FindLatestStaffStatus(context.Context, uint64, uint64) (*model.StaffStatus, error) {
	if len(m.statuses) == 0 {
		return nil, repository.ErrNotFound
	}
	row := m.statuses[len(m.statuses)-1]
	return &row, nil
}

var _ repository.PeopleRepository = (*peopleMemory)(nil)
