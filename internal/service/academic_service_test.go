package service

import (
	"backendapi/internal/authz"
	"backendapi/internal/model"
	"backendapi/internal/repository"
	"context"
	"errors"
	"testing"
	"time"
)

func TestAcademicServiceEnforcesSchoolScope(t *testing.T) {
	schoolID := uint64(8)
	service := newAcademicTestService(&academicMemory{})
	actor := authz.Principal{SchoolID: &schoolID, Permissions: []string{model.PermissionViewResults}}
	if _, err := service.ListStudentClasses(context.Background(), actor, schoolID+1, repository.AcademicListOptions{}); err != ErrForbidden {
		t.Fatalf("ListStudentClasses(cross-school) error = %v, want ErrForbidden", err)
	}
}

func TestCreateStudentClassRejectsInactiveStudent(t *testing.T) {
	schoolID := uint64(8)
	memory := validAcademicMemory(schoolID)
	memory.students[1].Status = model.StudentStatusDropped
	service := newAcademicTestService(memory)
	actor := academicActor(schoolID, model.PermissionManageEnrollments)
	_, err := service.CreateStudentClass(context.Background(), actor, schoolID, CreateStudentClassInput{StudentID: 1, ClassID: 2, AcademicYearID: 3}, RequestMetadata{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("CreateStudentClass(inactive student) error = %v, want ErrInvalidInput", err)
	}
}

func TestCreateExamRegistrationRejectsDatesOutsideAcademicYear(t *testing.T) {
	schoolID := uint64(8)
	memory := validAcademicMemory(schoolID)
	service := newAcademicTestService(memory)
	actor := academicActor(schoolID, model.PermissionManageExams)
	_, err := service.CreateExamRegistration(context.Background(), actor, schoolID, CreateExamRegistrationInput{
		ExamID: 4, AcademicYearID: 3, StartsOn: "2027-07-01", EndsOn: "2027-07-05",
	}, RequestMetadata{})
	if !errors.Is(err, ErrInvalidDate) {
		t.Fatalf("CreateExamRegistration(outside year) error = %v, want ErrInvalidDate", err)
	}
}

func TestTeacherCannotEnterMarksForAnotherTeacherAssignment(t *testing.T) {
	schoolID := uint64(8)
	memory := validAcademicMemory(schoolID)
	service := newAcademicTestService(memory)
	wrongStaffID := uint64(99)
	actor := authz.Principal{UserID: 10, SchoolID: &schoolID, StaffID: &wrongStaffID, Role: model.RoleTeacher, Permissions: []string{model.PermissionEnterMarks}}
	_, err := service.CreateExamResult(context.Background(), actor, schoolID, CreateExamResultInput{
		ExamRegistrationID: 7, StudentClassID: 6, SubjectClassID: 5, Marks: 80,
	}, RequestMetadata{})
	if err != ErrForbidden {
		t.Fatalf("CreateExamResult(other teacher) error = %v, want ErrForbidden", err)
	}
}

func TestCreateExamResultRejectsMarksAboveMaximum(t *testing.T) {
	schoolID := uint64(8)
	memory := validAcademicMemory(schoolID)
	service := newAcademicTestService(memory)
	actor := academicActor(schoolID, model.PermissionEnterMarks)
	_, err := service.CreateExamResult(context.Background(), actor, schoolID, CreateExamResultInput{
		ExamRegistrationID: 7, StudentClassID: 6, SubjectClassID: 5, Marks: 101,
	}, RequestMetadata{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("CreateExamResult(excess marks) error = %v, want ErrInvalidInput", err)
	}
}

func TestCreateExamResultRejectsMismatchedClass(t *testing.T) {
	schoolID := uint64(8)
	memory := validAcademicMemory(schoolID)
	memory.subjectClasses[5].ClassID = 99
	service := newAcademicTestService(memory)
	actor := academicActor(schoolID, model.PermissionEnterMarks)
	_, err := service.CreateExamResult(context.Background(), actor, schoolID, CreateExamResultInput{
		ExamRegistrationID: 7, StudentClassID: 6, SubjectClassID: 5, Marks: 80,
	}, RequestMetadata{})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("CreateExamResult(mismatched class) error = %v, want ErrInvalidInput", err)
	}
}

func TestTeacherCreatesAssignedExamResultAndAudit(t *testing.T) {
	schoolID := uint64(8)
	memory := validAcademicMemory(schoolID)
	audits := &memoryAudits{}
	service := NewAcademicService(memory, activeSchoolFoundation(schoolID), NewAuditWriter(audits))
	staffID := uint64(9)
	actor := authz.Principal{UserID: 10, SchoolID: &schoolID, StaffID: &staffID, Role: model.RoleTeacher, Permissions: []string{model.PermissionEnterMarks}}
	row, err := service.CreateExamResult(context.Background(), actor, schoolID, CreateExamResultInput{
		ExamRegistrationID: 7, StudentClassID: 6, SubjectClassID: 5, Marks: 85,
	}, RequestMetadata{})
	if err != nil {
		t.Fatal(err)
	}
	if row.ID == 0 || row.RecordedBy != actor.UserID || len(audits.entries) != 1 || audits.entries[0].ResourceType != "exam_results" {
		t.Fatalf("result = %#v, audits = %#v", row, audits.entries)
	}
}

func TestSubjectAssignmentWithResultsCannotBeChanged(t *testing.T) {
	schoolID := uint64(8)
	memory := validAcademicMemory(schoolID)
	memory.subjectClassResultCount = 1
	service := newAcademicTestService(memory)
	actor := academicActor(schoolID, model.PermissionManageAssignments)
	maxMark := 90.0
	_, err := service.UpdateSubjectClass(context.Background(), actor, schoolID, 5, UpdateSubjectClassInput{MaxMark: &maxMark}, RequestMetadata{})
	if err != ErrConflict {
		t.Fatalf("UpdateSubjectClass(with results) error = %v, want ErrConflict", err)
	}
}

func academicActor(schoolID uint64, permission string) authz.Principal {
	return authz.Principal{UserID: 10, SchoolID: &schoolID, Role: model.RoleSchoolAdmin, Permissions: []string{permission}}
}

func newAcademicTestService(memory *academicMemory) *AcademicService {
	return NewAcademicService(memory, activeSchoolFoundation(8), NewAuditWriter(&memoryAudits{}))
}

func validAcademicMemory(schoolID uint64) *academicMemory {
	starts, _ := time.Parse(time.DateOnly, "2026-09-01")
	ends, _ := time.Parse(time.DateOnly, "2027-06-30")
	return &academicMemory{
		students: map[uint64]*model.Student{1: {ID: 1, SchoolID: schoolID, Status: model.StudentStatusActive}},
		classes:  map[uint64]*model.Class{2: {ID: 2, SchoolID: schoolID, Status: model.RecordStatusActive}},
		years:    map[uint64]*model.AcademicYear{3: {ID: 3, SchoolID: schoolID, StartsOn: starts, EndsOn: ends}},
		exams:    map[uint64]*model.Exam{4: {ID: 4, Status: model.RecordStatusActive}},
		subjects: map[uint64]*model.Subject{4: {ID: 4, Status: model.RecordStatusActive}},
		staff:    map[uint64]*model.Staff{9: {ID: 9, SchoolID: schoolID}},
		staffStatuses: map[uint64]*model.StaffStatus{9: {
			ID: 1, SchoolID: schoolID, StaffID: 9, StatusType: model.StaffStatusType{Name: "Active"},
		}},
		studentClasses:    map[uint64]*model.StudentClass{6: {ID: 6, SchoolID: schoolID, StudentID: 1, ClassID: 2, AcademicYearID: 3}},
		subjectClasses:    map[uint64]*model.SubjectClass{5: {ID: 5, SchoolID: schoolID, SubjectID: 4, ClassID: 2, StaffID: 9, AcademicYearID: 3, MaxMark: 100}},
		examRegistrations: map[uint64]*model.ExamRegistration{7: {ID: 7, SchoolID: schoolID, ExamID: 4, AcademicYearID: 3, StartsOn: starts, EndsOn: starts.AddDate(0, 0, 5)}},
		examResults:       map[uint64]*model.ExamResult{},
	}
}

type academicMemory struct {
	students                map[uint64]*model.Student
	classes                 map[uint64]*model.Class
	years                   map[uint64]*model.AcademicYear
	staff                   map[uint64]*model.Staff
	staffStatuses           map[uint64]*model.StaffStatus
	subjects                map[uint64]*model.Subject
	exams                   map[uint64]*model.Exam
	studentClasses          map[uint64]*model.StudentClass
	subjectClasses          map[uint64]*model.SubjectClass
	examRegistrations       map[uint64]*model.ExamRegistration
	examResults             map[uint64]*model.ExamResult
	studentClassResultCount int64
	subjectClassResultCount int64
	examRegistrationResults int64
}

func (m *academicMemory) CreateStudentClass(_ context.Context, row *model.StudentClass) error {
	row.ID = uint64(len(m.studentClasses) + 10)
	if m.studentClasses == nil {
		m.studentClasses = map[uint64]*model.StudentClass{}
	}
	m.studentClasses[row.ID] = row
	return nil
}
func (m *academicMemory) ListStudentClasses(context.Context, uint64, repository.AcademicListOptions) ([]model.StudentClass, error) {
	return nil, nil
}
func (m *academicMemory) FindStudentClassByID(_ context.Context, schoolID, id uint64) (*model.StudentClass, error) {
	row := m.studentClasses[id]
	if row == nil || row.SchoolID != schoolID {
		return nil, repository.ErrNotFound
	}
	copy := *row
	return &copy, nil
}
func (m *academicMemory) UpdateStudentClass(_ context.Context, row *model.StudentClass) error {
	copy := *row
	m.studentClasses[row.ID] = &copy
	return nil
}
func (m *academicMemory) DeleteStudentClass(context.Context, uint64, uint64) error { return nil }
func (m *academicMemory) CreateSubjectClass(_ context.Context, row *model.SubjectClass) error {
	row.ID = uint64(len(m.subjectClasses) + 10)
	if m.subjectClasses == nil {
		m.subjectClasses = map[uint64]*model.SubjectClass{}
	}
	m.subjectClasses[row.ID] = row
	return nil
}
func (m *academicMemory) ListSubjectClasses(_ context.Context, schoolID uint64, options repository.AcademicListOptions) ([]model.SubjectClass, error) {
	var rows []model.SubjectClass
	for _, row := range m.subjectClasses {
		if row.SchoolID == schoolID && (options.ClassID == nil || row.ClassID == *options.ClassID) && (options.AcademicYearID == nil || row.AcademicYearID == *options.AcademicYearID) && (options.StaffID == nil || row.StaffID == *options.StaffID) {
			rows = append(rows, *row)
		}
	}
	return rows, nil
}
func (m *academicMemory) FindSubjectClassByID(_ context.Context, schoolID, id uint64) (*model.SubjectClass, error) {
	row := m.subjectClasses[id]
	if row == nil || row.SchoolID != schoolID {
		return nil, repository.ErrNotFound
	}
	copy := *row
	return &copy, nil
}
func (m *academicMemory) UpdateSubjectClass(_ context.Context, row *model.SubjectClass) error {
	copy := *row
	m.subjectClasses[row.ID] = &copy
	return nil
}
func (m *academicMemory) DeleteSubjectClass(context.Context, uint64, uint64) error { return nil }
func (m *academicMemory) CreateExamRegistration(_ context.Context, row *model.ExamRegistration) error {
	row.ID = uint64(len(m.examRegistrations) + 10)
	if m.examRegistrations == nil {
		m.examRegistrations = map[uint64]*model.ExamRegistration{}
	}
	m.examRegistrations[row.ID] = row
	return nil
}
func (m *academicMemory) ListExamRegistrations(context.Context, uint64, repository.AcademicListOptions) ([]model.ExamRegistration, error) {
	return nil, nil
}
func (m *academicMemory) FindExamRegistrationByID(_ context.Context, schoolID, id uint64) (*model.ExamRegistration, error) {
	row := m.examRegistrations[id]
	if row == nil || row.SchoolID != schoolID {
		return nil, repository.ErrNotFound
	}
	copy := *row
	return &copy, nil
}
func (m *academicMemory) UpdateExamRegistration(_ context.Context, row *model.ExamRegistration) error {
	copy := *row
	m.examRegistrations[row.ID] = &copy
	return nil
}
func (m *academicMemory) DeleteExamRegistration(context.Context, uint64, uint64) error { return nil }
func (m *academicMemory) CreateExamResult(_ context.Context, row *model.ExamResult) error {
	row.ID = uint64(len(m.examResults) + 10)
	if m.examResults == nil {
		m.examResults = map[uint64]*model.ExamResult{}
	}
	copy := *row
	m.examResults[row.ID] = &copy
	return nil
}
func (m *academicMemory) ListExamResults(context.Context, uint64, repository.AcademicListOptions) ([]model.ExamResult, error) {
	return nil, nil
}
func (m *academicMemory) FindExamResultByID(_ context.Context, schoolID, id uint64) (*model.ExamResult, error) {
	row := m.examResults[id]
	if row == nil || row.SchoolID != schoolID {
		return nil, repository.ErrNotFound
	}
	copy := *row
	if assignment := m.subjectClasses[copy.SubjectClassID]; assignment != nil {
		copy.SubjectClass = *assignment
	}
	return &copy, nil
}
func (m *academicMemory) UpdateExamResult(_ context.Context, row *model.ExamResult) error {
	copy := *row
	m.examResults[row.ID] = &copy
	return nil
}
func (m *academicMemory) DeleteExamResult(context.Context, uint64, uint64) error { return nil }
func (m *academicMemory) CountResultsByStudentClass(context.Context, uint64, uint64) (int64, error) {
	return m.studentClassResultCount, nil
}
func (m *academicMemory) CountResultsBySubjectClass(context.Context, uint64, uint64) (int64, error) {
	return m.subjectClassResultCount, nil
}
func (m *academicMemory) CountResultsByExamRegistration(context.Context, uint64, uint64) (int64, error) {
	return m.examRegistrationResults, nil
}
func (m *academicMemory) FindStudentReference(_ context.Context, schoolID, id uint64) (*model.Student, error) {
	row := m.students[id]
	if row == nil || row.SchoolID != schoolID {
		return nil, repository.ErrNotFound
	}
	copy := *row
	return &copy, nil
}
func (m *academicMemory) FindClassReference(_ context.Context, schoolID, id uint64) (*model.Class, error) {
	row := m.classes[id]
	if row == nil || row.SchoolID != schoolID {
		return nil, repository.ErrNotFound
	}
	copy := *row
	return &copy, nil
}
func (m *academicMemory) FindAcademicYearReference(_ context.Context, schoolID, id uint64) (*model.AcademicYear, error) {
	row := m.years[id]
	if row == nil || row.SchoolID != schoolID {
		return nil, repository.ErrNotFound
	}
	copy := *row
	return &copy, nil
}
func (m *academicMemory) FindStaffReference(_ context.Context, schoolID, id uint64) (*model.Staff, error) {
	row := m.staff[id]
	if row == nil || row.SchoolID != schoolID {
		return nil, repository.ErrNotFound
	}
	copy := *row
	return &copy, nil
}
func (m *academicMemory) FindLatestStaffStatus(_ context.Context, schoolID, id uint64) (*model.StaffStatus, error) {
	row := m.staffStatuses[id]
	if row == nil || row.SchoolID != schoolID {
		return nil, repository.ErrNotFound
	}
	copy := *row
	return &copy, nil
}
func (m *academicMemory) FindSubjectReference(_ context.Context, id uint64) (*model.Subject, error) {
	row := m.subjects[id]
	if row == nil {
		return nil, repository.ErrNotFound
	}
	copy := *row
	return &copy, nil
}
func (m *academicMemory) FindExamReference(_ context.Context, id uint64) (*model.Exam, error) {
	row := m.exams[id]
	if row == nil {
		return nil, repository.ErrNotFound
	}
	copy := *row
	return &copy, nil
}

var _ repository.AcademicRepository = (*academicMemory)(nil)
