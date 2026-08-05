package repository

import (
	"backendapi/internal/database"
	"backendapi/internal/model"
	"context"
	"strings"

	"gorm.io/gorm"
)

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

type foundationRepository struct{ db *gorm.DB }

func NewFoundationRepository(db *gorm.DB) FoundationRepository {
	return &foundationRepository{db: db}
}

func (r *foundationRepository) CreateSchool(ctx context.Context, school *model.School) error {
	return database.FromContext(ctx, r.db).Create(school).Error
}

func (r *foundationRepository) ListSchools(ctx context.Context) ([]model.School, error) {
	var schools []model.School
	return schools, database.FromContext(ctx, r.db).Order("sch_name").Find(&schools).Error
}

func (r *foundationRepository) FindSchoolByID(ctx context.Context, id uint64) (*model.School, error) {
	var school model.School
	if err := database.FromContext(ctx, r.db).First(&school, "sch_no = ?", id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &school, nil
}

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

func (r *foundationRepository) CreateAcademicYear(ctx context.Context, year *model.AcademicYear) error {
	return database.FromContext(ctx, r.db).Create(year).Error
}

func (r *foundationRepository) ListAcademicYears(ctx context.Context, schoolID uint64) ([]model.AcademicYear, error) {
	var years []model.AcademicYear
	err := database.FromContext(ctx, r.db).Where("sch_no = ?", schoolID).Order("started DESC").Find(&years).Error
	return years, err
}

func (r *foundationRepository) FindAcademicYearByID(ctx context.Context, schoolID, id uint64) (*model.AcademicYear, error) {
	var year model.AcademicYear
	if err := database.FromContext(ctx, r.db).Where("sch_no = ?", schoolID).First(&year, "y_no = ?", id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &year, nil
}

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

func (r *foundationRepository) FindRoleByID(ctx context.Context, id uint64) (*model.Role, error) {
	var role model.Role
	if err := database.FromContext(ctx, r.db).Preload("Permissions").First(&role, "role_no = ?", id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &role, nil
}

func (r *foundationRepository) FindRoleByName(ctx context.Context, name string) (*model.Role, error) {
	var role model.Role
	err := database.FromContext(ctx, r.db).Preload("Permissions").
		Where("LOWER(role_name) = ?", strings.ToLower(strings.TrimSpace(name))).First(&role).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	return &role, nil
}

func (r *foundationRepository) ListRoles(ctx context.Context) ([]model.Role, error) {
	var roles []model.Role
	return roles, database.FromContext(ctx, r.db).Preload("Permissions").Order("role_name").Find(&roles).Error
}

func (r *foundationRepository) CreateRole(ctx context.Context, role *model.Role) error {
	return database.FromContext(ctx, r.db).Create(role).Error
}

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

func (r *foundationRepository) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	var permissions []model.Permission
	return permissions, database.FromContext(ctx, r.db).Order("perm_name").Find(&permissions).Error
}

func (r *foundationRepository) ListUsers(ctx context.Context, schoolID *uint64) ([]model.User, error) {
	var users []model.User
	query := database.FromContext(ctx, r.db).Preload("School").Preload("Role.Permissions").Order("username")
	if schoolID != nil {
		query = query.Where("sch_no = ?", *schoolID)
	}
	return users, query.Find(&users).Error
}

func mapNotFound(err error) error {
	if err == gorm.ErrRecordNotFound {
		return ErrNotFound
	}
	return err
}
