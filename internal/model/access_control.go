package model

import "time"

const (
	RoleSuperAdmin  = "SuperAdmin"
	RoleSchoolAdmin = "SchoolAdmin"
	RoleRegistrar   = "Registrar"
	RoleFinance     = "Finance"
	RoleTeacher     = "Teacher"
)

type RoleStatus string

const (
	RoleStatusActive   RoleStatus = "active"
	RoleStatusInactive RoleStatus = "inactive"
)

const (
	PermissionManageSchools       = "manage_schools"
	PermissionManageAcademicYears = "manage_academic_years"
	PermissionManageUsers         = "manage_users"
	PermissionManageRoles         = "manage_roles"
	PermissionViewAuditLogs       = "view_audit_logs"
	PermissionManageLookups       = "manage_lookups"
	PermissionManageStructure     = "manage_school_structure"
	PermissionManageStudents      = "manage_students"
	PermissionManageStaff         = "manage_staff"
)

type Role struct {
	ID          uint64       `gorm:"column:role_no;primaryKey" json:"id"`
	Name        string       `gorm:"column:role_name;size:40;not null;uniqueIndex" json:"name"`
	Description string       `gorm:"size:255" json:"description,omitempty"`
	Status      RoleStatus   `gorm:"type:varchar(20);not null;default:active;check:ck_roles_status,status IN ('active','inactive')" json:"status"`
	IsSystem    bool         `gorm:"not null;default:false" json:"is_system"`
	Permissions []Permission `gorm:"many2many:role_permissions;joinForeignKey:RoleNo;joinReferences:PermNo" json:"permissions,omitempty"`
	CreatedAt   time.Time    `gorm:"not null" json:"created_at"`
	UpdatedAt   time.Time    `gorm:"not null" json:"updated_at"`
}

func (Role) TableName() string { return "roles" }

type Permission struct {
	ID          uint64    `gorm:"column:perm_no;primaryKey" json:"id"`
	Name        string    `gorm:"column:perm_name;size:60;not null;uniqueIndex" json:"name"`
	Description string    `gorm:"size:255" json:"description,omitempty"`
	CreatedAt   time.Time `gorm:"not null" json:"created_at"`
	UpdatedAt   time.Time `gorm:"not null" json:"updated_at"`
}

func (Permission) TableName() string { return "permissions" }

type RolePermission struct {
	ID           uint64     `gorm:"column:rp_no;primaryKey" json:"id"`
	RoleID       uint64     `gorm:"column:role_no;not null;uniqueIndex:uq_role_permission" json:"role_id"`
	PermissionID uint64     `gorm:"column:perm_no;not null;uniqueIndex:uq_role_permission" json:"permission_id"`
	Role         Role       `gorm:"foreignKey:RoleID;references:ID" json:"-"`
	Permission   Permission `gorm:"foreignKey:PermissionID;references:ID" json:"-"`
	CreatedAt    time.Time  `gorm:"not null" json:"created_at"`
}

func (RolePermission) TableName() string { return "role_permissions" }
