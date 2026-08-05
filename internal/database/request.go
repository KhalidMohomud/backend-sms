package database

import (
	"backendapi/internal/authz"
	"context"
	"strconv"

	"gorm.io/gorm"
)

type requestDBKey struct{}

type SecurityScope struct {
	Principal  authz.Principal
	AuthLookup bool
}

func BeginRequest(ctx context.Context, db *gorm.DB, scope SecurityScope) (context.Context, *gorm.DB, error) {
	tx := db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return ctx, nil, tx.Error
	}
	if err := tx.Exec("SET LOCAL ROLE kobciye_runtime").Error; err != nil {
		tx.Rollback()
		return ctx, nil, err
	}
	if err := SetSecurityScope(tx, scope); err != nil {
		tx.Rollback()
		return ctx, nil, err
	}
	return context.WithValue(ctx, requestDBKey{}, tx), tx, nil
}

func SetSecurityScope(tx *gorm.DB, scope SecurityScope) error {
	schoolID := ""
	if scope.Principal.SchoolID != nil {
		schoolID = strconv.FormatUint(*scope.Principal.SchoolID, 10)
	}
	userID := ""
	if scope.Principal.UserID != 0 {
		userID = strconv.FormatUint(scope.Principal.UserID, 10)
	}
	return tx.Exec(
		"SELECT set_config('app.current_school', ?, true), set_config('app.current_user', ?, true), set_config('app.is_superadmin', ?, true), set_config('app.auth_lookup', ?, true)",
		schoolID, userID, strconv.FormatBool(scope.Principal.IsSuperAdmin()), strconv.FormatBool(scope.AuthLookup),
	).Error
}

func FromContext(ctx context.Context, fallback *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(requestDBKey{}).(*gorm.DB); ok && tx != nil {
		return tx.WithContext(ctx)
	}
	return fallback.WithContext(ctx)
}
