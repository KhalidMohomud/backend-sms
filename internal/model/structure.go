package model

import "time"

type RecordStatus string

const (
	RecordStatusActive   RecordStatus = "active"
	RecordStatusInactive RecordStatus = "inactive"
)

// LookupItem is the stable API representation shared by Phase 2 lookup tables.
type LookupItem struct {
	ID        uint64       `json:"id"`
	Name      string       `json:"name"`
	Status    RecordStatus `json:"status"`
	CreatedAt time.Time    `json:"created_at"`
	UpdatedAt time.Time    `json:"updated_at"`
}

type Level struct {
	ID        uint64       `gorm:"column:lev_no;primaryKey;uniqueIndex:uq_levels_id_school" json:"id"`
	SchoolID  uint64       `gorm:"column:sch_no;not null;index;uniqueIndex:uq_levels_id_school" json:"school_id"`
	Name      string       `gorm:"column:lev_name;size:40;not null" json:"name"`
	Price     float64      `gorm:"type:numeric(10,2);not null;default:0;check:ck_levels_price,price >= 0" json:"price"`
	Status    RecordStatus `gorm:"type:varchar(20);not null;default:active;check:ck_levels_status,status IN ('active','inactive')" json:"status"`
	CreatedAt time.Time    `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time    `gorm:"not null" json:"updated_at"`
	School    School       `gorm:"foreignKey:SchoolID;references:ID" json:"-"`
}

func (Level) TableName() string { return "levels" }

type Class struct {
	ID        uint64       `gorm:"column:cl_no;primaryKey;uniqueIndex:uq_classes_id_school" json:"id"`
	SchoolID  uint64       `gorm:"column:sch_no;not null;index;uniqueIndex:uq_classes_id_school" json:"school_id"`
	LevelID   uint64       `gorm:"column:lev_no;not null;index" json:"level_id"`
	Name      string       `gorm:"column:cl_name;size:40;not null" json:"name"`
	Status    RecordStatus `gorm:"type:varchar(20);not null;default:active;check:ck_classes_status,status IN ('active','inactive')" json:"status"`
	CreatedAt time.Time    `gorm:"not null" json:"created_at"`
	UpdatedAt time.Time    `gorm:"not null" json:"updated_at"`
	School    School       `gorm:"foreignKey:SchoolID;references:ID" json:"-"`
	Level     Level        `gorm:"foreignKey:LevelID,SchoolID;references:ID,SchoolID" json:"level"`
}

func (Class) TableName() string { return "classes" }
