package repository

import (
	"backendapi/internal/database"
	"backendapi/internal/model"
	"context"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PeopleListOptions struct {
	Search string
	Status string
	Limit  int
	Offset int
}

type PeopleRepository interface {
	CreateAddress(context.Context, *model.Address) error
	ListAddresses(context.Context, uint64, PeopleListOptions) ([]model.Address, error)
	FindAddressByID(context.Context, uint64, uint64) (*model.Address, error)
	UpdateAddress(context.Context, *model.Address) error
	DeleteAddress(context.Context, uint64, uint64) error
	CountAddressReferences(context.Context, uint64, uint64) (int64, error)

	CreateResponsible(context.Context, *model.Responsible) error
	ListResponsibles(context.Context, uint64, PeopleListOptions) ([]model.Responsible, error)
	FindResponsibleByID(context.Context, uint64, uint64) (*model.Responsible, error)
	UpdateResponsible(context.Context, *model.Responsible) error
	DeleteResponsible(context.Context, uint64, uint64) error
	CountResponsibleReferences(context.Context, uint64, uint64) (int64, error)

	CreateStudent(context.Context, *model.Student) error
	ListStudents(context.Context, uint64, PeopleListOptions) ([]model.Student, error)
	FindStudentByID(context.Context, uint64, uint64) (*model.Student, error)
	UpdateStudent(context.Context, *model.Student) error

	CreateStaff(context.Context, *model.Staff) error
	ListStaff(context.Context, uint64, PeopleListOptions) ([]model.Staff, error)
	FindStaffByID(context.Context, uint64, uint64) (*model.Staff, error)
	UpdateStaff(context.Context, *model.Staff) error
	FindJobByID(context.Context, uint64) (*model.Job, error)
	FindDecreeByID(context.Context, uint64) (*model.Decree, error)

	CreateStaffStatus(context.Context, *model.StaffStatus) error
	ListStaffStatuses(context.Context, uint64, uint64) ([]model.StaffStatus, error)
	FindStaffStatusTypeByID(context.Context, uint64) (*model.StaffStatusType, error)
	FindStaffStatusTypeByName(context.Context, string) (*model.StaffStatusType, error)
	FindLatestStaffStatus(context.Context, uint64, uint64) (*model.StaffStatus, error)
}

type peopleRepository struct{ db *gorm.DB }

func NewPeopleRepository(db *gorm.DB) PeopleRepository { return &peopleRepository{db: db} }

func (r *peopleRepository) CreateAddress(ctx context.Context, address *model.Address) error {
	return database.FromContext(ctx, r.db).Create(address).Error
}

func (r *peopleRepository) ListAddresses(ctx context.Context, schoolID uint64, options PeopleListOptions) ([]model.Address, error) {
	var rows []model.Address
	query := page(database.FromContext(ctx, r.db).Where("sch_no = ?", schoolID), options)
	if search := normalizedSearch(options.Search); search != "" {
		query = query.Where("district ILIKE ? OR village ILIKE ? OR area ILIKE ?", search, search, search)
	}
	return rows, query.Order("district, village, area, add_no").Find(&rows).Error
}

func (r *peopleRepository) FindAddressByID(ctx context.Context, schoolID, id uint64) (*model.Address, error) {
	var row model.Address
	err := database.FromContext(ctx, r.db).Where("sch_no = ?", schoolID).First(&row, "add_no = ?", id).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	return &row, nil
}

func (r *peopleRepository) UpdateAddress(ctx context.Context, address *model.Address) error {
	result := database.FromContext(ctx, r.db).Model(&model.Address{}).
		Where("add_no = ? AND sch_no = ?", address.ID, address.SchoolID).
		Updates(map[string]any{"district": address.District, "village": address.Village, "area": address.Area})
	return changed(result)
}

func (r *peopleRepository) DeleteAddress(ctx context.Context, schoolID, id uint64) error {
	return changed(database.FromContext(ctx, r.db).Where("add_no = ? AND sch_no = ?", id, schoolID).Delete(&model.Address{}))
}

func (r *peopleRepository) CountAddressReferences(ctx context.Context, schoolID, id uint64) (int64, error) {
	db := database.FromContext(ctx, r.db)
	var students, staff int64
	if err := db.Model(&model.Student{}).Where("sch_no = ? AND add_no = ?", schoolID, id).Count(&students).Error; err != nil {
		return 0, err
	}
	if err := db.Model(&model.Staff{}).Where("sch_no = ? AND add_no = ?", schoolID, id).Count(&staff).Error; err != nil {
		return 0, err
	}
	return students + staff, nil
}

func (r *peopleRepository) CreateResponsible(ctx context.Context, responsible *model.Responsible) error {
	return database.FromContext(ctx, r.db).Create(responsible).Error
}

func (r *peopleRepository) ListResponsibles(ctx context.Context, schoolID uint64, options PeopleListOptions) ([]model.Responsible, error) {
	var rows []model.Responsible
	query := page(database.FromContext(ctx, r.db).Where("sch_no = ?", schoolID), options)
	if search := normalizedSearch(options.Search); search != "" {
		query = query.Where("res_name ILIKE ? OR res_tell ILIKE ?", search, search)
	}
	return rows, query.Order("res_name, res_no").Find(&rows).Error
}

func (r *peopleRepository) FindResponsibleByID(ctx context.Context, schoolID, id uint64) (*model.Responsible, error) {
	var row model.Responsible
	err := database.FromContext(ctx, r.db).Where("sch_no = ?", schoolID).First(&row, "res_no = ?", id).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	return &row, nil
}

func (r *peopleRepository) UpdateResponsible(ctx context.Context, responsible *model.Responsible) error {
	result := database.FromContext(ctx, r.db).Model(&model.Responsible{}).
		Where("res_no = ? AND sch_no = ?", responsible.ID, responsible.SchoolID).
		Updates(map[string]any{"res_name": responsible.Name, "res_tell": responsible.Phone, "relationship": responsible.Relationship})
	return changed(result)
}

func (r *peopleRepository) DeleteResponsible(ctx context.Context, schoolID, id uint64) error {
	return changed(database.FromContext(ctx, r.db).Where("res_no = ? AND sch_no = ?", id, schoolID).Delete(&model.Responsible{}))
}

func (r *peopleRepository) CountResponsibleReferences(ctx context.Context, schoolID, id uint64) (int64, error) {
	var count int64
	err := database.FromContext(ctx, r.db).Model(&model.Student{}).Where("sch_no = ? AND res_no = ?", schoolID, id).Count(&count).Error
	return count, err
}

func (r *peopleRepository) CreateStudent(ctx context.Context, student *model.Student) error {
	return database.FromContext(ctx, r.db).Omit(clause.Associations).Create(student).Error
}

func (r *peopleRepository) ListStudents(ctx context.Context, schoolID uint64, options PeopleListOptions) ([]model.Student, error) {
	var rows []model.Student
	query := database.FromContext(ctx, r.db).Preload("Address").Preload("Responsible").Where("students.sch_no = ?", schoolID)
	query = page(query, options)
	if search := normalizedSearch(options.Search); search != "" {
		query = query.Where("std_name ILIKE ? OR mother_name ILIKE ? OR tell ILIKE ?", search, search, search)
	}
	if options.Status != "" {
		query = query.Where("students.status = ?", options.Status)
	}
	return rows, query.Order("std_name, std_id").Find(&rows).Error
}

func (r *peopleRepository) FindStudentByID(ctx context.Context, schoolID, id uint64) (*model.Student, error) {
	var row model.Student
	err := database.FromContext(ctx, r.db).Preload("Address").Preload("Responsible").
		Where("students.sch_no = ?", schoolID).First(&row, "std_id = ?", id).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	return &row, nil
}

func (r *peopleRepository) UpdateStudent(ctx context.Context, student *model.Student) error {
	result := database.FromContext(ctx, r.db).Model(&model.Student{}).
		Where("std_id = ? AND sch_no = ?", student.ID, student.SchoolID).
		Updates(map[string]any{
			"std_name": student.Name, "mother_name": student.MotherName, "sex": student.Sex,
			"tell": student.Phone, "b_date": student.BirthDate, "p_birth": student.BirthPlace,
			"add_no": student.AddressID, "res_no": student.ResponsibleID, "image": student.Image,
			"reg_date": student.RegisteredOn, "status": student.Status,
		})
	return changed(result)
}

func (r *peopleRepository) CreateStaff(ctx context.Context, staff *model.Staff) error {
	return database.FromContext(ctx, r.db).Omit(clause.Associations).Create(staff).Error
}

func (r *peopleRepository) ListStaff(ctx context.Context, schoolID uint64, options PeopleListOptions) ([]model.Staff, error) {
	var rows []model.Staff
	query := database.FromContext(ctx, r.db).Preload("Address").Preload("Job").Preload("Decree").Where("staff.sch_no = ?", schoolID)
	query = page(query, options)
	if search := normalizedSearch(options.Search); search != "" {
		query = query.Where("stf_name ILIKE ? OR tell ILIKE ?", search, search)
	}
	return rows, query.Order("stf_name, stf_no").Find(&rows).Error
}

func (r *peopleRepository) FindStaffByID(ctx context.Context, schoolID, id uint64) (*model.Staff, error) {
	var row model.Staff
	err := database.FromContext(ctx, r.db).Preload("Address").Preload("Job").Preload("Decree").
		Where("staff.sch_no = ?", schoolID).First(&row, "stf_no = ?", id).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	return &row, nil
}

func (r *peopleRepository) UpdateStaff(ctx context.Context, staff *model.Staff) error {
	result := database.FromContext(ctx, r.db).Model(&model.Staff{}).
		Where("stf_no = ? AND sch_no = ?", staff.ID, staff.SchoolID).
		Updates(map[string]any{
			"stf_name": staff.Name, "sex": staff.Sex, "tell": staff.Phone, "add_no": staff.AddressID,
			"job_no": staff.JobID, "dec_no": staff.DecreeID, "salary": staff.Salary,
			"hired_date": staff.HiredDate, "description": staff.Description,
		})
	return changed(result)
}

func (r *peopleRepository) FindJobByID(ctx context.Context, id uint64) (*model.Job, error) {
	var row model.Job
	if err := database.FromContext(ctx, r.db).First(&row, "job_no = ?", id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &row, nil
}

func (r *peopleRepository) FindDecreeByID(ctx context.Context, id uint64) (*model.Decree, error) {
	var row model.Decree
	if err := database.FromContext(ctx, r.db).First(&row, "dec_no = ?", id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &row, nil
}

func (r *peopleRepository) CreateStaffStatus(ctx context.Context, status *model.StaffStatus) error {
	return database.FromContext(ctx, r.db).Omit(clause.Associations).Create(status).Error
}

func (r *peopleRepository) ListStaffStatuses(ctx context.Context, schoolID, staffID uint64) ([]model.StaffStatus, error) {
	var rows []model.StaffStatus
	err := database.FromContext(ctx, r.db).Preload("StatusType").
		Where("staff_status.sch_no = ? AND staff_status.stf_no = ?", schoolID, staffID).
		Order("st_date DESC, ss_no DESC").Find(&rows).Error
	return rows, err
}

func (r *peopleRepository) FindStaffStatusTypeByID(ctx context.Context, id uint64) (*model.StaffStatusType, error) {
	var row model.StaffStatusType
	if err := database.FromContext(ctx, r.db).First(&row, "sst_no = ?", id).Error; err != nil {
		return nil, mapNotFound(err)
	}
	return &row, nil
}

func (r *peopleRepository) FindStaffStatusTypeByName(ctx context.Context, name string) (*model.StaffStatusType, error) {
	var row model.StaffStatusType
	err := database.FromContext(ctx, r.db).Where("LOWER(sst_name) = ?", strings.ToLower(strings.TrimSpace(name))).First(&row).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	return &row, nil
}

func (r *peopleRepository) FindLatestStaffStatus(ctx context.Context, schoolID, staffID uint64) (*model.StaffStatus, error) {
	var row model.StaffStatus
	err := database.FromContext(ctx, r.db).Preload("StatusType").
		Where("staff_status.sch_no = ? AND staff_status.stf_no = ?", schoolID, staffID).
		Order("st_date DESC, ss_no DESC").First(&row).Error
	if err != nil {
		return nil, mapNotFound(err)
	}
	return &row, nil
}

func page(query *gorm.DB, options PeopleListOptions) *gorm.DB {
	limit := options.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if options.Offset < 0 {
		options.Offset = 0
	}
	return query.Limit(limit).Offset(options.Offset)
}

func normalizedSearch(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	return "%" + value + "%"
}

func changed(result *gorm.DB) error {
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	return nil
}
