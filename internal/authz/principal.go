// Package authz contains the authenticated identity used by authorization checks.
package authz

import "backendapi/internal/model"

type Principal struct {
	UserID      uint64   `json:"user_id"`
	SchoolID    *uint64  `json:"school_id,omitempty"`
	StaffID     *uint64  `json:"staff_id,omitempty"`
	Role        string   `json:"role"`
	Permissions []string `json:"permissions"`
}

func FromUser(user *model.User) Principal {
	return Principal{
		UserID:      user.ID,
		SchoolID:    user.SchoolID,
		StaffID:     user.StaffID,
		Role:        user.Role.Name,
		Permissions: user.PermissionNames(),
	}
}

func (p Principal) IsSuperAdmin() bool {
	return p.SchoolID == nil && p.Role == model.RoleSuperAdmin
}

func (p Principal) HasPermission(required string) bool {
	if p.IsSuperAdmin() {
		return true
	}
	for _, permission := range p.Permissions {
		if permission == required {
			return true
		}
	}
	return false
}
