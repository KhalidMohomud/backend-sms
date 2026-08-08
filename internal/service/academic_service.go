package service

import (
	"backendapi/internal/authz"
	"backendapi/internal/model"
	"backendapi/internal/repository"
	"context"
	"errors"
	"fmt"
	"strings"

	"gorm.io/gorm"
)

type CreateStudentClassInput struct {
	StudentID      uint64 `json:"student_id" binding:"required,min=1" example:"1"`
	ClassID        uint64 `json:"class_id" binding:"required,min=1" example:"1"`
	AcademicYearID uint64 `json:"academic_year_id" binding:"required,min=1" example:"1"`
}

type UpdateStudentClassInput struct {
	StudentID      *uint64 `json:"student_id" binding:"omitempty,min=1"`
	ClassID        *uint64 `json:"class_id" binding:"omitempty,min=1"`
	AcademicYearID *uint64 `json:"academic_year_id" binding:"omitempty,min=1"`
}

type CreateSubjectClassInput struct {
	SubjectID      uint64  `json:"subject_id" binding:"required,min=1" example:"1"`
	ClassID        uint64  `json:"class_id" binding:"required,min=1" example:"1"`
	StaffID        uint64  `json:"staff_id" binding:"required,min=1" example:"1"`
	AcademicYearID uint64  `json:"academic_year_id" binding:"required,min=1" example:"1"`
	MaxMark        float64 `json:"max_mark" binding:"omitempty,gt=0,lte=999.99" example:"100"`
}

type UpdateSubjectClassInput struct {
	SubjectID      *uint64  `json:"subject_id" binding:"omitempty,min=1"`
	ClassID        *uint64  `json:"class_id" binding:"omitempty,min=1"`
	StaffID        *uint64  `json:"staff_id" binding:"omitempty,min=1"`
	AcademicYearID *uint64  `json:"academic_year_id" binding:"omitempty,min=1"`
	MaxMark        *float64 `json:"max_mark" binding:"omitempty,gt=0,lte=999.99"`
}

type CreateExamRegistrationInput struct {
	ExamID         uint64 `json:"exam_id" binding:"required,min=1" example:"1"`
	AcademicYearID uint64 `json:"academic_year_id" binding:"required,min=1" example:"1"`
	StartsOn       string `json:"starts_on" binding:"required" example:"2026-10-01"`
	EndsOn         string `json:"ends_on" binding:"required" example:"2026-10-07"`
}

type UpdateExamRegistrationInput struct {
	ExamID         *uint64 `json:"exam_id" binding:"omitempty,min=1"`
	AcademicYearID *uint64 `json:"academic_year_id" binding:"omitempty,min=1"`
	StartsOn       *string `json:"starts_on"`
	EndsOn         *string `json:"ends_on"`
}

type CreateExamResultInput struct {
	ExamRegistrationID uint64  `json:"exam_registration_id" binding:"required,min=1" example:"1"`
	StudentClassID     uint64  `json:"student_class_id" binding:"required,min=1" example:"1"`
	SubjectClassID     uint64  `json:"subject_class_id" binding:"required,min=1" example:"1"`
	Marks              float64 `json:"marks" binding:"gte=0,lte=999.99" example:"85"`
}

type UpdateExamResultInput struct {
	Marks *float64 `json:"marks" binding:"required,gte=0,lte=999.99" example:"88"`
}

type AcademicService struct {
	academic   repository.AcademicRepository
	foundation repository.FoundationRepository
	audit      *AuditWriter
}

func NewAcademicService(academic repository.AcademicRepository, foundation repository.FoundationRepository, audit *AuditWriter) *AcademicService {
	return &AcademicService{academic: academic, foundation: foundation, audit: audit}
}

func (s *AcademicService) ListStudentClasses(ctx context.Context, actor authz.Principal, schoolID uint64, options repository.AcademicListOptions) ([]model.StudentClass, error) {
	if !canReadAcademic(actor, schoolID) {
		return nil, ErrForbidden
	}
	if err := applyTeacherScope(actor, &options); err != nil {
		return nil, err
	}
	return s.academic.ListStudentClasses(ctx, schoolID, options)
}

func (s *AcademicService) GetStudentClass(ctx context.Context, actor authz.Principal, schoolID, id uint64) (*model.StudentClass, error) {
	if !canReadAcademic(actor, schoolID) {
		return nil, ErrForbidden
	}
	row, err := s.academic.FindStudentClassByID(ctx, schoolID, id)
	if err != nil {
		return nil, err
	}
	if err := s.requireTeacherClassAccess(ctx, actor, schoolID, row.ClassID, row.AcademicYearID); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *AcademicService) CreateStudentClass(ctx context.Context, actor authz.Principal, schoolID uint64, input CreateStudentClassInput, request RequestMetadata) (*model.StudentClass, error) {
	if !canManageEnrollments(actor, schoolID) {
		return nil, ErrForbidden
	}
	if err := s.requireActiveSchool(ctx, schoolID); err != nil {
		return nil, err
	}
	if err := s.validateEnrollmentReferences(ctx, schoolID, input.StudentID, input.ClassID, input.AcademicYearID); err != nil {
		return nil, err
	}
	row := &model.StudentClass{SchoolID: schoolID, StudentID: input.StudentID, ClassID: input.ClassID, AcademicYearID: input.AcademicYearID}
	if err := s.academic.CreateStudentClass(ctx, row); err != nil {
		return nil, academicWriteError("create enrollment", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "INSERT", "student_classes", &row.ID, request, nil); err != nil {
		return nil, err
	}
	return s.academic.FindStudentClassByID(ctx, schoolID, row.ID)
}

func (s *AcademicService) UpdateStudentClass(ctx context.Context, actor authz.Principal, schoolID, id uint64, input UpdateStudentClassInput, request RequestMetadata) (*model.StudentClass, error) {
	if !canManageEnrollments(actor, schoolID) {
		return nil, ErrForbidden
	}
	if input.StudentID == nil && input.ClassID == nil && input.AcademicYearID == nil {
		return nil, ErrNoChanges
	}
	row, err := s.academic.FindStudentClassByID(ctx, schoolID, id)
	if err != nil {
		return nil, err
	}
	count, err := s.academic.CountResultsByStudentClass(ctx, schoolID, id)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrConflict
	}
	if input.StudentID != nil {
		row.StudentID = *input.StudentID
	}
	if input.ClassID != nil {
		row.ClassID = *input.ClassID
	}
	if input.AcademicYearID != nil {
		row.AcademicYearID = *input.AcademicYearID
	}
	if err := s.validateEnrollmentReferences(ctx, schoolID, row.StudentID, row.ClassID, row.AcademicYearID); err != nil {
		return nil, err
	}
	if err := s.academic.UpdateStudentClass(ctx, row); err != nil {
		return nil, academicWriteError("update enrollment", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "UPDATE", "student_classes", &row.ID, request, nil); err != nil {
		return nil, err
	}
	return s.academic.FindStudentClassByID(ctx, schoolID, row.ID)
}

func (s *AcademicService) DeleteStudentClass(ctx context.Context, actor authz.Principal, schoolID, id uint64, request RequestMetadata) error {
	if !canManageEnrollments(actor, schoolID) {
		return ErrForbidden
	}
	if _, err := s.academic.FindStudentClassByID(ctx, schoolID, id); err != nil {
		return err
	}
	count, err := s.academic.CountResultsByStudentClass(ctx, schoolID, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrConflict
	}
	if err := s.academic.DeleteStudentClass(ctx, schoolID, id); err != nil {
		return academicWriteError("delete enrollment", err)
	}
	return s.audit.Write(ctx, &actor.UserID, &schoolID, "DELETE", "student_classes", &id, request, nil)
}

func (s *AcademicService) ListSubjectClasses(ctx context.Context, actor authz.Principal, schoolID uint64, options repository.AcademicListOptions) ([]model.SubjectClass, error) {
	if !canReadAcademic(actor, schoolID) {
		return nil, ErrForbidden
	}
	if err := applyTeacherScope(actor, &options); err != nil {
		return nil, err
	}
	return s.academic.ListSubjectClasses(ctx, schoolID, options)
}

func (s *AcademicService) GetSubjectClass(ctx context.Context, actor authz.Principal, schoolID, id uint64) (*model.SubjectClass, error) {
	if !canReadAcademic(actor, schoolID) {
		return nil, ErrForbidden
	}
	row, err := s.academic.FindSubjectClassByID(ctx, schoolID, id)
	if err != nil {
		return nil, err
	}
	if actor.Role == model.RoleTeacher && (actor.StaffID == nil || row.StaffID != *actor.StaffID) {
		return nil, ErrForbidden
	}
	return row, nil
}

func (s *AcademicService) CreateSubjectClass(ctx context.Context, actor authz.Principal, schoolID uint64, input CreateSubjectClassInput, request RequestMetadata) (*model.SubjectClass, error) {
	if !canManageAssignments(actor, schoolID) {
		return nil, ErrForbidden
	}
	maxMark := input.MaxMark
	if maxMark == 0 {
		maxMark = 100
	}
	if err := s.validateAssignmentReferences(ctx, schoolID, input.SubjectID, input.ClassID, input.StaffID, input.AcademicYearID, maxMark); err != nil {
		return nil, err
	}
	row := &model.SubjectClass{SchoolID: schoolID, SubjectID: input.SubjectID, ClassID: input.ClassID, StaffID: input.StaffID, AcademicYearID: input.AcademicYearID, MaxMark: maxMark}
	if err := s.academic.CreateSubjectClass(ctx, row); err != nil {
		return nil, academicWriteError("create subject assignment", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "INSERT", "subject_classes", &row.ID, request, nil); err != nil {
		return nil, err
	}
	return s.academic.FindSubjectClassByID(ctx, schoolID, row.ID)
}

func (s *AcademicService) UpdateSubjectClass(ctx context.Context, actor authz.Principal, schoolID, id uint64, input UpdateSubjectClassInput, request RequestMetadata) (*model.SubjectClass, error) {
	if !canManageAssignments(actor, schoolID) {
		return nil, ErrForbidden
	}
	if input.SubjectID == nil && input.ClassID == nil && input.StaffID == nil && input.AcademicYearID == nil && input.MaxMark == nil {
		return nil, ErrNoChanges
	}
	row, err := s.academic.FindSubjectClassByID(ctx, schoolID, id)
	if err != nil {
		return nil, err
	}
	count, err := s.academic.CountResultsBySubjectClass(ctx, schoolID, id)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrConflict
	}
	if input.SubjectID != nil {
		row.SubjectID = *input.SubjectID
	}
	if input.ClassID != nil {
		row.ClassID = *input.ClassID
	}
	if input.StaffID != nil {
		row.StaffID = *input.StaffID
	}
	if input.AcademicYearID != nil {
		row.AcademicYearID = *input.AcademicYearID
	}
	if input.MaxMark != nil {
		row.MaxMark = *input.MaxMark
	}
	if err := s.validateAssignmentReferences(ctx, schoolID, row.SubjectID, row.ClassID, row.StaffID, row.AcademicYearID, row.MaxMark); err != nil {
		return nil, err
	}
	if err := s.academic.UpdateSubjectClass(ctx, row); err != nil {
		return nil, academicWriteError("update subject assignment", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "UPDATE", "subject_classes", &row.ID, request, nil); err != nil {
		return nil, err
	}
	return s.academic.FindSubjectClassByID(ctx, schoolID, row.ID)
}

func (s *AcademicService) DeleteSubjectClass(ctx context.Context, actor authz.Principal, schoolID, id uint64, request RequestMetadata) error {
	if !canManageAssignments(actor, schoolID) {
		return ErrForbidden
	}
	if _, err := s.academic.FindSubjectClassByID(ctx, schoolID, id); err != nil {
		return err
	}
	count, err := s.academic.CountResultsBySubjectClass(ctx, schoolID, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrConflict
	}
	if err := s.academic.DeleteSubjectClass(ctx, schoolID, id); err != nil {
		return academicWriteError("delete subject assignment", err)
	}
	return s.audit.Write(ctx, &actor.UserID, &schoolID, "DELETE", "subject_classes", &id, request, nil)
}

func (s *AcademicService) ListExamRegistrations(ctx context.Context, actor authz.Principal, schoolID uint64, options repository.AcademicListOptions) ([]model.ExamRegistration, error) {
	if !canReadAcademic(actor, schoolID) {
		return nil, ErrForbidden
	}
	return s.academic.ListExamRegistrations(ctx, schoolID, options)
}

func (s *AcademicService) GetExamRegistration(ctx context.Context, actor authz.Principal, schoolID, id uint64) (*model.ExamRegistration, error) {
	if !canReadAcademic(actor, schoolID) {
		return nil, ErrForbidden
	}
	return s.academic.FindExamRegistrationByID(ctx, schoolID, id)
}

func (s *AcademicService) CreateExamRegistration(ctx context.Context, actor authz.Principal, schoolID uint64, input CreateExamRegistrationInput, request RequestMetadata) (*model.ExamRegistration, error) {
	if !canManageExams(actor, schoolID) {
		return nil, ErrForbidden
	}
	row, err := s.buildExamRegistration(ctx, schoolID, input.ExamID, input.AcademicYearID, input.StartsOn, input.EndsOn)
	if err != nil {
		return nil, err
	}
	if err := s.academic.CreateExamRegistration(ctx, row); err != nil {
		return nil, academicWriteError("create exam registration", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "INSERT", "exam_registrations", &row.ID, request, nil); err != nil {
		return nil, err
	}
	return s.academic.FindExamRegistrationByID(ctx, schoolID, row.ID)
}

func (s *AcademicService) UpdateExamRegistration(ctx context.Context, actor authz.Principal, schoolID, id uint64, input UpdateExamRegistrationInput, request RequestMetadata) (*model.ExamRegistration, error) {
	if !canManageExams(actor, schoolID) {
		return nil, ErrForbidden
	}
	if input.ExamID == nil && input.AcademicYearID == nil && input.StartsOn == nil && input.EndsOn == nil {
		return nil, ErrNoChanges
	}
	row, err := s.academic.FindExamRegistrationByID(ctx, schoolID, id)
	if err != nil {
		return nil, err
	}
	count, err := s.academic.CountResultsByExamRegistration(ctx, schoolID, id)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, ErrConflict
	}
	if input.ExamID != nil {
		row.ExamID = *input.ExamID
	}
	if input.AcademicYearID != nil {
		row.AcademicYearID = *input.AcademicYearID
	}
	starts, ends := row.StartsOn.Format("2006-01-02"), row.EndsOn.Format("2006-01-02")
	if input.StartsOn != nil {
		starts = *input.StartsOn
	}
	if input.EndsOn != nil {
		ends = *input.EndsOn
	}
	validated, err := s.buildExamRegistration(ctx, schoolID, row.ExamID, row.AcademicYearID, starts, ends)
	if err != nil {
		return nil, err
	}
	row.ExamID, row.AcademicYearID, row.StartsOn, row.EndsOn = validated.ExamID, validated.AcademicYearID, validated.StartsOn, validated.EndsOn
	if err := s.academic.UpdateExamRegistration(ctx, row); err != nil {
		return nil, academicWriteError("update exam registration", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "UPDATE", "exam_registrations", &row.ID, request, nil); err != nil {
		return nil, err
	}
	return s.academic.FindExamRegistrationByID(ctx, schoolID, row.ID)
}

func (s *AcademicService) DeleteExamRegistration(ctx context.Context, actor authz.Principal, schoolID, id uint64, request RequestMetadata) error {
	if !canManageExams(actor, schoolID) {
		return ErrForbidden
	}
	if _, err := s.academic.FindExamRegistrationByID(ctx, schoolID, id); err != nil {
		return err
	}
	count, err := s.academic.CountResultsByExamRegistration(ctx, schoolID, id)
	if err != nil {
		return err
	}
	if count > 0 {
		return ErrConflict
	}
	if err := s.academic.DeleteExamRegistration(ctx, schoolID, id); err != nil {
		return academicWriteError("delete exam registration", err)
	}
	return s.audit.Write(ctx, &actor.UserID, &schoolID, "DELETE", "exam_registrations", &id, request, nil)
}

func (s *AcademicService) ListExamResults(ctx context.Context, actor authz.Principal, schoolID uint64, options repository.AcademicListOptions) ([]model.ExamResult, error) {
	if !canViewResults(actor, schoolID) {
		return nil, ErrForbidden
	}
	if err := applyTeacherScope(actor, &options); err != nil {
		return nil, err
	}
	return s.academic.ListExamResults(ctx, schoolID, options)
}

func (s *AcademicService) GetExamResult(ctx context.Context, actor authz.Principal, schoolID, id uint64) (*model.ExamResult, error) {
	if !canViewResults(actor, schoolID) {
		return nil, ErrForbidden
	}
	row, err := s.academic.FindExamResultByID(ctx, schoolID, id)
	if err != nil {
		return nil, err
	}
	if err := requireTeacherAssignment(actor, row.SubjectClass.StaffID); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *AcademicService) CreateExamResult(ctx context.Context, actor authz.Principal, schoolID uint64, input CreateExamResultInput, request RequestMetadata) (*model.ExamResult, error) {
	if !canEnterMarks(actor, schoolID) {
		return nil, ErrForbidden
	}
	registration, enrollment, assignment, err := s.validateResultReferences(ctx, actor, schoolID, input.ExamRegistrationID, input.StudentClassID, input.SubjectClassID, input.Marks)
	if err != nil {
		return nil, err
	}
	row := &model.ExamResult{SchoolID: schoolID, ExamRegistrationID: registration.ID, StudentClassID: enrollment.ID, SubjectClassID: assignment.ID, Marks: input.Marks, RecordedBy: actor.UserID}
	if err := s.academic.CreateExamResult(ctx, row); err != nil {
		return nil, academicWriteError("create exam result", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "INSERT", "exam_results", &row.ID, request, map[string]any{"marks": row.Marks}); err != nil {
		return nil, err
	}
	return s.academic.FindExamResultByID(ctx, schoolID, row.ID)
}

func (s *AcademicService) UpdateExamResult(ctx context.Context, actor authz.Principal, schoolID, id uint64, input UpdateExamResultInput, request RequestMetadata) (*model.ExamResult, error) {
	if !canEnterMarks(actor, schoolID) {
		return nil, ErrForbidden
	}
	if input.Marks == nil {
		return nil, ErrNoChanges
	}
	row, err := s.academic.FindExamResultByID(ctx, schoolID, id)
	if err != nil {
		return nil, err
	}
	if err := requireTeacherAssignment(actor, row.SubjectClass.StaffID); err != nil {
		return nil, err
	}
	if *input.Marks < 0 || *input.Marks > row.SubjectClass.MaxMark {
		return nil, fmt.Errorf("%w: marks must be between 0 and %.2f", ErrInvalidInput, row.SubjectClass.MaxMark)
	}
	oldMarks := row.Marks
	row.Marks, row.RecordedBy = *input.Marks, actor.UserID
	if err := s.academic.UpdateExamResult(ctx, row); err != nil {
		return nil, academicWriteError("update exam result", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "UPDATE", "exam_results", &row.ID, request, map[string]any{"old_marks": oldMarks, "new_marks": row.Marks}); err != nil {
		return nil, err
	}
	return s.academic.FindExamResultByID(ctx, schoolID, row.ID)
}

func (s *AcademicService) DeleteExamResult(ctx context.Context, actor authz.Principal, schoolID, id uint64, request RequestMetadata) error {
	if !canEnterMarks(actor, schoolID) {
		return ErrForbidden
	}
	row, err := s.academic.FindExamResultByID(ctx, schoolID, id)
	if err != nil {
		return err
	}
	if err := requireTeacherAssignment(actor, row.SubjectClass.StaffID); err != nil {
		return err
	}
	if err := s.academic.DeleteExamResult(ctx, schoolID, id); err != nil {
		return academicWriteError("delete exam result", err)
	}
	return s.audit.Write(ctx, &actor.UserID, &schoolID, "DELETE", "exam_results", &id, request, map[string]any{"marks": row.Marks})
}

func (s *AcademicService) validateEnrollmentReferences(ctx context.Context, schoolID, studentID, classID, yearID uint64) error {
	student, err := s.academic.FindStudentReference(ctx, schoolID, studentID)
	if err != nil {
		return err
	}
	if student.Status != model.StudentStatusActive {
		return fmt.Errorf("%w: only active students can be enrolled", ErrInvalidInput)
	}
	class, err := s.academic.FindClassReference(ctx, schoolID, classID)
	if err != nil {
		return err
	}
	if class.Status != model.RecordStatusActive {
		return fmt.Errorf("%w: class is inactive", ErrInvalidInput)
	}
	_, err = s.academic.FindAcademicYearReference(ctx, schoolID, yearID)
	return err
}

func (s *AcademicService) validateAssignmentReferences(ctx context.Context, schoolID, subjectID, classID, staffID, yearID uint64, maxMark float64) error {
	if maxMark <= 0 || maxMark > 999.99 {
		return fmt.Errorf("%w: max_mark must be between 0.01 and 999.99", ErrInvalidInput)
	}
	subject, err := s.academic.FindSubjectReference(ctx, subjectID)
	if err != nil {
		return err
	}
	if subject.Status != model.RecordStatusActive {
		return fmt.Errorf("%w: subject is inactive", ErrInvalidInput)
	}
	class, err := s.academic.FindClassReference(ctx, schoolID, classID)
	if err != nil {
		return err
	}
	if class.Status != model.RecordStatusActive {
		return fmt.Errorf("%w: class is inactive", ErrInvalidInput)
	}
	if _, err := s.academic.FindStaffReference(ctx, schoolID, staffID); err != nil {
		return err
	}
	latest, err := s.academic.FindLatestStaffStatus(ctx, schoolID, staffID)
	if err != nil {
		return err
	}
	if !strings.EqualFold(latest.StatusType.Name, "Active") {
		return fmt.Errorf("%w: staff member is not active", ErrInvalidInput)
	}
	_, err = s.academic.FindAcademicYearReference(ctx, schoolID, yearID)
	return err
}

func (s *AcademicService) buildExamRegistration(ctx context.Context, schoolID, examID, yearID uint64, startsValue, endsValue string) (*model.ExamRegistration, error) {
	exam, err := s.academic.FindExamReference(ctx, examID)
	if err != nil {
		return nil, err
	}
	if exam.Status != model.RecordStatusActive {
		return nil, fmt.Errorf("%w: exam type is inactive", ErrInvalidInput)
	}
	year, err := s.academic.FindAcademicYearReference(ctx, schoolID, yearID)
	if err != nil {
		return nil, err
	}
	starts, err := requiredDate(startsValue)
	if err != nil {
		return nil, err
	}
	ends, err := requiredDate(endsValue)
	if err != nil {
		return nil, err
	}
	if ends.Before(starts) || starts.Before(year.StartsOn) || ends.After(year.EndsOn) {
		return nil, fmt.Errorf("%w: exam dates must be ordered and inside the academic year", ErrInvalidDate)
	}
	return &model.ExamRegistration{SchoolID: schoolID, ExamID: exam.ID, AcademicYearID: year.ID, StartsOn: starts, EndsOn: ends}, nil
}

func (s *AcademicService) validateResultReferences(ctx context.Context, actor authz.Principal, schoolID, registrationID, enrollmentID, assignmentID uint64, marks float64) (*model.ExamRegistration, *model.StudentClass, *model.SubjectClass, error) {
	registration, err := s.academic.FindExamRegistrationByID(ctx, schoolID, registrationID)
	if err != nil {
		return nil, nil, nil, err
	}
	enrollment, err := s.academic.FindStudentClassByID(ctx, schoolID, enrollmentID)
	if err != nil {
		return nil, nil, nil, err
	}
	assignment, err := s.academic.FindSubjectClassByID(ctx, schoolID, assignmentID)
	if err != nil {
		return nil, nil, nil, err
	}
	if registration.AcademicYearID != enrollment.AcademicYearID || registration.AcademicYearID != assignment.AcademicYearID || enrollment.ClassID != assignment.ClassID {
		return nil, nil, nil, fmt.Errorf("%w: exam, enrollment, and assignment must share a class and academic year", ErrInvalidInput)
	}
	if err := requireTeacherAssignment(actor, assignment.StaffID); err != nil {
		return nil, nil, nil, err
	}
	if marks < 0 || marks > assignment.MaxMark {
		return nil, nil, nil, fmt.Errorf("%w: marks must be between 0 and %.2f", ErrInvalidInput, assignment.MaxMark)
	}
	return registration, enrollment, assignment, nil
}

func (s *AcademicService) requireActiveSchool(ctx context.Context, schoolID uint64) error {
	school, err := s.foundation.FindSchoolByID(ctx, schoolID)
	if err != nil {
		return err
	}
	if school.Status != model.SchoolStatusActive {
		return ErrInvalidScope
	}
	return nil
}

func (s *AcademicService) requireTeacherClassAccess(ctx context.Context, actor authz.Principal, schoolID, classID, yearID uint64) error {
	if actor.Role != model.RoleTeacher {
		return nil
	}
	if actor.StaffID == nil {
		return ErrForbidden
	}
	options := repository.AcademicListOptions{ClassID: &classID, AcademicYearID: &yearID, StaffID: actor.StaffID, Limit: 1}
	rows, err := s.academic.ListSubjectClasses(ctx, schoolID, options)
	if err != nil {
		return err
	}
	if len(rows) == 0 {
		return ErrForbidden
	}
	return nil
}

func applyTeacherScope(actor authz.Principal, options *repository.AcademicListOptions) error {
	if actor.Role != model.RoleTeacher {
		return nil
	}
	if actor.StaffID == nil {
		return ErrForbidden
	}
	if options.StaffID != nil && *options.StaffID != *actor.StaffID {
		return ErrForbidden
	}
	options.StaffID = actor.StaffID
	return nil
}

func requireTeacherAssignment(actor authz.Principal, assignedStaffID uint64) error {
	if actor.Role == model.RoleTeacher && (actor.StaffID == nil || *actor.StaffID != assignedStaffID) {
		return ErrForbidden
	}
	return nil
}

func canReadAcademic(actor authz.Principal, schoolID uint64) bool {
	return canAccessSchool(actor, schoolID) && (actor.HasPermission(model.PermissionManageEnrollments) || actor.HasPermission(model.PermissionManageAssignments) || actor.HasPermission(model.PermissionManageExams) || actor.HasPermission(model.PermissionEnterMarks) || actor.HasPermission(model.PermissionViewResults))
}

func canManageEnrollments(actor authz.Principal, schoolID uint64) bool {
	return canAccessSchool(actor, schoolID) && actor.HasPermission(model.PermissionManageEnrollments)
}

func canManageAssignments(actor authz.Principal, schoolID uint64) bool {
	return canAccessSchool(actor, schoolID) && actor.HasPermission(model.PermissionManageAssignments)
}

func canManageExams(actor authz.Principal, schoolID uint64) bool {
	return canAccessSchool(actor, schoolID) && actor.HasPermission(model.PermissionManageExams)
}

func canEnterMarks(actor authz.Principal, schoolID uint64) bool {
	return canAccessSchool(actor, schoolID) && actor.HasPermission(model.PermissionEnterMarks)
}

func canViewResults(actor authz.Principal, schoolID uint64) bool {
	return canAccessSchool(actor, schoolID) && (actor.HasPermission(model.PermissionViewResults) || actor.HasPermission(model.PermissionEnterMarks))
}

func academicWriteError(operation string, err error) error {
	switch {
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return ErrDuplicateRecord
	case errors.Is(err, gorm.ErrForeignKeyViolated):
		return ErrInvalidScope
	case errors.Is(err, gorm.ErrCheckConstraintViolated):
		return ErrInvalidInput
	default:
		return fmt.Errorf("%s: %w", operation, err)
	}
}
