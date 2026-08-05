package service

import (
	"backendapi/internal/authz"
	"backendapi/internal/model"
	"backendapi/internal/repository"
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"gorm.io/gorm"
)

type CreateLookupInput struct {
	Name string `json:"name" binding:"required,min=2,max=60" example:"Mathematics"`
}

type UpdateLookupInput struct {
	Name   *string             `json:"name" binding:"omitempty,min=2,max=60" example:"Mathematics"`
	Status *model.RecordStatus `json:"status" binding:"omitempty,oneof=active inactive" example:"active"`
}

type CreateLevelInput struct {
	Name   string             `json:"name" binding:"required,min=2,max=40" example:"Form 1"`
	Price  float64            `json:"price" binding:"gte=0" example:"25"`
	Status model.RecordStatus `json:"status" binding:"omitempty,oneof=active inactive" example:"active"`
}

type UpdateLevelInput struct {
	Name   *string             `json:"name" binding:"omitempty,min=2,max=40" example:"Form 1"`
	Price  *float64            `json:"price" binding:"omitempty,gte=0" example:"25"`
	Status *model.RecordStatus `json:"status" binding:"omitempty,oneof=active inactive" example:"active"`
}

type CreateClassInput struct {
	Name    string             `json:"name" binding:"required,min=2,max=40" example:"Form 1-A"`
	LevelID uint64             `json:"level_id" binding:"required,min=1" example:"1"`
	Status  model.RecordStatus `json:"status" binding:"omitempty,oneof=active inactive" example:"active"`
}

type UpdateClassInput struct {
	Name    *string             `json:"name" binding:"omitempty,min=2,max=40" example:"Form 1-A"`
	LevelID *uint64             `json:"level_id" binding:"omitempty,min=1" example:"1"`
	Status  *model.RecordStatus `json:"status" binding:"omitempty,oneof=active inactive" example:"active"`
}

type StructureService struct {
	structure  repository.StructureRepository
	foundation repository.FoundationRepository
	audit      *AuditWriter
}

func NewStructureService(
	structure repository.StructureRepository,
	foundation repository.FoundationRepository,
	audit *AuditWriter,
) *StructureService {
	return &StructureService{structure: structure, foundation: foundation, audit: audit}
}

func (s *StructureService) ListLookups(ctx context.Context, kind repository.LookupKind) ([]model.LookupItem, error) {
	return s.structure.ListLookups(ctx, kind)
}

func (s *StructureService) GetLookup(ctx context.Context, kind repository.LookupKind, id uint64) (*model.LookupItem, error) {
	return s.structure.FindLookupByID(ctx, kind, id)
}

func (s *StructureService) CreateLookup(
	ctx context.Context,
	actor authz.Principal,
	kind repository.LookupKind,
	input CreateLookupInput,
	request RequestMetadata,
) (*model.LookupItem, error) {
	if !actor.IsSuperAdmin() || !actor.HasPermission(model.PermissionManageLookups) {
		return nil, ErrForbidden
	}
	name := strings.TrimSpace(input.Name)
	if err := validateName(name, lookupNameLimit(kind)); err != nil {
		return nil, err
	}
	item := &model.LookupItem{Name: name, Status: model.RecordStatusActive}
	if err := s.structure.CreateLookup(ctx, kind, item); err != nil {
		return nil, structureWriteError("create lookup", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, nil, "INSERT", string(kind), &item.ID, request, nil); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *StructureService) UpdateLookup(
	ctx context.Context,
	actor authz.Principal,
	kind repository.LookupKind,
	id uint64,
	input UpdateLookupInput,
	request RequestMetadata,
) (*model.LookupItem, error) {
	if !actor.IsSuperAdmin() || !actor.HasPermission(model.PermissionManageLookups) {
		return nil, ErrForbidden
	}
	if input.Name == nil && input.Status == nil {
		return nil, ErrNoChanges
	}
	item, err := s.structure.FindLookupByID(ctx, kind, id)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if err := validateName(name, lookupNameLimit(kind)); err != nil {
			return nil, err
		}
		item.Name = name
	}
	if input.Status != nil {
		item.Status = *input.Status
	}
	if err := s.structure.UpdateLookup(ctx, kind, item); err != nil {
		return nil, structureWriteError("update lookup", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, nil, "UPDATE", string(kind), &item.ID, request, nil); err != nil {
		return nil, err
	}
	return item, nil
}

func (s *StructureService) ArchiveLookup(
	ctx context.Context,
	actor authz.Principal,
	kind repository.LookupKind,
	id uint64,
	request RequestMetadata,
) error {
	if !actor.IsSuperAdmin() || !actor.HasPermission(model.PermissionManageLookups) {
		return ErrForbidden
	}
	item, err := s.structure.FindLookupByID(ctx, kind, id)
	if err != nil {
		return err
	}
	item.Status = model.RecordStatusInactive
	if err := s.structure.UpdateLookup(ctx, kind, item); err != nil {
		return structureWriteError("deactivate lookup", err)
	}
	return s.audit.Write(ctx, &actor.UserID, nil, "DELETE", string(kind), &item.ID, request, map[string]any{"soft_delete": true})
}

func (s *StructureService) ListLevels(ctx context.Context, actor authz.Principal, schoolID uint64) ([]model.Level, error) {
	if !canAccessSchool(actor, schoolID) {
		return nil, ErrInvalidScope
	}
	return s.structure.ListLevels(ctx, schoolID)
}

func (s *StructureService) GetLevel(ctx context.Context, actor authz.Principal, schoolID, id uint64) (*model.Level, error) {
	if !canAccessSchool(actor, schoolID) {
		return nil, ErrInvalidScope
	}
	return s.structure.FindLevelByID(ctx, schoolID, id)
}

func (s *StructureService) CreateLevel(
	ctx context.Context,
	actor authz.Principal,
	schoolID uint64,
	input CreateLevelInput,
	request RequestMetadata,
) (*model.Level, error) {
	if !canManageStructure(actor, schoolID) {
		return nil, ErrForbidden
	}
	if err := s.requireActiveSchool(ctx, schoolID); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(input.Name)
	if err := validateName(name, 40); err != nil {
		return nil, err
	}
	if input.Price < 0 {
		return nil, fmt.Errorf("%w: price cannot be negative", ErrInvalidInput)
	}
	status := input.Status
	if status == "" {
		status = model.RecordStatusActive
	}
	level := &model.Level{SchoolID: schoolID, Name: name, Price: input.Price, Status: status}
	if err := s.structure.CreateLevel(ctx, level); err != nil {
		return nil, structureWriteError("create level", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "INSERT", "levels", &level.ID, request, nil); err != nil {
		return nil, err
	}
	return level, nil
}

func (s *StructureService) UpdateLevel(
	ctx context.Context,
	actor authz.Principal,
	schoolID, id uint64,
	input UpdateLevelInput,
	request RequestMetadata,
) (*model.Level, error) {
	if !canManageStructure(actor, schoolID) {
		return nil, ErrForbidden
	}
	if input.Name == nil && input.Price == nil && input.Status == nil {
		return nil, ErrNoChanges
	}
	level, err := s.structure.FindLevelByID(ctx, schoolID, id)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if err := validateName(name, 40); err != nil {
			return nil, err
		}
		level.Name = name
	}
	if input.Price != nil {
		if *input.Price < 0 {
			return nil, fmt.Errorf("%w: price cannot be negative", ErrInvalidInput)
		}
		level.Price = *input.Price
	}
	if input.Status != nil {
		if *input.Status == model.RecordStatusInactive {
			count, countErr := s.structure.CountActiveClassesByLevel(ctx, schoolID, id)
			if countErr != nil {
				return nil, countErr
			}
			if count != 0 {
				return nil, ErrConflict
			}
		}
		level.Status = *input.Status
	}
	if err := s.structure.UpdateLevel(ctx, level); err != nil {
		return nil, structureWriteError("update level", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "UPDATE", "levels", &level.ID, request, nil); err != nil {
		return nil, err
	}
	return level, nil
}

func (s *StructureService) ArchiveLevel(ctx context.Context, actor authz.Principal, schoolID, id uint64, request RequestMetadata) error {
	if !canManageStructure(actor, schoolID) {
		return ErrForbidden
	}
	level, err := s.structure.FindLevelByID(ctx, schoolID, id)
	if err != nil {
		return err
	}
	count, err := s.structure.CountActiveClassesByLevel(ctx, schoolID, id)
	if err != nil {
		return err
	}
	if count != 0 {
		return ErrConflict
	}
	level.Status = model.RecordStatusInactive
	if err := s.structure.UpdateLevel(ctx, level); err != nil {
		return structureWriteError("deactivate level", err)
	}
	return s.audit.Write(ctx, &actor.UserID, &schoolID, "DELETE", "levels", &level.ID, request, map[string]any{"soft_delete": true})
}

func (s *StructureService) ListClasses(ctx context.Context, actor authz.Principal, schoolID uint64) ([]model.Class, error) {
	if !canAccessSchool(actor, schoolID) {
		return nil, ErrInvalidScope
	}
	return s.structure.ListClasses(ctx, schoolID)
}

func (s *StructureService) GetClass(ctx context.Context, actor authz.Principal, schoolID, id uint64) (*model.Class, error) {
	if !canAccessSchool(actor, schoolID) {
		return nil, ErrInvalidScope
	}
	return s.structure.FindClassByID(ctx, schoolID, id)
}

func (s *StructureService) CreateClass(
	ctx context.Context,
	actor authz.Principal,
	schoolID uint64,
	input CreateClassInput,
	request RequestMetadata,
) (*model.Class, error) {
	if !canManageStructure(actor, schoolID) {
		return nil, ErrForbidden
	}
	if err := s.requireActiveSchool(ctx, schoolID); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(input.Name)
	if err := validateName(name, 40); err != nil {
		return nil, err
	}
	level, err := s.structure.FindLevelByID(ctx, schoolID, input.LevelID)
	if err != nil {
		return nil, err
	}
	if level.Status != model.RecordStatusActive {
		return nil, ErrInvalidScope
	}
	status := input.Status
	if status == "" {
		status = model.RecordStatusActive
	}
	class := &model.Class{SchoolID: schoolID, LevelID: level.ID, Name: name, Status: status}
	if err := s.structure.CreateClass(ctx, class); err != nil {
		return nil, structureWriteError("create class", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "INSERT", "classes", &class.ID, request, nil); err != nil {
		return nil, err
	}
	class.Level = *level
	return class, nil
}

func (s *StructureService) UpdateClass(
	ctx context.Context,
	actor authz.Principal,
	schoolID, id uint64,
	input UpdateClassInput,
	request RequestMetadata,
) (*model.Class, error) {
	if !canManageStructure(actor, schoolID) {
		return nil, ErrForbidden
	}
	if input.Name == nil && input.LevelID == nil && input.Status == nil {
		return nil, ErrNoChanges
	}
	class, err := s.structure.FindClassByID(ctx, schoolID, id)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if err := validateName(name, 40); err != nil {
			return nil, err
		}
		class.Name = name
	}
	if input.LevelID != nil {
		class.LevelID = *input.LevelID
	}
	level, err := s.structure.FindLevelByID(ctx, schoolID, class.LevelID)
	if err != nil {
		return nil, err
	}
	if input.Status != nil {
		class.Status = *input.Status
	}
	if class.Status == model.RecordStatusActive && level.Status != model.RecordStatusActive {
		return nil, ErrInvalidScope
	}
	if err := s.structure.UpdateClass(ctx, class); err != nil {
		return nil, structureWriteError("update class", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "UPDATE", "classes", &class.ID, request, nil); err != nil {
		return nil, err
	}
	class.Level = *level
	return class, nil
}

func (s *StructureService) ArchiveClass(ctx context.Context, actor authz.Principal, schoolID, id uint64, request RequestMetadata) error {
	if !canManageStructure(actor, schoolID) {
		return ErrForbidden
	}
	class, err := s.structure.FindClassByID(ctx, schoolID, id)
	if err != nil {
		return err
	}
	class.Status = model.RecordStatusInactive
	if err := s.structure.UpdateClass(ctx, class); err != nil {
		return structureWriteError("deactivate class", err)
	}
	return s.audit.Write(ctx, &actor.UserID, &schoolID, "DELETE", "classes", &class.ID, request, map[string]any{"soft_delete": true})
}

func (s *StructureService) requireActiveSchool(ctx context.Context, schoolID uint64) error {
	school, err := s.foundation.FindSchoolByID(ctx, schoolID)
	if err != nil {
		return err
	}
	if school.Status != model.SchoolStatusActive {
		return ErrInvalidScope
	}
	return nil
}

func canAccessSchool(actor authz.Principal, schoolID uint64) bool {
	return actor.IsSuperAdmin() || (actor.SchoolID != nil && *actor.SchoolID == schoolID)
}

func canManageStructure(actor authz.Principal, schoolID uint64) bool {
	return canAccessSchool(actor, schoolID) && actor.HasPermission(model.PermissionManageStructure)
}

func lookupNameLimit(kind repository.LookupKind) int {
	switch kind {
	case repository.LookupPeriods, repository.LookupAttendanceStatus, repository.LookupAttendanceConditions:
		return 30
	case repository.LookupStaffStatusTypes:
		return 40
	default:
		return 60
	}
}

func validateName(name string, maximum int) error {
	length := utf8.RuneCountInString(name)
	if length < 2 || length > maximum {
		return fmt.Errorf("%w: name must contain 2 to %d characters", ErrInvalidInput, maximum)
	}
	return nil
}

func structureWriteError(operation string, err error) error {
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrDuplicateRecord
	}
	return fmt.Errorf("%s: %w", operation, err)
}
