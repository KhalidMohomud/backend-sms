package repository

import (
	"backendapi/internal/database"
	"backendapi/internal/model"
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type AcademicListOptions struct {
	Search             string
	AcademicYearID     *uint64
	ClassID            *uint64
	StudentID          *uint64
	SubjectID          *uint64
	StaffID            *uint64
	ExamID             *uint64
	ExamRegistrationID *uint64
	StudentClassID     *uint64
	SubjectClassID     *uint64
	Limit              int
	Offset             int
}

type AcademicRepository interface {
	CreateStudentClass(context.Context, *model.StudentClass) error
	ListStudentClasses(context.Context, uint64, AcademicListOptions) ([]model.StudentClass, error)
	FindStudentClassByID(context.Context, uint64, uint64) (*model.StudentClass, error)
	UpdateStudentClass(context.Context, *model.StudentClass) error
	DeleteStudentClass(context.Context, uint64, uint64) error

	CreateSubjectClass(context.Context, *model.SubjectClass) error
	ListSubjectClasses(context.Context, uint64, AcademicListOptions) ([]model.SubjectClass, error)
	FindSubjectClassByID(context.Context, uint64, uint64) (*model.SubjectClass, error)
	UpdateSubjectClass(context.Context, *model.SubjectClass) error
	DeleteSubjectClass(context.Context, uint64, uint64) error

	CreateExamRegistration(context.Context, *model.ExamRegistration) error
	ListExamRegistrations(context.Context, uint64, AcademicListOptions) ([]model.ExamRegistration, error)
	FindExamRegistrationByID(context.Context, uint64, uint64) (*model.ExamRegistration, error)
	UpdateExamRegistration(context.Context, *model.ExamRegistration) error
	DeleteExamRegistration(context.Context, uint64, uint64) error

	CreateExamResult(context.Context, *model.ExamResult) error
	ListExamResults(context.Context, uint64, AcademicListOptions) ([]model.ExamResult, error)
	FindExamResultByID(context.Context, uint64, uint64) (*model.ExamResult, error)
	UpdateExamResult(context.Context, *model.ExamResult) error
	DeleteExamResult(context.Context, uint64, uint64) error
	CountResultsByStudentClass(context.Context, uint64, uint64) (int64, error)
	CountResultsBySubjectClass(context.Context, uint64, uint64) (int64, error)
	CountResultsByExamRegistration(context.Context, uint64, uint64) (int64, error)

	FindStudentReference(context.Context, uint64, uint64) (*model.Student, error)
	FindClassReference(context.Context, uint64, uint64) (*model.Class, error)
	FindAcademicYearReference(context.Context, uint64, uint64) (*model.AcademicYear, error)
	FindStaffReference(context.Context, uint64, uint64) (*model.Staff, error)
	FindLatestStaffStatus(context.Context, uint64, uint64) (*model.StaffStatus, error)
	FindSubjectReference(context.Context, uint64) (*model.Subject, error)
	FindExamReference(context.Context, uint64) (*model.Exam, error)
}

type academicRepository struct{ db *gorm.DB }

func NewAcademicRepository(db *gorm.DB) AcademicRepository { return &academicRepository{db: db} }

func (r *academicRepository) CreateStudentClass(ctx context.Context, row *model.StudentClass) error {
	return database.FromContext(ctx, r.db).Omit(clause.Associations).Create(row).Error
}

func (r *academicRepository) ListStudentClasses(ctx context.Context, schoolID uint64, options AcademicListOptions) ([]model.StudentClass, error) {
	var rows []model.StudentClass
	query := database.FromContext(ctx, r.db).Model(&model.StudentClass{}).
		Preload("Student").Preload("Class.Level").Preload("AcademicYear").
		Where("student_classes.sch_no = ?", schoolID)
	if options.StaffID != nil {
		query = query.Joins(`JOIN subject_classes AS access_assignment
			ON access_assignment.sch_no = student_classes.sch_no
			AND access_assignment.cl_no = student_classes.cl_no
			AND access_assignment.y_no = student_classes.y_no`).
			Where("access_assignment.stf_no = ?", *options.StaffID).Distinct("student_classes.*")
	}
	if options.AcademicYearID != nil {
		query = query.Where("student_classes.y_no = ?", *options.AcademicYearID)
	}
	if options.ClassID != nil {
		query = query.Where("student_classes.cl_no = ?", *options.ClassID)
	}
	if options.StudentID != nil {
		query = query.Where("student_classes.std_id = ?", *options.StudentID)
	}
	if search := normalizedSearch(options.Search); search != "" {
		query = query.Joins("JOIN students AS search_student ON search_student.std_id = student_classes.std_id AND search_student.sch_no = student_classes.sch_no").
			Where("search_student.std_name ILIKE ?", search)
	}
	query = pageAcademic(query, options)
	err := query.Order("student_classes.y_no DESC, student_classes.sc_no DESC").Find(&rows).Error
	setStudentClassNames(rows)
	return rows, err
}

func (r *academicRepository) FindStudentClassByID(ctx context.Context, schoolID, id uint64) (*model.StudentClass, error) {
	var row model.StudentClass
	err := database.FromContext(ctx, r.db).Preload("Student").Preload("Class.Level").Preload("AcademicYear").
		Where("student_classes.sch_no = ?", schoolID).First(&row, "sc_no = ?", id).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	row.StudentName = row.Student.Name
	return &row, nil
}

func (r *academicRepository) UpdateStudentClass(ctx context.Context, row *model.StudentClass) error {
	return changed(database.FromContext(ctx, r.db).Model(&model.StudentClass{}).
		Where("sc_no = ? AND sch_no = ?", row.ID, row.SchoolID).
		Updates(map[string]any{"std_id": row.StudentID, "cl_no": row.ClassID, "y_no": row.AcademicYearID}))
}

func (r *academicRepository) DeleteStudentClass(ctx context.Context, schoolID, id uint64) error {
	return changed(database.FromContext(ctx, r.db).Where("sc_no = ? AND sch_no = ?", id, schoolID).Delete(&model.StudentClass{}))
}

func (r *academicRepository) CreateSubjectClass(ctx context.Context, row *model.SubjectClass) error {
	return database.FromContext(ctx, r.db).Omit(clause.Associations).Create(row).Error
}

func (r *academicRepository) ListSubjectClasses(ctx context.Context, schoolID uint64, options AcademicListOptions) ([]model.SubjectClass, error) {
	var rows []model.SubjectClass
	query := database.FromContext(ctx, r.db).Model(&model.SubjectClass{}).
		Preload("Subject").Preload("Class.Level").Preload("Staff").Preload("AcademicYear").
		Where("subject_classes.sch_no = ?", schoolID)
	if options.AcademicYearID != nil {
		query = query.Where("subject_classes.y_no = ?", *options.AcademicYearID)
	}
	if options.ClassID != nil {
		query = query.Where("subject_classes.cl_no = ?", *options.ClassID)
	}
	if options.SubjectID != nil {
		query = query.Where("subject_classes.sub_no = ?", *options.SubjectID)
	}
	if options.StaffID != nil {
		query = query.Where("subject_classes.stf_no = ?", *options.StaffID)
	}
	query = pageAcademic(query, options)
	err := query.Order("subject_classes.y_no DESC, subject_classes.cl_no, subject_classes.sub_no").Find(&rows).Error
	setSubjectClassNames(rows)
	return rows, err
}

func (r *academicRepository) FindSubjectClassByID(ctx context.Context, schoolID, id uint64) (*model.SubjectClass, error) {
	var row model.SubjectClass
	err := database.FromContext(ctx, r.db).Preload("Subject").Preload("Class.Level").Preload("Staff").Preload("AcademicYear").
		Where("subject_classes.sch_no = ?", schoolID).First(&row, "sb_cl_no = ?", id).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	row.StaffName = row.Staff.Name
	return &row, nil
}

func (r *academicRepository) UpdateSubjectClass(ctx context.Context, row *model.SubjectClass) error {
	return changed(database.FromContext(ctx, r.db).Model(&model.SubjectClass{}).
		Where("sb_cl_no = ? AND sch_no = ?", row.ID, row.SchoolID).
		Updates(map[string]any{"sub_no": row.SubjectID, "cl_no": row.ClassID, "stf_no": row.StaffID, "y_no": row.AcademicYearID, "max_mark": row.MaxMark}))
}

func (r *academicRepository) DeleteSubjectClass(ctx context.Context, schoolID, id uint64) error {
	return changed(database.FromContext(ctx, r.db).Where("sb_cl_no = ? AND sch_no = ?", id, schoolID).Delete(&model.SubjectClass{}))
}

func (r *academicRepository) CreateExamRegistration(ctx context.Context, row *model.ExamRegistration) error {
	return database.FromContext(ctx, r.db).Omit(clause.Associations).Create(row).Error
}

func (r *academicRepository) ListExamRegistrations(ctx context.Context, schoolID uint64, options AcademicListOptions) ([]model.ExamRegistration, error) {
	var rows []model.ExamRegistration
	query := database.FromContext(ctx, r.db).Preload("Exam").Preload("AcademicYear").Where("exam_registrations.sch_no = ?", schoolID)
	if options.AcademicYearID != nil {
		query = query.Where("exam_registrations.y_no = ?", *options.AcademicYearID)
	}
	if options.ExamID != nil {
		query = query.Where("exam_registrations.ex_no = ?", *options.ExamID)
	}
	query = pageAcademic(query, options)
	return rows, query.Order("exam_registrations.started DESC, exam_registrations.ex_r_no DESC").Find(&rows).Error
}

func (r *academicRepository) FindExamRegistrationByID(ctx context.Context, schoolID, id uint64) (*model.ExamRegistration, error) {
	var row model.ExamRegistration
	err := database.FromContext(ctx, r.db).Preload("Exam").Preload("AcademicYear").
		Where("exam_registrations.sch_no = ?", schoolID).First(&row, "ex_r_no = ?", id).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	return &row, nil
}

func (r *academicRepository) UpdateExamRegistration(ctx context.Context, row *model.ExamRegistration) error {
	return changed(database.FromContext(ctx, r.db).Model(&model.ExamRegistration{}).
		Where("ex_r_no = ? AND sch_no = ?", row.ID, row.SchoolID).
		Updates(map[string]any{"ex_no": row.ExamID, "y_no": row.AcademicYearID, "started": row.StartsOn, "ended": row.EndsOn}))
}

func (r *academicRepository) DeleteExamRegistration(ctx context.Context, schoolID, id uint64) error {
	return changed(database.FromContext(ctx, r.db).Where("ex_r_no = ? AND sch_no = ?", id, schoolID).Delete(&model.ExamRegistration{}))
}

func (r *academicRepository) CreateExamResult(ctx context.Context, row *model.ExamResult) error {
	return database.FromContext(ctx, r.db).Omit(clause.Associations).Create(row).Error
}

func (r *academicRepository) ListExamResults(ctx context.Context, schoolID uint64, options AcademicListOptions) ([]model.ExamResult, error) {
	var rows []model.ExamResult
	query := examResultPreloads(database.FromContext(ctx, r.db).Model(&model.ExamResult{})).Where("exam_results.sch_no = ?", schoolID)
	if options.StaffID != nil {
		query = query.Joins("JOIN subject_classes AS access_assignment ON access_assignment.sb_cl_no = exam_results.sb_cl_no AND access_assignment.sch_no = exam_results.sch_no").
			Where("access_assignment.stf_no = ?", *options.StaffID)
	}
	if options.ExamRegistrationID != nil {
		query = query.Where("exam_results.ex_r_no = ?", *options.ExamRegistrationID)
	}
	if options.StudentClassID != nil {
		query = query.Where("exam_results.sc_no = ?", *options.StudentClassID)
	}
	if options.SubjectClassID != nil {
		query = query.Where("exam_results.sb_cl_no = ?", *options.SubjectClassID)
	}
	if options.StudentID != nil {
		query = query.Joins("JOIN student_classes AS result_enrollment ON result_enrollment.sc_no = exam_results.sc_no AND result_enrollment.sch_no = exam_results.sch_no").
			Where("result_enrollment.std_id = ?", *options.StudentID)
	}
	query = pageAcademic(query, options)
	err := query.Order("exam_results.ex_r_no DESC, exam_results.res_no DESC").Find(&rows).Error
	setExamResultNames(rows)
	return rows, err
}

func (r *academicRepository) FindExamResultByID(ctx context.Context, schoolID, id uint64) (*model.ExamResult, error) {
	var row model.ExamResult
	err := examResultPreloads(database.FromContext(ctx, r.db)).Where("exam_results.sch_no = ?", schoolID).First(&row, "res_no = ?", id).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	row.StudentClass.StudentName = row.StudentClass.Student.Name
	row.SubjectClass.StaffName = row.SubjectClass.Staff.Name
	return &row, nil
}

func (r *academicRepository) UpdateExamResult(ctx context.Context, row *model.ExamResult) error {
	return changed(database.FromContext(ctx, r.db).Model(&model.ExamResult{}).
		Where("res_no = ? AND sch_no = ?", row.ID, row.SchoolID).
		Updates(map[string]any{"marks": row.Marks, "recorded_by": row.RecordedBy}))
}

func (r *academicRepository) DeleteExamResult(ctx context.Context, schoolID, id uint64) error {
	return changed(database.FromContext(ctx, r.db).Where("res_no = ? AND sch_no = ?", id, schoolID).Delete(&model.ExamResult{}))
}

func (r *academicRepository) CountResultsByStudentClass(ctx context.Context, schoolID, id uint64) (int64, error) {
	return r.countResults(ctx, schoolID, "sc_no", id)
}

func (r *academicRepository) CountResultsBySubjectClass(ctx context.Context, schoolID, id uint64) (int64, error) {
	return r.countResults(ctx, schoolID, "sb_cl_no", id)
}

func (r *academicRepository) CountResultsByExamRegistration(ctx context.Context, schoolID, id uint64) (int64, error) {
	return r.countResults(ctx, schoolID, "ex_r_no", id)
}

func (r *academicRepository) countResults(ctx context.Context, schoolID uint64, column string, id uint64) (int64, error) {
	var count int64
	err := database.FromContext(ctx, r.db).Model(&model.ExamResult{}).Where("sch_no = ? AND "+column+" = ?", schoolID, id).Count(&count).Error
	return count, err
}

func (r *academicRepository) FindStudentReference(ctx context.Context, schoolID, id uint64) (*model.Student, error) {
	var row model.Student
	if err := database.FromContext(ctx, r.db).Where("sch_no = ?", schoolID).First(&row, "std_id = ?", id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &row, nil
}

func (r *academicRepository) FindClassReference(ctx context.Context, schoolID, id uint64) (*model.Class, error) {
	var row model.Class
	if err := database.FromContext(ctx, r.db).Where("sch_no = ?", schoolID).First(&row, "cl_no = ?", id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &row, nil
}

func (r *academicRepository) FindAcademicYearReference(ctx context.Context, schoolID, id uint64) (*model.AcademicYear, error) {
	var row model.AcademicYear
	if err := database.FromContext(ctx, r.db).Where("sch_no = ?", schoolID).First(&row, "y_no = ?", id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &row, nil
}

func (r *academicRepository) FindStaffReference(ctx context.Context, schoolID, id uint64) (*model.Staff, error) {
	var row model.Staff
	if err := database.FromContext(ctx, r.db).Where("sch_no = ?", schoolID).First(&row, "stf_no = ?", id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &row, nil
}

func (r *academicRepository) FindLatestStaffStatus(ctx context.Context, schoolID, id uint64) (*model.StaffStatus, error) {
	var row model.StaffStatus
	err := database.FromContext(ctx, r.db).Preload("StatusType").Where("sch_no = ? AND stf_no = ?", schoolID, id).
		Order("st_date DESC, ss_no DESC").First(&row).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	return &row, nil
}

func (r *academicRepository) FindSubjectReference(ctx context.Context, id uint64) (*model.Subject, error) {
	var row model.Subject
	if err := database.FromContext(ctx, r.db).First(&row, "sub_no = ?", id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &row, nil
}

func (r *academicRepository) FindExamReference(ctx context.Context, id uint64) (*model.Exam, error) {
	var row model.Exam
	if err := database.FromContext(ctx, r.db).First(&row, "ex_no = ?", id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &row, nil
}

func pageAcademic(query *gorm.DB, options AcademicListOptions) *gorm.DB {
	limit := options.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if options.Offset < 0 {
		options.Offset = 0
	}
	return query.Limit(limit).Offset(options.Offset)
}

func examResultPreloads(query *gorm.DB) *gorm.DB {
	return query.
		Preload("ExamRegistration.Exam").Preload("ExamRegistration.AcademicYear").
		Preload("StudentClass.Student").Preload("StudentClass.Class.Level").
		Preload("SubjectClass.Subject").Preload("SubjectClass.Staff")
}

func setStudentClassNames(rows []model.StudentClass) {
	for i := range rows {
		rows[i].StudentName = rows[i].Student.Name
	}
}

func setSubjectClassNames(rows []model.SubjectClass) {
	for i := range rows {
		rows[i].StaffName = rows[i].Staff.Name
	}
}

func setExamResultNames(rows []model.ExamResult) {
	for i := range rows {
		rows[i].StudentClass.StudentName = rows[i].StudentClass.Student.Name
		rows[i].SubjectClass.StaffName = rows[i].SubjectClass.Staff.Name
	}
}

var _ AcademicRepository = (*academicRepository)(nil)
