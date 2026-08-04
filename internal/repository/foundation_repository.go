package repository

import (
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
	ListPermissions(context.Context) ([]model.Permission, error)
	ListUsers(context.Context, *uint64) ([]model.User, error)
}

type foundationRepository struct{ db *gorm.DB }

func NewFoundationRepository(db *gorm.DB) FoundationRepository {
	return &foundationRepository{db: db}
}

func (r *foundationRepository) CreateSchool(ctx context.Context, school *model.School) error {
	return r.db.WithContext(ctx).Create(school).Error
}

func (r *foundationRepository) ListSchools(ctx context.Context) ([]model.School, error) {
	var schools []model.School
	return schools, r.db.WithContext(ctx).Order("sch_name").Find(&schools).Error
}

func (r *foundationRepository) FindSchoolByID(ctx context.Context, id uint64) (*model.School, error) {
	var school model.School
	if err := r.db.WithContext(ctx).First(&school, "sch_no = ?", id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &school, nil
}

func (r *foundationRepository) UpdateSchool(ctx context.Context, school *model.School) error {
	result := r.db.WithContext(ctx).Model(&model.School{}).
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
	return r.db.WithContext(ctx).Create(year).Error
}

func (r *foundationRepository) ListAcademicYears(ctx context.Context, schoolID uint64) ([]model.AcademicYear, error) {
	var years []model.AcademicYear
	err := r.db.WithContext(ctx).Where("sch_no = ?", schoolID).Order("started DESC").Find(&years).Error
	return years, err
}

func (r *foundationRepository) FindAcademicYearByID(ctx context.Context, schoolID, id uint64) (*model.AcademicYear, error) {
	var year model.AcademicYear
	if err := r.db.WithContext(ctx).Where("sch_no = ?", schoolID).First(&year, "y_no = ?", id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &year, nil
}

func (r *foundationRepository) UpdateAcademicYear(ctx context.Context, year *model.AcademicYear) error {
	result := r.db.WithContext(ctx).Model(&model.AcademicYear{}).
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
	result := r.db.WithContext(ctx).Where("y_no = ? AND sch_no = ?", id, schoolID).Delete(&model.AcademicYear{})
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
	if err := r.db.WithContext(ctx).Preload("Permissions").First(&role, "role_no = ?", id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &role, nil
}

func (r *foundationRepository) FindRoleByName(ctx context.Context, name string) (*model.Role, error) {
	var role model.Role
	err := r.db.WithContext(ctx).Preload("Permissions").
		Where("LOWER(role_name) = ?", strings.ToLower(strings.TrimSpace(name))).First(&role).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	return &role, nil
}

func (r *foundationRepository) ListRoles(ctx context.Context) ([]model.Role, error) {
	var roles []model.Role
	return roles, r.db.WithContext(ctx).Preload("Permissions").Order("role_name").Find(&roles).Error
}

func (r *foundationRepository) ListPermissions(ctx context.Context) ([]model.Permission, error) {
	var permissions []model.Permission
	return permissions, r.db.WithContext(ctx).Order("perm_name").Find(&permissions).Error
}

func (r *foundationRepository) ListUsers(ctx context.Context, schoolID *uint64) ([]model.User, error) {
	var users []model.User
	query := r.db.WithContext(ctx).Preload("School").Preload("Role.Permissions").Order("username")
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
