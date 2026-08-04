package repository

import (
	"backendapi/internal/model"
	"context"
	"errors"
	"strings"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const maximumFailedLogins = 5

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByUsername(ctx context.Context, username string) (*model.User, error)
	FindByID(ctx context.Context, id uint64) (*model.User, error)
	FindByIDForUpdate(ctx context.Context, tx *gorm.DB, id uint64) (*model.User, error)
	RecordFailedLogin(ctx context.Context, id uint64) error
	RecordSuccessfulLogin(ctx context.Context, id uint64, at time.Time) error
	CountSuperAdmins(ctx context.Context) (int64, error)
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.withAccessControl(r.db.WithContext(ctx)).
		Where("LOWER(username) = ?", strings.ToLower(strings.TrimSpace(username))).
		First(&user).Error
	return mapUserResult(&user, err)
}

func (r *userRepository) FindByID(ctx context.Context, id uint64) (*model.User, error) {
	var user model.User
	err := r.withAccessControl(r.db.WithContext(ctx)).First(&user, "usr_no = ?", id).Error
	return mapUserResult(&user, err)
}

func (r *userRepository) FindByIDForUpdate(ctx context.Context, tx *gorm.DB, id uint64) (*model.User, error) {
	var user model.User
	err := r.withAccessControl(tx.WithContext(ctx)).
		Clauses(clause.Locking{Strength: "UPDATE"}).
		First(&user, "usr_no = ?", id).Error
	return mapUserResult(&user, err)
}

func (r *userRepository) RecordFailedLogin(ctx context.Context, id uint64) error {
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("usr_no = ? AND status = ?", id, model.UserStatusActive).
		Updates(map[string]any{
			"failed_logins": gorm.Expr("LEAST(failed_logins + 1, ?)", maximumFailedLogins),
			"status": gorm.Expr(
				"CASE WHEN failed_logins + 1 >= ? THEN ? ELSE status END",
				maximumFailedLogins,
				model.UserStatusLocked,
			),
		}).Error
}

func (r *userRepository) RecordSuccessfulLogin(ctx context.Context, id uint64, at time.Time) error {
	return r.db.WithContext(ctx).Model(&model.User{}).
		Where("usr_no = ?", id).
		Updates(map[string]any{"failed_logins": 0, "last_login": at}).Error
}

func (r *userRepository) CountSuperAdmins(ctx context.Context) (int64, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&model.User{}).
		Joins("JOIN roles ON roles.role_no = users.role_no").
		Where("users.sch_no IS NULL AND roles.role_name = ?", model.RoleSuperAdmin).
		Count(&count).Error
	return count, err
}

func (r *userRepository) withAccessControl(db *gorm.DB) *gorm.DB {
	return db.Preload("School").Preload("Role.Permissions")
}

func mapUserResult(user *model.User, err error) (*model.User, error) {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return user, nil
}
