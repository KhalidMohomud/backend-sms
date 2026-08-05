package service

import (
	"backendapi/internal/authz"
	"backendapi/internal/model"
	"backendapi/internal/repository"
	"backendapi/internal/security"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

var (
	ErrForbidden       = errors.New("operation is not permitted")
	ErrDuplicateRecord = errors.New("record already exists")
	ErrInvalidScope    = errors.New("invalid school scope")
	ErrInvalidDate     = errors.New("invalid date range")
	ErrNoChanges       = errors.New("at least one field must be provided")
	ErrConflict        = errors.New("record is in use and cannot be deleted")
	ErrProtectedRecord = errors.New("protected system record cannot be changed")
)

type CreateSchoolInput struct {
	Name    string             `json:"name" binding:"required,min=2,max=100" example:"Kobciye School"`
	Address string             `json:"address" binding:"max=150" example:"Mogadishu"`
	Phone   string             `json:"phone" binding:"max=20" example:"+252610000000"`
	Email   string             `json:"email" binding:"omitempty,email,max=254" example:"info@kobciye.edu"`
	Logo    string             `json:"logo" binding:"max=255"`
	Status  model.SchoolStatus `json:"status" binding:"omitempty,oneof=active inactive" example:"active"`
}

type CreateAcademicYearInput struct {
	YearName string `json:"year_name" binding:"required,max=20" example:"2026-2027"`
	StartsOn string `json:"starts_on" binding:"required" example:"2026-09-01"`
	EndsOn   string `json:"ends_on" binding:"required" example:"2027-06-30"`
}

type UpdateSchoolInput struct {
	Name    *string             `json:"name" binding:"omitempty,min=2,max=100" example:"Kobciye School"`
	Address *string             `json:"address" binding:"omitempty,max=150" example:"Mogadishu"`
	Phone   *string             `json:"phone" binding:"omitempty,max=20" example:"+252610000000"`
	Email   *string             `json:"email" binding:"omitempty,email,max=254" example:"info@kobciye.edu"`
	Logo    *string             `json:"logo" binding:"omitempty,max=255"`
	Status  *model.SchoolStatus `json:"status" binding:"omitempty,oneof=active inactive" example:"active"`
}

type UpdateAcademicYearInput struct {
	YearName *string `json:"year_name" binding:"omitempty,max=20" example:"2026-2027"`
	StartsOn *string `json:"starts_on" binding:"omitempty" example:"2026-09-01"`
	EndsOn   *string `json:"ends_on" binding:"omitempty" example:"2027-06-30"`
}

type CreateUserInput struct {
	SchoolID *uint64 `json:"school_id"`
	StaffID  *uint64 `json:"staff_id"`
	Username string  `json:"username" binding:"required,min=3,max=50" example:"registrar1"`
	Password string  `json:"password" binding:"required,min=12,max=72" example:"a-strong-password"`
	RoleID   uint64  `json:"role_id" binding:"required" example:"3"`
}

type UpdateUserStatusInput struct {
	Status model.UserStatus `json:"status" binding:"required,oneof=active disabled" example:"active"`
}

type UpdateUserInput struct {
	Username *string `json:"username" binding:"omitempty,min=3,max=50" example:"registrar1"`
	StaffID  *uint64 `json:"staff_id"`
	RoleID   *uint64 `json:"role_id" binding:"omitempty,min=1" example:"3"`
}

type CreateRoleInput struct {
	Name          string   `json:"name" binding:"required,min=2,max=40" example:"Counselor"`
	Description   string   `json:"description" binding:"max=255"`
	PermissionIDs []uint64 `json:"permission_ids"`
}

type UpdateRoleInput struct {
	Name        *string           `json:"name" binding:"omitempty,min=2,max=40"`
	Description *string           `json:"description" binding:"omitempty,max=255"`
	Status      *model.RoleStatus `json:"status" binding:"omitempty,oneof=active inactive"`
}

type ReplaceRolePermissionsInput struct {
	PermissionIDs []uint64 `json:"permission_ids" binding:"required"`
}

type FoundationService struct {
	foundation repository.FoundationRepository
	users      repository.UserRepository
	audits     repository.AuditRepository
	audit      *AuditWriter
	sessions   security.SessionRepository
}

func NewFoundationService(
	foundation repository.FoundationRepository,
	users repository.UserRepository,
	audits repository.AuditRepository,
	audit *AuditWriter,
	sessions security.SessionRepository,
) *FoundationService {
	return &FoundationService{foundation: foundation, users: users, audits: audits, audit: audit, sessions: sessions}
}

type PasswordResetTokenResult struct {
	ResetToken string    `json:"reset_token"`
	ExpiresAt  time.Time `json:"expires_at"`
}

func (s *FoundationService) CreateSchool(ctx context.Context, actor authz.Principal, input CreateSchoolInput, request RequestMetadata) (*model.School, error) {
	if !actor.IsSuperAdmin() || !actor.HasPermission(model.PermissionManageSchools) {
		return nil, ErrForbidden
	}
	status := input.Status
	if status == "" {
		status = model.SchoolStatusActive
	}
	school := &model.School{
		Name: strings.TrimSpace(input.Name), Address: strings.TrimSpace(input.Address),
		Phone: strings.TrimSpace(input.Phone), Email: strings.ToLower(strings.TrimSpace(input.Email)),
		Logo: strings.TrimSpace(input.Logo), Status: status,
	}
	if err := s.foundation.CreateSchool(ctx, school); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrDuplicateRecord
		}
		return nil, fmt.Errorf("create school: %w", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &school.ID, "INSERT", "schools", &school.ID, request, nil); err != nil {
		return nil, err
	}
	return school, nil
}

func (s *FoundationService) ListSchools(ctx context.Context, actor authz.Principal) ([]model.School, error) {
	if !actor.IsSuperAdmin() || !actor.HasPermission(model.PermissionManageSchools) {
		return nil, ErrForbidden
	}
	return s.foundation.ListSchools(ctx)
}

func (s *FoundationService) UpdateSchool(ctx context.Context, actor authz.Principal, schoolID uint64, input UpdateSchoolInput, request RequestMetadata) (*model.School, error) {
	if !actor.IsSuperAdmin() || !actor.HasPermission(model.PermissionManageSchools) {
		return nil, ErrForbidden
	}
	if input.Name == nil && input.Address == nil && input.Phone == nil && input.Email == nil && input.Logo == nil && input.Status == nil {
		return nil, ErrNoChanges
	}
	school, err := s.foundation.FindSchoolByID(ctx, schoolID)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		school.Name = strings.TrimSpace(*input.Name)
	}
	if input.Address != nil {
		school.Address = strings.TrimSpace(*input.Address)
	}
	if input.Phone != nil {
		school.Phone = strings.TrimSpace(*input.Phone)
	}
	if input.Email != nil {
		school.Email = strings.ToLower(strings.TrimSpace(*input.Email))
	}
	if input.Logo != nil {
		school.Logo = strings.TrimSpace(*input.Logo)
	}
	if input.Status != nil {
		school.Status = *input.Status
	}
	if err := s.foundation.UpdateSchool(ctx, school); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrDuplicateRecord
		}
		return nil, fmt.Errorf("update school: %w", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &school.ID, "UPDATE", "schools", &school.ID, request, nil); err != nil {
		return nil, err
	}
	return school, nil
}

func (s *FoundationService) ArchiveSchool(ctx context.Context, actor authz.Principal, schoolID uint64, request RequestMetadata) error {
	if !actor.IsSuperAdmin() || !actor.HasPermission(model.PermissionManageSchools) {
		return ErrForbidden
	}
	school, err := s.foundation.FindSchoolByID(ctx, schoolID)
	if err != nil {
		return err
	}
	school.Status = model.SchoolStatusInactive
	if err := s.foundation.UpdateSchool(ctx, school); err != nil {
		return fmt.Errorf("deactivate school: %w", err)
	}
	return s.audit.Write(ctx, &actor.UserID, &school.ID, "DELETE", "schools", &school.ID, request, map[string]any{"status": school.Status, "soft_delete": true})
}

func (s *FoundationService) CreateAcademicYear(ctx context.Context, actor authz.Principal, schoolID uint64, input CreateAcademicYearInput, request RequestMetadata) (*model.AcademicYear, error) {
	if !actor.HasPermission(model.PermissionManageAcademicYears) {
		return nil, ErrForbidden
	}
	if !actor.IsSuperAdmin() && (actor.SchoolID == nil || *actor.SchoolID != schoolID) {
		return nil, ErrInvalidScope
	}
	school, err := s.foundation.FindSchoolByID(ctx, schoolID)
	if err != nil {
		return nil, err
	}
	if school.Status != model.SchoolStatusActive {
		return nil, ErrInvalidScope
	}
	startsOn, err := time.Parse("2006-01-02", input.StartsOn)
	if err != nil {
		return nil, ErrInvalidDate
	}
	endsOn, err := time.Parse("2006-01-02", input.EndsOn)
	if err != nil || !endsOn.After(startsOn) {
		return nil, ErrInvalidDate
	}
	year := &model.AcademicYear{SchoolID: schoolID, YearName: strings.TrimSpace(input.YearName), StartsOn: startsOn, EndsOn: endsOn}
	if err := s.foundation.CreateAcademicYear(ctx, year); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrDuplicateRecord
		}
		return nil, fmt.Errorf("create academic year: %w", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "INSERT", "academic_years", &year.ID, request, nil); err != nil {
		return nil, err
	}
	return year, nil
}

func (s *FoundationService) ListAcademicYears(ctx context.Context, actor authz.Principal, schoolID uint64) ([]model.AcademicYear, error) {
	if !actor.IsSuperAdmin() && (actor.SchoolID == nil || *actor.SchoolID != schoolID) {
		return nil, ErrInvalidScope
	}
	return s.foundation.ListAcademicYears(ctx, schoolID)
}

func (s *FoundationService) UpdateAcademicYear(ctx context.Context, actor authz.Principal, schoolID, yearID uint64, input UpdateAcademicYearInput, request RequestMetadata) (*model.AcademicYear, error) {
	if !actor.HasPermission(model.PermissionManageAcademicYears) {
		return nil, ErrForbidden
	}
	if !actor.IsSuperAdmin() && (actor.SchoolID == nil || *actor.SchoolID != schoolID) {
		return nil, ErrInvalidScope
	}
	if input.YearName == nil && input.StartsOn == nil && input.EndsOn == nil {
		return nil, ErrNoChanges
	}
	year, err := s.foundation.FindAcademicYearByID(ctx, schoolID, yearID)
	if err != nil {
		return nil, err
	}
	if input.YearName != nil {
		name := strings.TrimSpace(*input.YearName)
		if name == "" {
			return nil, ErrNoChanges
		}
		year.YearName = name
	}
	if input.StartsOn != nil {
		year.StartsOn, err = time.Parse("2006-01-02", *input.StartsOn)
		if err != nil {
			return nil, ErrInvalidDate
		}
	}
	if input.EndsOn != nil {
		year.EndsOn, err = time.Parse("2006-01-02", *input.EndsOn)
		if err != nil {
			return nil, ErrInvalidDate
		}
	}
	if !year.EndsOn.After(year.StartsOn) {
		return nil, ErrInvalidDate
	}
	if err := s.foundation.UpdateAcademicYear(ctx, year); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrDuplicateRecord
		}
		return nil, fmt.Errorf("update academic year: %w", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "UPDATE", "academic_years", &year.ID, request, nil); err != nil {
		return nil, err
	}
	return year, nil
}

func (s *FoundationService) DeleteAcademicYear(ctx context.Context, actor authz.Principal, schoolID, yearID uint64, request RequestMetadata) error {
	if !actor.HasPermission(model.PermissionManageAcademicYears) {
		return ErrForbidden
	}
	if !actor.IsSuperAdmin() && (actor.SchoolID == nil || *actor.SchoolID != schoolID) {
		return ErrInvalidScope
	}
	if _, err := s.foundation.FindAcademicYearByID(ctx, schoolID, yearID); err != nil {
		return err
	}
	if err := s.foundation.DeleteAcademicYear(ctx, schoolID, yearID); err != nil {
		if errors.Is(err, gorm.ErrForeignKeyViolated) {
			return ErrConflict
		}
		return fmt.Errorf("delete academic year: %w", err)
	}
	return s.audit.Write(ctx, &actor.UserID, &schoolID, "DELETE", "academic_years", &yearID, request, nil)
}

func (s *FoundationService) CreateUser(ctx context.Context, actor authz.Principal, input CreateUserInput, request RequestMetadata) (*model.User, error) {
	if !actor.HasPermission(model.PermissionManageUsers) {
		return nil, ErrForbidden
	}
	role, err := s.foundation.FindRoleByID(ctx, input.RoleID)
	if err != nil {
		return nil, err
	}
	if role.Status == model.RoleStatusInactive || roleEscalates(actor, role) {
		return nil, ErrForbidden
	}
	if !actor.IsSuperAdmin() {
		if actor.SchoolID == nil || input.SchoolID == nil || *actor.SchoolID != *input.SchoolID || role.Name == model.RoleSchoolAdmin {
			return nil, ErrForbidden
		}
	}
	if role.Name == model.RoleSuperAdmin {
		if !actor.IsSuperAdmin() || input.SchoolID != nil {
			return nil, ErrForbidden
		}
	} else {
		if input.SchoolID == nil {
			return nil, ErrInvalidScope
		}
		school, err := s.foundation.FindSchoolByID(ctx, *input.SchoolID)
		if err != nil || school.Status != model.SchoolStatusActive {
			return nil, ErrInvalidScope
		}
	}
	hash, err := security.HashPassword(input.Password)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	user := &model.User{
		SchoolID: input.SchoolID, StaffID: input.StaffID, Username: strings.ToLower(strings.TrimSpace(input.Username)),
		PasswordHash: hash, RoleID: role.ID, Role: *role, Status: model.UserStatusActive,
	}
	if err := s.users.Create(ctx, user); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrDuplicateRecord
		}
		return nil, fmt.Errorf("create user: %w", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, user.SchoolID, "INSERT", "users", &user.ID, request, map[string]any{"role": role.Name}); err != nil {
		return nil, err
	}
	return user, nil
}

func (s *FoundationService) UpdateUserStatus(ctx context.Context, actor authz.Principal, userID uint64, input UpdateUserStatusInput, request RequestMetadata) (*model.User, error) {
	return s.changeUserStatus(ctx, actor, userID, input.Status, "UPDATE", request)
}

func (s *FoundationService) UpdateUser(ctx context.Context, actor authz.Principal, userID uint64, input UpdateUserInput, request RequestMetadata) (*model.User, error) {
	if input.Username == nil && input.StaffID == nil && input.RoleID == nil {
		return nil, ErrNoChanges
	}
	target, err := s.userTarget(ctx, actor, userID)
	if err != nil {
		return nil, err
	}
	if input.Username != nil {
		target.Username = strings.ToLower(strings.TrimSpace(*input.Username))
	}
	if input.StaffID != nil {
		target.StaffID = input.StaffID
	}
	if input.RoleID != nil {
		role, err := s.foundation.FindRoleByID(ctx, *input.RoleID)
		if err != nil {
			return nil, err
		}
		if role.Status == model.RoleStatusInactive || roleEscalates(actor, role) {
			return nil, ErrForbidden
		}
		if !actor.IsSuperAdmin() && (role.Name == model.RoleSchoolAdmin || role.Name == model.RoleSuperAdmin) {
			return nil, ErrForbidden
		}
		if (role.Name == model.RoleSuperAdmin) != (target.SchoolID == nil) {
			return nil, ErrInvalidScope
		}
		target.RoleID = role.ID
		target.Role = *role
	}
	if err := s.users.UpdateProfile(ctx, target); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrDuplicateRecord
		}
		return nil, fmt.Errorf("update user: %w", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, target.SchoolID, "UPDATE", "users", &target.ID, request, map[string]any{"role_id": target.RoleID}); err != nil {
		return nil, err
	}
	return target, nil
}

func (s *FoundationService) DisableUser(ctx context.Context, actor authz.Principal, userID uint64, request RequestMetadata) error {
	_, err := s.changeUserStatus(ctx, actor, userID, model.UserStatusDisabled, "DELETE", request)
	return err
}

func (s *FoundationService) CreatePasswordResetToken(ctx context.Context, actor authz.Principal, userID uint64, request RequestMetadata) (*PasswordResetTokenResult, error) {
	target, err := s.userTarget(ctx, actor, userID)
	if err != nil {
		return nil, err
	}
	token, expiresAt, err := s.sessions.CreatePasswordReset(ctx, target.ID)
	if err != nil {
		return nil, fmt.Errorf("create password reset token: %w", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, target.SchoolID, "PASSWORD_RESET_ISSUED", "users", &target.ID, request, nil); err != nil {
		return nil, err
	}
	return &PasswordResetTokenResult{ResetToken: token, ExpiresAt: expiresAt}, nil
}

func (s *FoundationService) changeUserStatus(ctx context.Context, actor authz.Principal, userID uint64, status model.UserStatus, auditAction string, request RequestMetadata) (*model.User, error) {
	target, err := s.userTarget(ctx, actor, userID)
	if err != nil {
		return nil, err
	}
	if err := s.users.UpdateStatus(ctx, userID, status); err != nil {
		return nil, fmt.Errorf("update user status: %w", err)
	}
	target.Status = status
	if status == model.UserStatusActive {
		target.FailedLogins = 0
	}
	if err := s.audit.Write(ctx, &actor.UserID, target.SchoolID, auditAction, "users", &target.ID, request, map[string]any{"status": status, "soft_delete": auditAction == "DELETE"}); err != nil {
		return nil, err
	}
	return target, nil
}

func (s *FoundationService) userTarget(ctx context.Context, actor authz.Principal, userID uint64) (*model.User, error) {
	if !actor.HasPermission(model.PermissionManageUsers) || actor.UserID == userID {
		return nil, ErrForbidden
	}
	target, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !actor.IsSuperAdmin() && (actor.SchoolID == nil || target.SchoolID == nil || *actor.SchoolID != *target.SchoolID || target.Role.Name == model.RoleSchoolAdmin || target.IsSuperAdmin()) {
		return nil, ErrForbidden
	}
	return target, nil
}

func (s *FoundationService) ListUsers(ctx context.Context, actor authz.Principal, schoolID *uint64) ([]model.User, error) {
	if !actor.HasPermission(model.PermissionManageUsers) {
		return nil, ErrForbidden
	}
	if !actor.IsSuperAdmin() {
		schoolID = actor.SchoolID
	}
	return s.foundation.ListUsers(ctx, schoolID)
}

func (s *FoundationService) ListRoles(ctx context.Context, actor authz.Principal) ([]model.Role, error) {
	if !actor.HasPermission(model.PermissionManageRoles) && !actor.HasPermission(model.PermissionManageUsers) {
		return nil, ErrForbidden
	}
	return s.foundation.ListRoles(ctx)
}

func (s *FoundationService) CreateRole(ctx context.Context, actor authz.Principal, input CreateRoleInput, request RequestMetadata) (*model.Role, error) {
	if !canManageRoles(actor) {
		return nil, ErrForbidden
	}
	name := strings.TrimSpace(input.Name)
	if strings.EqualFold(name, model.RoleSuperAdmin) {
		return nil, ErrProtectedRecord
	}
	permissions, err := s.permissionsByIDs(ctx, input.PermissionIDs)
	if err != nil {
		return nil, err
	}
	role := &model.Role{Name: name, Description: strings.TrimSpace(input.Description), Status: model.RoleStatusActive, Permissions: permissions}
	if err := s.foundation.CreateRole(ctx, role); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrDuplicateRecord
		}
		return nil, fmt.Errorf("create role: %w", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, nil, "INSERT", "roles", &role.ID, request, map[string]any{"permission_ids": input.PermissionIDs}); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *FoundationService) UpdateRole(ctx context.Context, actor authz.Principal, roleID uint64, input UpdateRoleInput, request RequestMetadata) (*model.Role, error) {
	if !canManageRoles(actor) {
		return nil, ErrForbidden
	}
	if input.Name == nil && input.Description == nil && input.Status == nil {
		return nil, ErrNoChanges
	}
	role, err := s.mutableRole(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if strings.EqualFold(name, model.RoleSuperAdmin) {
			return nil, ErrProtectedRecord
		}
		role.Name = name
	}
	if input.Description != nil {
		role.Description = strings.TrimSpace(*input.Description)
	}
	if input.Status != nil {
		role.Status = *input.Status
	}
	if err := s.foundation.UpdateRole(ctx, role); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return nil, ErrDuplicateRecord
		}
		return nil, fmt.Errorf("update role: %w", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, nil, "UPDATE", "roles", &role.ID, request, map[string]any{"status": role.Status}); err != nil {
		return nil, err
	}
	return role, nil
}

func (s *FoundationService) ArchiveRole(ctx context.Context, actor authz.Principal, roleID uint64, request RequestMetadata) error {
	status := model.RoleStatusInactive
	role, err := s.UpdateRole(ctx, actor, roleID, UpdateRoleInput{Status: &status}, request)
	if err != nil {
		return err
	}
	return s.audit.Write(ctx, &actor.UserID, nil, "DELETE", "roles", &role.ID, request, map[string]any{"soft_delete": true})
}

func (s *FoundationService) ReplaceRolePermissions(ctx context.Context, actor authz.Principal, roleID uint64, input ReplaceRolePermissionsInput, request RequestMetadata) (*model.Role, error) {
	if !canManageRoles(actor) {
		return nil, ErrForbidden
	}
	role, err := s.mutableRole(ctx, roleID)
	if err != nil {
		return nil, err
	}
	permissions, err := s.permissionsByIDs(ctx, input.PermissionIDs)
	if err != nil {
		return nil, err
	}
	if err := s.foundation.ReplaceRolePermissions(ctx, role.ID, permissions); err != nil {
		return nil, fmt.Errorf("replace role permissions: %w", err)
	}
	role.Permissions = permissions
	if err := s.audit.Write(ctx, &actor.UserID, nil, "UPDATE", "role_permissions", &role.ID, request, map[string]any{"permission_ids": input.PermissionIDs}); err != nil {
		return nil, err
	}
	return role, nil
}

func canManageRoles(actor authz.Principal) bool {
	return actor.IsSuperAdmin() && actor.HasPermission(model.PermissionManageRoles)
}

func (s *FoundationService) mutableRole(ctx context.Context, roleID uint64) (*model.Role, error) {
	role, err := s.foundation.FindRoleByID(ctx, roleID)
	if err != nil {
		return nil, err
	}
	if role.Name == model.RoleSuperAdmin {
		return nil, ErrProtectedRecord
	}
	return role, nil
}

func (s *FoundationService) permissionsByIDs(ctx context.Context, input []uint64) ([]model.Permission, error) {
	seen := make(map[uint64]struct{}, len(input))
	ids := make([]uint64, 0, len(input))
	for _, id := range input {
		if id == 0 {
			return nil, repository.ErrNotFound
		}
		if _, exists := seen[id]; !exists {
			seen[id] = struct{}{}
			ids = append(ids, id)
		}
	}
	return s.foundation.FindPermissionsByIDs(ctx, ids)
}

func roleEscalates(actor authz.Principal, role *model.Role) bool {
	if actor.IsSuperAdmin() {
		return false
	}
	for _, permission := range role.Permissions {
		if !actor.HasPermission(permission.Name) {
			return true
		}
	}
	return false
}

func (s *FoundationService) ListPermissions(ctx context.Context, actor authz.Principal) ([]model.Permission, error) {
	if !actor.HasPermission(model.PermissionManageRoles) {
		return nil, ErrForbidden
	}
	return s.foundation.ListPermissions(ctx)
}

func (s *FoundationService) ListAuditLogs(ctx context.Context, actor authz.Principal, schoolID *uint64, limit, offset int) ([]model.AuditLog, error) {
	if !actor.HasPermission(model.PermissionViewAuditLogs) {
		return nil, ErrForbidden
	}
	if !actor.IsSuperAdmin() {
		schoolID = actor.SchoolID
	}
	return s.audits.List(ctx, schoolID, limit, offset)
}
