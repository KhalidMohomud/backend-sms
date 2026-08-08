package model

import "time"

type Job struct {
	ID        uint64       `gorm:"column:job_no;primaryKey" json:"id"`
	Name      string       `gorm:"column:job_name;size:60;not null" json:"name"`
	Status    RecordStatus `gorm:"type:varchar(20);not null;default:active;check:ck_jobs_status,status IN ('active','inactive')" json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

func (Job) TableName() string { return "jobs" }

type Decree struct {
	ID        uint64       `gorm:"column:dec_no;primaryKey" json:"id"`
	Name      string       `gorm:"column:dec_name;size:60;not null" json:"name"`
	Status    RecordStatus `gorm:"type:varchar(20);not null;default:active;check:ck_decrees_status,status IN ('active','inactive')" json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

func (Decree) TableName() string { return "decrees" }

type Subject struct {
	ID        uint64       `gorm:"column:sub_no;primaryKey" json:"id"`
	Name      string       `gorm:"column:sub_name;size:60;not null" json:"name"`
	Status    RecordStatus `gorm:"type:varchar(20);not null;default:active;check:ck_subjects_status,status IN ('active','inactive')" json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

func (Subject) TableName() string { return "subjects" }

type Exam struct {
	ID        uint64       `gorm:"column:ex_no;primaryKey" json:"id"`
	Name      string       `gorm:"column:ex_name;size:60;not null" json:"name"`
	Status    RecordStatus `gorm:"type:varchar(20);not null;default:active;check:ck_exams_status,status IN ('active','inactive')" json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

func (Exam) TableName() string { return "exams" }

type Period struct {
	ID        uint64       `gorm:"column:per_no;primaryKey"`
	Name      string       `gorm:"column:per_name;size:30;not null"`
	Status    RecordStatus `gorm:"type:varchar(20);not null;default:active;check:ck_periods_status,status IN ('active','inactive')"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (Period) TableName() string { return "periods" }

type AttendanceStatus struct {
	ID        uint64       `gorm:"column:ast_no;primaryKey"`
	Name      string       `gorm:"column:ast_name;size:30;not null"`
	Status    RecordStatus `gorm:"type:varchar(20);not null;default:active;check:ck_attendance_status_status,status IN ('active','inactive')"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (AttendanceStatus) TableName() string { return "attendance_status" }

type AttendanceCondition struct {
	ID        uint64       `gorm:"column:con_no;primaryKey"`
	Name      string       `gorm:"column:con_name;size:30;not null"`
	Status    RecordStatus `gorm:"type:varchar(20);not null;default:active;check:ck_att_conditions_status,status IN ('active','inactive')"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (AttendanceCondition) TableName() string { return "att_conditions" }

type StaffStatusType struct {
	ID        uint64       `gorm:"column:sst_no;primaryKey" json:"id"`
	Name      string       `gorm:"column:sst_name;size:40;not null" json:"name"`
	Status    RecordStatus `gorm:"type:varchar(20);not null;default:active;check:ck_staff_status_types_status,status IN ('active','inactive')" json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

func (StaffStatusType) TableName() string { return "staff_status_types" }

type AmountType struct {
	ID        uint64       `gorm:"column:am_no;primaryKey"`
	Name      string       `gorm:"column:am_name;size:60;not null"`
	Status    RecordStatus `gorm:"type:varchar(20);not null;default:active;check:ck_amount_types_status,status IN ('active','inactive')"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (AmountType) TableName() string { return "amount_types" }

type ExpenseType struct {
	ID        uint64       `gorm:"column:exp_no;primaryKey"`
	Name      string       `gorm:"column:exp_name;size:60;not null"`
	Status    RecordStatus `gorm:"type:varchar(20);not null;default:active;check:ck_expense_types_status,status IN ('active','inactive')"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (ExpenseType) TableName() string { return "expense_types" }
