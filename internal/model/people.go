package model

import "time"

type Sex string

const (
	SexMale   Sex = "M"
	SexFemale Sex = "F"
)

type StudentStatus string

const (
	StudentStatusActive      StudentStatus = "active"
	StudentStatusGraduated   StudentStatus = "graduated"
	StudentStatusTransferred StudentStatus = "transferred"
	StudentStatusDropped     StudentStatus = "dropped"
)

type Address struct {
	ID        uint64    `gorm:"column:add_no;primaryKey;uniqueIndex:uq_addresses_id_school" json:"id"`
	SchoolID  uint64    `gorm:"column:sch_no;not null;index;uniqueIndex:uq_addresses_id_school" json:"school_id"`
	District  string    `gorm:"size:60;not null" json:"district"`
	Village   string    `gorm:"size:60" json:"village,omitempty"`
	Area      string    `gorm:"size:60" json:"area,omitempty"`
	CreatedAt time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time `gorm:"not null" json:"updated_at"`
}

func (Address) TableName() string { return "addresses" }

type Responsible struct {
	ID           uint64    `gorm:"column:res_no;primaryKey;uniqueIndex:uq_responsibles_id_school" json:"id"`
	SchoolID     uint64    `gorm:"column:sch_no;not null;index;uniqueIndex:uq_responsibles_id_school" json:"school_id"`
	Name         string    `gorm:"column:res_name;size:100;not null" json:"name"`
	Phone        string    `gorm:"column:res_tell;size:20;not null" json:"phone"`
	Relationship string    `gorm:"size:40;not null" json:"relationship"`
	CreatedAt    time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt    time.Time `gorm:"not null" json:"updated_at"`
}

func (Responsible) TableName() string { return "responsibles" }

type Student struct {
	ID            uint64        `gorm:"column:std_id;primaryKey;uniqueIndex:uq_students_id_school" json:"id"`
	SchoolID      uint64        `gorm:"column:sch_no;not null;index;uniqueIndex:uq_students_id_school" json:"school_id"`
	Name          string        `gorm:"column:std_name;size:100;not null" json:"name"`
	MotherName    string        `gorm:"size:100;not null" json:"mother_name"`
	Sex           Sex           `gorm:"type:char(1);not null;check:ck_students_sex,sex IN ('M','F')" json:"sex"`
	Phone         string        `gorm:"column:tell;size:20" json:"phone,omitempty"`
	BirthDate     *time.Time    `gorm:"column:b_date;type:date" json:"birth_date,omitempty"`
	BirthPlace    string        `gorm:"column:p_birth;size:60" json:"birth_place,omitempty"`
	AddressID     *uint64       `gorm:"column:add_no;index" json:"address_id,omitempty"`
	ResponsibleID uint64        `gorm:"column:res_no;not null;index" json:"guardian_id"`
	Image         string        `gorm:"size:255" json:"image,omitempty"`
	RegisteredOn  time.Time     `gorm:"column:reg_date;type:date;not null" json:"registered_on"`
	Status        StudentStatus `gorm:"type:varchar(20);not null;default:active;check:ck_students_status,status IN ('active','graduated','transferred','dropped')" json:"status"`
	CreatedAt     time.Time     `gorm:"not null" json:"created_at"`
	UpdatedAt     time.Time     `gorm:"not null" json:"updated_at"`
	Address       *Address      `gorm:"foreignKey:AddressID,SchoolID;references:ID,SchoolID" json:"address,omitempty"`
	Responsible   Responsible   `gorm:"foreignKey:ResponsibleID,SchoolID;references:ID,SchoolID" json:"guardian"`
}

func (Student) TableName() string { return "students" }

type Staff struct {
	ID          uint64    `gorm:"column:stf_no;primaryKey;uniqueIndex:uq_staff_id_school" json:"id"`
	SchoolID    uint64    `gorm:"column:sch_no;not null;index;uniqueIndex:uq_staff_id_school" json:"school_id"`
	Name        string    `gorm:"column:stf_name;size:100;not null" json:"name"`
	Sex         Sex       `gorm:"type:char(1);not null;check:ck_staff_sex,sex IN ('M','F')" json:"sex"`
	Phone       string    `gorm:"column:tell;size:20" json:"phone,omitempty"`
	AddressID   *uint64   `gorm:"column:add_no;index" json:"address_id,omitempty"`
	JobID       uint64    `gorm:"column:job_no;not null;index" json:"job_id"`
	DecreeID    uint64    `gorm:"column:dec_no;not null;index" json:"decree_id"`
	Salary      float64   `gorm:"type:numeric(10,2);not null;check:ck_staff_salary,salary >= 0" json:"salary"`
	HiredDate   time.Time `gorm:"column:hired_date;type:date;not null" json:"hired_date"`
	Description string    `gorm:"size:255" json:"description,omitempty"`
	CreatedAt   time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null" json:"updated_at"`
	Address     *Address  `gorm:"foreignKey:AddressID,SchoolID;references:ID,SchoolID" json:"address,omitempty"`
	Job         Job       `gorm:"foreignKey:JobID;references:ID" json:"job"`
	Decree      Decree    `gorm:"foreignKey:DecreeID;references:ID" json:"decree"`
}

func (Staff) TableName() string { return "staff" }

type StaffStatus struct {
	ID                uint64          `gorm:"column:ss_no;primaryKey" json:"id"`
	SchoolID          uint64          `gorm:"column:sch_no;not null;index" json:"school_id"`
	StaffID           uint64          `gorm:"column:stf_no;not null;index" json:"staff_id"`
	StaffStatusTypeID uint64          `gorm:"column:sst_no;not null;index" json:"status_type_id"`
	Description       string          `gorm:"size:255" json:"description,omitempty"`
	StatusDate        time.Time       `gorm:"column:st_date;type:date;not null" json:"status_date"`
	CreatedAt         time.Time       `gorm:"not null" json:"created_at"`
	Staff             Staff           `gorm:"foreignKey:StaffID,SchoolID;references:ID,SchoolID" json:"-"`
	StatusType        StaffStatusType `gorm:"foreignKey:StaffStatusTypeID;references:ID" json:"status_type"`
}

func (StaffStatus) TableName() string { return "staff_status" }
