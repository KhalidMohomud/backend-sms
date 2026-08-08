package model

import "time"

// StudentClass is a student's single class enrollment for an academic year.
type StudentClass struct {
	ID             uint64       `gorm:"column:sc_no;primaryKey;uniqueIndex:uq_student_classes_id_school" json:"id"`
	SchoolID       uint64       `gorm:"column:sch_no;not null;index;uniqueIndex:uq_student_classes_id_school;uniqueIndex:uq_student_classes_student_year" json:"school_id"`
	StudentID      uint64       `gorm:"column:std_id;not null;index;uniqueIndex:uq_student_classes_student_year" json:"student_id"`
	ClassID        uint64       `gorm:"column:cl_no;not null;index" json:"class_id"`
	AcademicYearID uint64       `gorm:"column:y_no;not null;index;uniqueIndex:uq_student_classes_student_year" json:"academic_year_id"`
	CreatedAt      time.Time    `gorm:"not null" json:"created_at"`
	UpdatedAt      time.Time    `gorm:"not null" json:"updated_at"`
	StudentName    string       `gorm:"-" json:"student_name"`
	Student        Student      `gorm:"foreignKey:StudentID,SchoolID;references:ID,SchoolID" json:"-"`
	Class          Class        `gorm:"foreignKey:ClassID,SchoolID;references:ID,SchoolID" json:"class"`
	AcademicYear   AcademicYear `gorm:"foreignKey:AcademicYearID,SchoolID;references:ID,SchoolID" json:"academic_year"`
}

func (StudentClass) TableName() string { return "student_classes" }

// SubjectClass assigns a subject and teacher to a class for an academic year.
type SubjectClass struct {
	ID             uint64       `gorm:"column:sb_cl_no;primaryKey;uniqueIndex:uq_subject_classes_id_school" json:"id"`
	SchoolID       uint64       `gorm:"column:sch_no;not null;index;uniqueIndex:uq_subject_classes_id_school;uniqueIndex:uq_subject_classes_assignment" json:"school_id"`
	SubjectID      uint64       `gorm:"column:sub_no;not null;index;uniqueIndex:uq_subject_classes_assignment" json:"subject_id"`
	ClassID        uint64       `gorm:"column:cl_no;not null;index;uniqueIndex:uq_subject_classes_assignment" json:"class_id"`
	StaffID        uint64       `gorm:"column:stf_no;not null;index" json:"staff_id"`
	AcademicYearID uint64       `gorm:"column:y_no;not null;index;uniqueIndex:uq_subject_classes_assignment" json:"academic_year_id"`
	MaxMark        float64      `gorm:"column:max_mark;type:numeric(5,2);not null;default:100;check:ck_subject_classes_max_mark,max_mark > 0 AND max_mark <= 999.99" json:"max_mark"`
	CreatedAt      time.Time    `gorm:"not null" json:"created_at"`
	UpdatedAt      time.Time    `gorm:"not null" json:"updated_at"`
	Subject        Subject      `gorm:"foreignKey:SubjectID;references:ID" json:"subject"`
	Class          Class        `gorm:"foreignKey:ClassID,SchoolID;references:ID,SchoolID" json:"class"`
	StaffName      string       `gorm:"-" json:"staff_name"`
	Staff          Staff        `gorm:"foreignKey:StaffID,SchoolID;references:ID,SchoolID" json:"-"`
	AcademicYear   AcademicYear `gorm:"foreignKey:AcademicYearID,SchoolID;references:ID,SchoolID" json:"academic_year"`
}

func (SubjectClass) TableName() string { return "subject_classes" }

// ExamRegistration schedules a global exam type for one school and academic year.
type ExamRegistration struct {
	ID             uint64       `gorm:"column:ex_r_no;primaryKey;uniqueIndex:uq_exam_registrations_id_school" json:"id"`
	SchoolID       uint64       `gorm:"column:sch_no;not null;index;uniqueIndex:uq_exam_registrations_id_school;uniqueIndex:uq_exam_registrations_schedule" json:"school_id"`
	ExamID         uint64       `gorm:"column:ex_no;not null;index;uniqueIndex:uq_exam_registrations_schedule" json:"exam_id"`
	AcademicYearID uint64       `gorm:"column:y_no;not null;index;uniqueIndex:uq_exam_registrations_schedule" json:"academic_year_id"`
	StartsOn       time.Time    `gorm:"column:started;type:date;not null;uniqueIndex:uq_exam_registrations_schedule" json:"starts_on"`
	EndsOn         time.Time    `gorm:"column:ended;type:date;not null;check:ck_exam_registrations_dates,ended >= started" json:"ends_on"`
	CreatedAt      time.Time    `gorm:"not null" json:"created_at"`
	UpdatedAt      time.Time    `gorm:"not null" json:"updated_at"`
	Exam           Exam         `gorm:"foreignKey:ExamID;references:ID" json:"exam"`
	AcademicYear   AcademicYear `gorm:"foreignKey:AcademicYearID,SchoolID;references:ID,SchoolID" json:"academic_year"`
}

func (ExamRegistration) TableName() string { return "exam_registrations" }

// ExamResult records one subject mark for one enrollment and scheduled exam.
type ExamResult struct {
	ID                 uint64           `gorm:"column:res_no;primaryKey" json:"id"`
	SchoolID           uint64           `gorm:"column:sch_no;not null;index;uniqueIndex:uq_exam_results_entry" json:"school_id"`
	ExamRegistrationID uint64           `gorm:"column:ex_r_no;not null;index;uniqueIndex:uq_exam_results_entry" json:"exam_registration_id"`
	StudentClassID     uint64           `gorm:"column:sc_no;not null;index;uniqueIndex:uq_exam_results_entry" json:"student_class_id"`
	SubjectClassID     uint64           `gorm:"column:sb_cl_no;not null;index;uniqueIndex:uq_exam_results_entry" json:"subject_class_id"`
	Marks              float64          `gorm:"type:numeric(5,2);not null;check:ck_exam_results_marks,marks >= 0" json:"marks"`
	RecordedBy         uint64           `gorm:"column:recorded_by;not null;index" json:"recorded_by"`
	CreatedAt          time.Time        `gorm:"not null" json:"created_at"`
	UpdatedAt          time.Time        `gorm:"not null" json:"updated_at"`
	ExamRegistration   ExamRegistration `gorm:"foreignKey:ExamRegistrationID,SchoolID;references:ID,SchoolID" json:"exam_registration"`
	StudentClass       StudentClass     `gorm:"foreignKey:StudentClassID,SchoolID;references:ID,SchoolID" json:"student_class"`
	SubjectClass       SubjectClass     `gorm:"foreignKey:SubjectClassID,SchoolID;references:ID,SchoolID" json:"subject_class"`
	Recorder           User             `gorm:"foreignKey:RecordedBy;references:ID" json:"-"`
}

func (ExamResult) TableName() string { return "exam_results" }
