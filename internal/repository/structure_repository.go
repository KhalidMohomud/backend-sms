package repository

import (
	"backendapi/internal/database"
	"backendapi/internal/model"
	"context"
	"fmt"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type LookupKind string

const (
	LookupJobs                 LookupKind = "jobs"
	LookupDecrees              LookupKind = "decrees"
	LookupSubjects             LookupKind = "subjects"
	LookupExams                LookupKind = "exams"
	LookupPeriods              LookupKind = "periods"
	LookupAttendanceStatus     LookupKind = "attendance-status"
	LookupAttendanceConditions LookupKind = "attendance-conditions"
	LookupStaffStatusTypes     LookupKind = "staff-status-types"
	LookupAmountTypes          LookupKind = "amount-types"
	LookupExpenseTypes         LookupKind = "expense-types"
)

type lookupTable struct {
	table      string
	idColumn   string
	nameColumn string
}

var lookupTables = map[LookupKind]lookupTable{
	LookupJobs:                 {table: "jobs", idColumn: "job_no", nameColumn: "job_name"},
	LookupDecrees:              {table: "decrees", idColumn: "dec_no", nameColumn: "dec_name"},
	LookupSubjects:             {table: "subjects", idColumn: "sub_no", nameColumn: "sub_name"},
	LookupExams:                {table: "exams", idColumn: "ex_no", nameColumn: "ex_name"},
	LookupPeriods:              {table: "periods", idColumn: "per_no", nameColumn: "per_name"},
	LookupAttendanceStatus:     {table: "attendance_status", idColumn: "ast_no", nameColumn: "ast_name"},
	LookupAttendanceConditions: {table: "att_conditions", idColumn: "con_no", nameColumn: "con_name"},
	LookupStaffStatusTypes:     {table: "staff_status_types", idColumn: "sst_no", nameColumn: "sst_name"},
	LookupAmountTypes:          {table: "amount_types", idColumn: "am_no", nameColumn: "am_name"},
	LookupExpenseTypes:         {table: "expense_types", idColumn: "exp_no", nameColumn: "exp_name"},
}

func ParseLookupKind(value string) (LookupKind, bool) {
	kind := LookupKind(strings.ToLower(strings.TrimSpace(value)))
	_, ok := lookupTables[kind]
	return kind, ok
}

func LookupKindNames() []string {
	return []string{
		string(LookupJobs), string(LookupDecrees), string(LookupSubjects), string(LookupExams),
		string(LookupPeriods), string(LookupAttendanceStatus), string(LookupAttendanceConditions),
		string(LookupStaffStatusTypes), string(LookupAmountTypes), string(LookupExpenseTypes),
	}
}

type StructureRepository interface {
	ListLookups(context.Context, LookupKind) ([]model.LookupItem, error)
	FindLookupByID(context.Context, LookupKind, uint64) (*model.LookupItem, error)
	CreateLookup(context.Context, LookupKind, *model.LookupItem) error
	UpdateLookup(context.Context, LookupKind, *model.LookupItem) error
	CreateLevel(context.Context, *model.Level) error
	ListLevels(context.Context, uint64) ([]model.Level, error)
	FindLevelByID(context.Context, uint64, uint64) (*model.Level, error)
	UpdateLevel(context.Context, *model.Level) error
	CountActiveClassesByLevel(context.Context, uint64, uint64) (int64, error)
	CreateClass(context.Context, *model.Class) error
	ListClasses(context.Context, uint64) ([]model.Class, error)
	FindClassByID(context.Context, uint64, uint64) (*model.Class, error)
	UpdateClass(context.Context, *model.Class) error
}

type structureRepository struct{ db *gorm.DB }

func NewStructureRepository(db *gorm.DB) StructureRepository {
	return &structureRepository{db: db}
}

func (r *structureRepository) ListLookups(ctx context.Context, kind LookupKind) ([]model.LookupItem, error) {
	config := lookupTables[kind]
	var items []model.LookupItem
	query := fmt.Sprintf("%s AS id, %s AS name, status, created_at, updated_at", config.idColumn, config.nameColumn)
	err := database.FromContext(ctx, r.db).Table(config.table).Select(query).Order(config.nameColumn).Scan(&items).Error
	return items, err
}

func (r *structureRepository) FindLookupByID(ctx context.Context, kind LookupKind, id uint64) (*model.LookupItem, error) {
	config := lookupTables[kind]
	var item model.LookupItem
	query := fmt.Sprintf("%s AS id, %s AS name, status, created_at, updated_at", config.idColumn, config.nameColumn)
	result := database.FromContext(ctx, r.db).Table(config.table).Select(query).Where(config.idColumn+" = ?", id).Scan(&item)
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected == 0 {
		return nil, ErrNotFound
	}
	return &item, nil
}

func (r *structureRepository) CreateLookup(ctx context.Context, kind LookupKind, item *model.LookupItem) error {
	config := lookupTables[kind]
	statement := fmt.Sprintf(
		"INSERT INTO %s (%s, status) VALUES (?, ?) RETURNING %s AS id, %s AS name, status, created_at, updated_at",
		config.table, config.nameColumn, config.idColumn, config.nameColumn,
	)
	return database.FromContext(ctx, r.db).Raw(statement, item.Name, item.Status).Scan(item).Error
}

func (r *structureRepository) UpdateLookup(ctx context.Context, kind LookupKind, item *model.LookupItem) error {
	config := lookupTables[kind]
	result := database.FromContext(ctx, r.db).Table(config.table).Where(config.idColumn+" = ?", item.ID).
		Updates(map[string]any{config.nameColumn: item.Name, "status": item.Status, "updated_at": gorm.Expr("NOW()")})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	updated, err := r.FindLookupByID(ctx, kind, item.ID)
	if err != nil {
		return err
	}
	*item = *updated
	return nil
}

func (r *structureRepository) CreateLevel(ctx context.Context, level *model.Level) error {
	return database.FromContext(ctx, r.db).Omit(clause.Associations).Create(level).Error
}

func (r *structureRepository) ListLevels(ctx context.Context, schoolID uint64) ([]model.Level, error) {
	var levels []model.Level
	err := database.FromContext(ctx, r.db).Where("sch_no = ?", schoolID).Order("lev_name").Find(&levels).Error
	return levels, err
}

func (r *structureRepository) FindLevelByID(ctx context.Context, schoolID, id uint64) (*model.Level, error) {
	var level model.Level
	if err := database.FromContext(ctx, r.db).Where("sch_no = ?", schoolID).First(&level, "lev_no = ?", id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &level, nil
}

func (r *structureRepository) UpdateLevel(ctx context.Context, level *model.Level) error {
	result := database.FromContext(ctx, r.db).Model(&model.Level{}).
		Where("lev_no = ? AND sch_no = ?", level.ID, level.SchoolID).
		Updates(map[string]any{"lev_name": level.Name, "price": level.Price, "status": level.Status})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *structureRepository) CountActiveClassesByLevel(ctx context.Context, schoolID, levelID uint64) (int64, error) {
	var count int64
	err := database.FromContext(ctx, r.db).Model(&model.Class{}).
		Where("sch_no = ? AND lev_no = ? AND status = ?", schoolID, levelID, model.RecordStatusActive).Count(&count).Error
	return count, err
}

func (r *structureRepository) CreateClass(ctx context.Context, class *model.Class) error {
	return database.FromContext(ctx, r.db).Omit(clause.Associations).Create(class).Error
}

func (r *structureRepository) ListClasses(ctx context.Context, schoolID uint64) ([]model.Class, error) {
	var classes []model.Class
	err := database.FromContext(ctx, r.db).Preload("Level").Where("classes.sch_no = ?", schoolID).Order("cl_name").Find(&classes).Error
	return classes, err
}

func (r *structureRepository) FindClassByID(ctx context.Context, schoolID, id uint64) (*model.Class, error) {
	var class model.Class
	err := database.FromContext(ctx, r.db).Preload("Level").Where("classes.sch_no = ?", schoolID).First(&class, "cl_no = ?", id).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	return &class, nil
}

func (r *structureRepository) UpdateClass(ctx context.Context, class *model.Class) error {
	result := database.FromContext(ctx, r.db).Model(&model.Class{}).
		Where("cl_no = ? AND sch_no = ?", class.ID, class.SchoolID).
		Updates(map[string]any{"lev_no": class.LevelID, "cl_name": class.Name, "status": class.Status})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
