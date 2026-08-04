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

type FoundationService struct {
	foundation repository.FoundationRepository
	users      repository.UserRepository
	audits     repository.AuditRepository
	audit      *AuditWriter
}

func NewFoundationService(
	foundation repository.FoundationRepository,
	users repository.UserRepository,
	audits repository.AuditRepository,
	audit *AuditWriter,
) *FoundationService {
	return &FoundationService{foundation: foundation, users: users, audits: audits, audit: audit}
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

func (s *FoundationService) CreateUser(ctx context.Context, actor authz.Principal, input CreateUserInput, request RequestMetadata) (*model.User, error) {
	if !actor.HasPermission(model.PermissionManageUsers) {
		return nil, ErrForbidden
	}
	role, err := s.foundation.FindRoleByID(ctx, input.RoleID)
	if err != nil {
		return nil, err
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
	if !actor.HasPermission(model.PermissionManageUsers) || actor.UserID == userID {
		return nil, ErrForbidden
	}
	target, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if !actor.IsSuperAdmin() {
		if actor.SchoolID == nil || target.SchoolID == nil || *actor.SchoolID != *target.SchoolID || target.Role.Name == model.RoleSchoolAdmin || target.IsSuperAdmin() {
			return nil, ErrForbidden
		}
	}
	if err := s.users.UpdateStatus(ctx, userID, input.Status); err != nil {
		return nil, fmt.Errorf("update user status: %w", err)
	}
	target.Status = input.Status
	if input.Status == model.UserStatusActive {
		target.FailedLogins = 0
	}
	if err := s.audit.Write(ctx, &actor.UserID, target.SchoolID, "UPDATE", "users", &target.ID, request, map[string]any{"status": input.Status}); err != nil {
		return nil, err
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
