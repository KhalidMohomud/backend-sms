package repository

import (
	"backendapi/internal/database"
	"backendapi/internal/model"
	"context"

	"gorm.io/gorm"
)

type AuditRepository interface {
	Create(ctx context.Context, entry *model.AuditLog) error
	List(ctx context.Context, schoolID *uint64, limit, offset int) ([]model.AuditLog, error)
}

type auditRepository struct {
	db *gorm.DB
}

func NewAuditRepository(db *gorm.DB) AuditRepository {
	return &auditRepository{db: db}
}

func (r *auditRepository) Create(ctx context.Context, entry *model.AuditLog) error {
	return database.FromContext(ctx, r.db).Create(entry).Error
}

func (r *auditRepository) List(ctx context.Context, schoolID *uint64, limit, offset int) ([]model.AuditLog, error) {
	var entries []model.AuditLog
	query := database.FromContext(ctx, r.db).Order("log_time DESC").Limit(limit).Offset(offset)
	if schoolID != nil {
		query = query.Where("sch_no = ?", *schoolID)
	}
	return entries, query.Find(&entries).Error
}
