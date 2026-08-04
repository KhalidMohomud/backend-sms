package service

import (
	"backendapi/internal/model"
	"context"
	"errors"
	"testing"
)

func TestCreateInitialSuperAdminIsExplicitAndOneTime(t *testing.T) {
	users := &memoryUsers{}
	audits := &memoryAudits{}
	roles := fixedAdminRole{role: &model.Role{
		ID: 1, Name: model.RoleSuperAdmin,
		Permissions: []model.Permission{{Name: model.PermissionManageSchools}},
	}}
	admin := NewAdminService(users, roles, NewAuditWriter(audits))

	created, err := admin.CreateInitialSuperAdmin(context.Background(), "RootAdmin", "strong-password")
	if err != nil {
		t.Fatalf("CreateInitialSuperAdmin() error = %v", err)
	}
	if created.Username != "rootadmin" || !created.IsSuperAdmin() || created.PasswordHash == "strong-password" {
		t.Fatalf("created SuperAdmin is invalid: %#v", created)
	}
	if len(audits.entries) != 1 || audits.entries[0].Action != "INSERT" {
		t.Fatalf("audit entries = %#v, want one INSERT", audits.entries)
	}

	_, err = admin.CreateInitialSuperAdmin(context.Background(), "second-admin", "another-password")
	if !errors.Is(err, ErrSuperAdminExists) {
		t.Fatalf("second creation error = %v, want ErrSuperAdminExists", err)
	}
}

type fixedAdminRole struct{ role *model.Role }

func (f fixedAdminRole) FindRoleByName(context.Context, string) (*model.Role, error) {
	return f.role, nil
}
