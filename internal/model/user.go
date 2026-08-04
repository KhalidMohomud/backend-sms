package model

import "time"

type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusLocked   UserStatus = "locked"
	UserStatusDisabled UserStatus = "disabled"
)

type User struct {
	ID           uint64     `gorm:"column:usr_no;primaryKey" json:"id"`
	SchoolID     *uint64    `gorm:"column:sch_no;index" json:"school_id,omitempty"`
	StaffID      *uint64    `gorm:"column:stf_no" json:"staff_id,omitempty"`
	Username     string     `gorm:"size:50;not null" json:"username"`
	PasswordHash string     `gorm:"size:255;not null" json:"-"`
	RoleID       uint64     `gorm:"column:role_no;not null;index" json:"role_id"`
	Status       UserStatus `gorm:"type:varchar(20);not null;default:active" json:"status"`
	FailedLogins uint8      `gorm:"not null;default:0" json:"-"`
	LastLogin    *time.Time `json:"last_login,omitempty"`
	CreatedAt    time.Time  `gorm:"not null" json:"created_at"`
	UpdatedAt    time.Time  `gorm:"not null" json:"updated_at"`
	School       *School    `gorm:"foreignKey:SchoolID;references:ID" json:"school,omitempty"`
	Role         Role       `gorm:"foreignKey:RoleID;references:ID" json:"role"`
}

func (User) TableName() string { return "users" }

func (u User) IsSuperAdmin() bool {
	return u.SchoolID == nil && u.Role.Name == RoleSuperAdmin
}

func (u User) PermissionNames() []string {
	permissions := make([]string, 0, len(u.Role.Permissions))
	for _, permission := range u.Role.Permissions {
		permissions = append(permissions, permission.Name)
	}
	return permissions
}
