package model

import "time"

type SchoolStatus string

const (
	SchoolStatusActive   SchoolStatus = "active"
	SchoolStatusInactive SchoolStatus = "inactive"
)

type School struct {
	ID        uint64       `gorm:"column:sch_no;primaryKey" json:"id"`
	Name      string       `gorm:"column:sch_name;size:100;not null" json:"name"`
	Address   string       `gorm:"size:150" json:"address,omitempty"`
	Phone     string       `gorm:"column:tell;size:20" json:"phone,omitempty"`
	Email     string       `gorm:"size:254" json:"email,omitempty"`
	Logo      string       `gorm:"size:255" json:"logo,omitempty"`
	Status    SchoolStatus `gorm:"type:varchar(20);not null;default:active;check:ck_schools_status,status IN ('active','inactive')" json:"status"`
	CreatedAt time.Time    `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time    `gorm:"not null" json:"updated_at"`
}

func (School) TableName() string { return "schools" }
