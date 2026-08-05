package repository

import (
	"backendapi/internal/database"
	"backendapi/internal/model"
	"context"
	"strings"

	"gorm.io/gorm"
)

// FoundationRepository defines the interface for managing schools, academic years, roles, permissions, and users in the system.
type FoundationRepository interface {
	CreateSchool(context.Context, *model.School) error
	ListSchools(context.Context) ([]model.School, error)
	FindSchoolByID(context.Context, uint64) (*model.School, error)
	UpdateSchool(context.Context, *model.School) error
	CreateAcademicYear(context.Context, *model.AcademicYear) error
	ListAcademicYears(context.Context, uint64) ([]model.AcademicYear, error)
	FindAcademicYearByID(context.Context, uint64, uint64) (*model.AcademicYear, error)
	UpdateAcademicYear(context.Context, *model.AcademicYear) error
	DeleteAcademicYear(context.Context, uint64, uint64) error
	FindRoleByID(context.Context, uint64) (*model.Role, error)
	FindRoleByName(context.Context, string) (*model.Role, error)
	ListRoles(context.Context) ([]model.Role, error)
	CreateRole(context.Context, *model.Role) error
	UpdateRole(context.Context, *model.Role) error
	ReplaceRolePermissions(context.Context, uint64, []model.Permission) error
	FindPermissionsByIDs(context.Context, []uint64) ([]model.Permission, error)
	ListPermissions(context.Context) ([]model.Permission, error)
	ListUsers(context.Context, *uint64) ([]model.User, error)
}

// foundationRepository is a repository for managing schools, academic years, roles, permissions, and users in the system.
type foundationRepository struct{ db *gorm.DB }

func NewFoundationRepository(db *gorm.DB) FoundationRepository {
	return &foundationRepository{db: db}
}

// CreateSchool creates a new school in the system.
func (r *foundationRepository) CreateSchool(ctx context.Context, school *model.School) error {
	return database.FromContext(ctx, r.db).Create(school).Error
}

// ListSchools returns all schools in the system, ordered by name.
func (r *foundationRepository) ListSchools(ctx context.Context) ([]model.School, error) {
	var schools []model.School
	return schools, database.FromContext(ctx, r.db).Order("sch_name").Find(&schools).Error
}

// FindSchoolByID returns the school with the given ID. If no such school exists, it returns ErrNotFound.
func (r *foundationRepository) FindSchoolByID(ctx context.Context, id uint64) (*model.School, error) {
	var school model.School
	if err := database.FromContext(ctx, r.db).First(&school, "sch_no = ?", id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &school, nil
}

// UpdateSchool updates the name, address, phone, email, logo, and status of a school. If the school does not exist, it returns ErrNotFound.
func (r *foundationRepository) UpdateSchool(ctx context.Context, school *model.School) error {
	result := database.FromContext(ctx, r.db).Model(&model.School{}).
		Where("sch_no = ?", school.ID).
		Updates(map[string]any{
			"sch_name": school.Name,
			"address":  school.Address,
			"tell":     school.Phone,
			"email":    school.Email,
			"logo":     school.Logo,
			"status":   school.Status,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateAcademicYear creates a new academic year for the given school.
func (r *foundationRepository) CreateAcademicYear(ctx context.Context, year *model.AcademicYear) error {
	return database.FromContext(ctx, r.db).Create(year).Error
}

// ListAcademicYears returns all academic years for the given school ID, ordered by start date descending.
func (r *foundationRepository) ListAcademicYears(ctx context.Context, schoolID uint64) ([]model.AcademicYear, error) {
	var years []model.AcademicYear
	err := database.FromContext(ctx, r.db).Where("sch_no = ?", schoolID).Order("started DESC").Find(&years).Error
	return years, err
}

// FindAcademicYearByID returns the academic year with the given ID and school ID. If no such academic year exists, it returns ErrNotFound.
func (r *foundationRepository) FindAcademicYearByID(ctx context.Context, schoolID, id uint64) (*model.AcademicYear, error) {
	var year model.AcademicYear
	if err := database.FromContext(ctx, r.db).Where("sch_no = ?", schoolID).First(&year, "y_no = ?", id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &year, nil
}

// UpdateAcademicYear updates the name, start date, and end date of an academic year. If the academic year does not exist, it returns ErrNotFound.
func (r *foundationRepository) UpdateAcademicYear(ctx context.Context, year *model.AcademicYear) error {
	result := database.FromContext(ctx, r.db).Model(&model.AcademicYear{}).
		Where("y_no = ? AND sch_no = ?", year.ID, year.SchoolID).
		Updates(map[string]any{
			"year_name": year.YearName,
			"started":   year.StartsOn,
			"ended":     year.EndsOn,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteAcademicYear deletes the academic year with the given ID and school ID. If no such academic year exists, it returns ErrNotFound.
func (r *foundationRepository) DeleteAcademicYear(ctx context.Context, schoolID, id uint64) error {
	result := database.FromContext(ctx, r.db).Where("y_no = ? AND sch_no = ?", id, schoolID).Delete(&model.AcademicYear{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// FindRoleByID returns the role with the given ID, with its permissions preloaded. If no such role exists, it returns ErrNotFound.
func (r *foundationRepository) FindRoleByID(ctx context.Context, id uint64) (*model.Role, error) {
	var role model.Role
	if err := database.FromContext(ctx, r.db).Preload("Permissions").First(&role, "role_no = ?", id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &role, nil
}

// / FindRoleByName returns the role with the given name, case-insensitively. If no such role exists, it returns ErrNotFound.
func (r *foundationRepository) FindRoleByName(ctx context.Context, name string) (*model.Role, error) {
	var role model.Role
	err := database.FromContext(ctx, r.db).Preload("Permissions").
		Where("LOWER(role_name) = ?", strings.ToLower(strings.TrimSpace(name))).First(&role).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	return &role, nil
}

// ListRoles returns all roles in the system, ordered by name, with their permissions preloaded.
func (r *foundationRepository) ListRoles(ctx context.Context) ([]model.Role, error) {
	var roles []model.Role
	return roles, database.FromContext(ctx, r.db).Preload("Permissions").Order("role_name").Find(&roles).Error
}

// CreateRole creates a new role in the system. It does not assign any permissions to the role; use ReplaceRolePermissions for that.
func (r *foundationRepository) CreateRole(ctx context.Context, role *model.Role) error {
	return database.FromContext(ctx, r.db).Create(role).Error
}

// UpdateRole updates the name, description, and status of a role. It does not update the permissions assigned to the role; use ReplaceRolePermissions for that. If the role does not exist, it returns ErrNotFound.
func (r *foundationRepository) UpdateRole(ctx context.Context, role *model.Role) error {
	result := database.FromContext(ctx, r.db).Model(&model.Role{}).Where("role_no = ?", role.ID).
		Updates(map[string]any{"role_name": role.Name, "description": role.Description, "status": role.Status})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

// ReplaceRolePermissions replaces the permissions assigned to a role with the given list of permissions. If any of the permissions do not exist, it returns ErrNotFound.
func (r *foundationRepository) ReplaceRolePermissions(ctx context.Context, roleID uint64, permissions []model.Permission) error {
	db := database.FromContext(ctx, r.db)
	if err := db.Where("role_no = ?", roleID).Delete(&model.RolePermission{}).Error; err != nil {
		return err
	}
	assignments := make([]model.RolePermission, 0, len(permissions))
	for _, permission := range permissions {
		assignments = append(assignments, model.RolePermission{RoleID: roleID, PermissionID: permission.ID})
	}
	if len(assignments) == 0 {
		return nil
	}
	return db.Create(&assignments).Error
}

// FindPermissionsByIDs returns the permissions with the given IDs. If any of the IDs do not exist, it returns ErrNotFound.
func (r *foundationRepository) FindPermissionsByIDs(ctx context.Context, ids []uint64) ([]model.Permission, error) {
	var permissions []model.Permission
	if len(ids) == 0 {
		return permissions, nil
	}
	if err := database.FromContext(ctx, r.db).Where("perm_no IN ?", ids).Find(&permissions).Error; err != nil {
		return nil, err
	}
	if len(permissions) != len(ids) {
		return nil, ErrNotFound
	}
	return permissions, nil
}

// ListPermissions returns all permissions in the system, ordered by name.
func (r *foundationRepository) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	var permissions []model.Permission
	return permissions, database.FromContext(ctx, r.db).Order("perm_name").Find(&permissions).Error
}

// ListUsers returns all users in the system, optionally filtered by school ID, ordered by username.
func (r *foundationRepository) ListUsers(ctx context.Context, schoolID *uint64) ([]model.User, error) {
	var users []model.User
	query := database.FromContext(ctx, r.db).Preload("School").Preload("Role.Permissions").Order("username")
	if schoolID != nil {
		query = query.Where("sch_no = ?", *schoolID)
	}
	return users, query.Find(&users).Error
}

// mapNotFound maps gorm.ErrRecordNotFound to ErrNotFound, and returns other errors unchanged.
func mapNotFound(err error) error {
	if err == gorm.ErrRecordNotFound {
		return ErrNotFound
	}
	return err
}
