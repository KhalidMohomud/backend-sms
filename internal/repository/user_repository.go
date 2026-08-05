package repository

import (
	"backendapi/internal/database"
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
	Create(context.Context, *model.User) error
	FindByUsername(context.Context, string) (*model.User, error)
	FindByID(context.Context, uint64) (*model.User, error)
	RecordFailedLogin(context.Context, uint64) error
	RecordSuccessfulLogin(context.Context, uint64, time.Time) error
	UpdateProfile(context.Context, *model.User) error
	UpdatePassword(context.Context, uint64, string) error
	UpdateStatus(context.Context, uint64, model.UserStatus) error
	CountSuperAdmins(context.Context) (int64, error)
}

type userRepository struct{ db *gorm.DB }

func NewUserRepository(db *gorm.DB) UserRepository { return &userRepository{db: db} }

func (r *userRepository) Create(ctx context.Context, user *model.User) error {
	// Roles and schools are existing access-control records. Never let GORM try
	// to insert or update them while creating a user.
	return database.FromContext(ctx, r.db).Omit(clause.Associations).Create(user).Error
}

func (r *userRepository) FindByUsername(ctx context.Context, username string) (*model.User, error) {
	var user model.User
	err := r.withAccessControl(database.FromContext(ctx, r.db)).
		Where("LOWER(username) = ?", strings.ToLower(strings.TrimSpace(username))).First(&user).Error
	return mapUserResult(&user, err)
}

func (r *userRepository) FindByID(ctx context.Context, id uint64) (*model.User, error) {
	var user model.User
	err := r.withAccessControl(database.FromContext(ctx, r.db)).First(&user, "usr_no = ?", id).Error
	return mapUserResult(&user, err)
}

func (r *userRepository) RecordFailedLogin(ctx context.Context, id uint64) error {
	return database.FromContext(ctx, r.db).Model(&model.User{}).
		Where("usr_no = ? AND status = ?", id, model.UserStatusActive).
		Updates(map[string]any{
			"failed_logins": gorm.Expr("LEAST(failed_logins + 1, ?)", maximumFailedLogins),
			"status":        gorm.Expr("CASE WHEN failed_logins + 1 >= ? THEN ? ELSE status END", maximumFailedLogins, model.UserStatusLocked),
		}).Error
}

func (r *userRepository) RecordSuccessfulLogin(ctx context.Context, id uint64, at time.Time) error {
	return database.FromContext(ctx, r.db).Model(&model.User{}).Where("usr_no = ?", id).
		Updates(map[string]any{"failed_logins": 0, "last_login": at}).Error
}

func (r *userRepository) UpdateProfile(ctx context.Context, user *model.User) error {
	result := database.FromContext(ctx, r.db).Model(&model.User{}).Where("usr_no = ?", user.ID).
		Updates(map[string]any{
			"username": user.Username,
			"stf_no":   user.StaffID,
			"role_no":  user.RoleID,
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *userRepository) UpdatePassword(ctx context.Context, id uint64, passwordHash string) error {
	result := database.FromContext(ctx, r.db).Model(&model.User{}).Where("usr_no = ?", id).
		Updates(map[string]any{
			"password_hash": passwordHash,
			"failed_logins": 0,
			"status":        gorm.Expr("CASE WHEN status = ? THEN ? ELSE status END", model.UserStatusLocked, model.UserStatusActive),
		})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *userRepository) UpdateStatus(ctx context.Context, id uint64, status model.UserStatus) error {
	updates := map[string]any{"status": status}
	if status == model.UserStatusActive {
		updates["failed_logins"] = 0
	}
	result := database.FromContext(ctx, r.db).Model(&model.User{}).Where("usr_no = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *userRepository) CountSuperAdmins(ctx context.Context) (int64, error) {
	var count int64
	err := database.FromContext(ctx, r.db).Model(&model.User{}).
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
