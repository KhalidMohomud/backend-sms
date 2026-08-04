package model

import (
	"encoding/json"
	"time"
)

type AuditLog struct {
	ID           uint64          `gorm:"column:log_no;primaryKey" json:"id"`
	UserID       *uint64         `gorm:"column:usr_no;index" json:"user_id,omitempty"`
	SchoolID     *uint64         `gorm:"column:sch_no;index" json:"school_id,omitempty"`
	Action       string          `gorm:"size:30;not null;index" json:"action"`
	ResourceType string          `gorm:"column:table_name;size:60;not null" json:"resource_type"`
	RecordID     *uint64         `gorm:"column:record_id" json:"record_id,omitempty"`
	IPAddress    string          `gorm:"column:ip_address;size:45" json:"ip_address,omitempty"`
	Metadata     json.RawMessage `gorm:"type:jsonb;not null;default:'{}'" json:"metadata,omitempty" swaggertype:"object"`
	CreatedAt    time.Time       `gorm:"column:log_time;not null;index" json:"created_at"`
	User         *User           `gorm:"foreignKey:UserID;references:ID" json:"-"`
	School       *School         `gorm:"foreignKey:SchoolID;references:ID" json:"-"`
}

func (AuditLog) TableName() string { return "audit_logs" }
