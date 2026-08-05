package service

import (
	"backendapi/internal/authz"
	"backendapi/internal/model"
	"backendapi/internal/repository"
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type CreateAddressInput struct {
	District string `json:"district" binding:"required,min=2,max=60" example:"Hodan"`
	Village  string `json:"village" binding:"max=60" example:"Taleex"`
	Area     string `json:"area" binding:"max=60" example:"Zone 1"`
}

type UpdateAddressInput struct {
	District *string `json:"district" binding:"omitempty,min=2,max=60"`
	Village  *string `json:"village" binding:"omitempty,max=60"`
	Area     *string `json:"area" binding:"omitempty,max=60"`
}

type CreateResponsibleInput struct {
	Name         string `json:"name" binding:"required,min=2,max=100" example:"Ahmed Ali"`
	Phone        string `json:"phone" binding:"required,min=5,max=20" example:"+252610000000"`
	Relationship string `json:"relationship" binding:"required,min=2,max=40" example:"Father"`
}

type UpdateResponsibleInput struct {
	Name         *string `json:"name" binding:"omitempty,min=2,max=100"`
	Phone        *string `json:"phone" binding:"omitempty,min=5,max=20"`
	Relationship *string `json:"relationship" binding:"omitempty,min=2,max=40"`
}

type CreateStudentInput struct {
	Name         string              `json:"name" binding:"required,min=2,max=100" example:"Amina Ahmed"`
	MotherName   string              `json:"mother_name" binding:"required,min=2,max=100" example:"Hodan Ali"`
	Sex          model.Sex           `json:"sex" binding:"required,oneof=M F" example:"F"`
	Phone        string              `json:"phone" binding:"max=20"`
	BirthDate    string              `json:"birth_date" binding:"omitempty" example:"2012-04-10"`
	BirthPlace   string              `json:"birth_place" binding:"max=60" example:"Mogadishu"`
	AddressID    *uint64             `json:"address_id" binding:"omitempty,min=1"`
	GuardianID   uint64              `json:"guardian_id" binding:"required,min=1"`
	Image        string              `json:"image" binding:"max=255"`
	RegisteredOn string              `json:"registered_on" binding:"required" example:"2026-08-05"`
	Status       model.StudentStatus `json:"status" binding:"omitempty,oneof=active graduated transferred dropped" example:"active"`
}

type UpdateStudentInput struct {
	Name         *string              `json:"name" binding:"omitempty,min=2,max=100"`
	MotherName   *string              `json:"mother_name" binding:"omitempty,min=2,max=100"`
	Sex          *model.Sex           `json:"sex" binding:"omitempty,oneof=M F"`
	Phone        *string              `json:"phone" binding:"omitempty,max=20"`
	BirthDate    *string              `json:"birth_date" binding:"omitempty"`
	BirthPlace   *string              `json:"birth_place" binding:"omitempty,max=60"`
	AddressID    *uint64              `json:"address_id" binding:"omitempty,min=1"`
	ClearAddress bool                 `json:"clear_address"`
	GuardianID   *uint64              `json:"guardian_id" binding:"omitempty,min=1"`
	Image        *string              `json:"image" binding:"omitempty,max=255"`
	RegisteredOn *string              `json:"registered_on" binding:"omitempty"`
	Status       *model.StudentStatus `json:"status" binding:"omitempty,oneof=active graduated transferred dropped"`
}

type CreateStaffInput struct {
	Name        string    `json:"name" binding:"required,min=2,max=100" example:"Mohamed Ali"`
	Sex         model.Sex `json:"sex" binding:"required,oneof=M F" example:"M"`
	Phone       string    `json:"phone" binding:"max=20"`
	AddressID   *uint64   `json:"address_id" binding:"omitempty,min=1"`
	JobID       uint64    `json:"job_id" binding:"required,min=1"`
	DecreeID    uint64    `json:"decree_id" binding:"required,min=1"`
	Salary      float64   `json:"salary" binding:"gte=0" example:"500"`
	HiredDate   string    `json:"hired_date" binding:"required" example:"2026-08-05"`
	Description string    `json:"description" binding:"max=255"`
}

type UpdateStaffInput struct {
	Name         *string    `json:"name" binding:"omitempty,min=2,max=100"`
	Sex          *model.Sex `json:"sex" binding:"omitempty,oneof=M F"`
	Phone        *string    `json:"phone" binding:"omitempty,max=20"`
	AddressID    *uint64    `json:"address_id" binding:"omitempty,min=1"`
	ClearAddress bool       `json:"clear_address"`
	JobID        *uint64    `json:"job_id" binding:"omitempty,min=1"`
	DecreeID     *uint64    `json:"decree_id" binding:"omitempty,min=1"`
	Salary       *float64   `json:"salary" binding:"omitempty,gte=0"`
	HiredDate    *string    `json:"hired_date" binding:"omitempty"`
	Description  *string    `json:"description" binding:"omitempty,max=255"`
}

type CreateStaffStatusInput struct {
	StatusTypeID uint64 `json:"status_type_id" binding:"required,min=1"`
	Description  string `json:"description" binding:"max=255"`
	StatusDate   string `json:"status_date" binding:"required" example:"2026-08-05"`
}

type PeopleService struct {
	people     repository.PeopleRepository
	foundation repository.FoundationRepository
	audit      *AuditWriter
}

func NewPeopleService(people repository.PeopleRepository, foundation repository.FoundationRepository, audit *AuditWriter) *PeopleService {
	return &PeopleService{people: people, foundation: foundation, audit: audit}
}

func (s *PeopleService) ListAddresses(ctx context.Context, actor authz.Principal, schoolID uint64, options repository.PeopleListOptions) ([]model.Address, error) {
	if !canManagePeople(actor, schoolID) {
		return nil, ErrForbidden
	}
	return s.people.ListAddresses(ctx, schoolID, options)
}

func (s *PeopleService) GetAddress(ctx context.Context, actor authz.Principal, schoolID, id uint64) (*model.Address, error) {
	if !canManagePeople(actor, schoolID) {
		return nil, ErrForbidden
	}
	return s.people.FindAddressByID(ctx, schoolID, id)
}

func (s *PeopleService) CreateAddress(ctx context.Context, actor authz.Principal, schoolID uint64, input CreateAddressInput, request RequestMetadata) (*model.Address, error) {
	if !canManagePeople(actor, schoolID) {
		return nil, ErrForbidden
	}
	if err := s.requireActiveSchool(ctx, schoolID); err != nil {
		return nil, err
	}
	district := strings.TrimSpace(input.District)
	if err := validateName(district, 60); err != nil {
		return nil, err
	}
	address := &model.Address{SchoolID: schoolID, District: district, Village: strings.TrimSpace(input.Village), Area: strings.TrimSpace(input.Area)}
	if err := s.people.CreateAddress(ctx, address); err != nil {
		return nil, peopleWriteError("create address", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "INSERT", "addresses", &address.ID, request, nil); err != nil {
		return nil, err
	}
	return address, nil
}

func (s *PeopleService) UpdateAddress(ctx context.Context, actor authz.Principal, schoolID, id uint64, input UpdateAddressInput, request RequestMetadata) (*model.Address, error) {
	if !canManagePeople(actor, schoolID) {
		return nil, ErrForbidden
	}
	if input.District == nil && input.Village == nil && input.Area == nil {
		return nil, ErrNoChanges
	}
	address, err := s.people.FindAddressByID(ctx, schoolID, id)
	if err != nil {
		return nil, err
	}
	if input.District != nil {
		address.District = strings.TrimSpace(*input.District)
		if err := validateName(address.District, 60); err != nil {
			return nil, err
		}
	}
	if input.Village != nil {
		address.Village = strings.TrimSpace(*input.Village)
	}
	if input.Area != nil {
		address.Area = strings.TrimSpace(*input.Area)
	}
	if err := s.people.UpdateAddress(ctx, address); err != nil {
		return nil, peopleWriteError("update address", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "UPDATE", "addresses", &address.ID, request, nil); err != nil {
		return nil, err
	}
	return address, nil
}

func (s *PeopleService) DeleteAddress(ctx context.Context, actor authz.Principal, schoolID, id uint64, request RequestMetadata) error {
	if !canManagePeople(actor, schoolID) {
		return ErrForbidden
	}
	if _, err := s.people.FindAddressByID(ctx, schoolID, id); err != nil {
		return err
	}
	count, err := s.people.CountAddressReferences(ctx, schoolID, id)
	if err != nil {
		return err
	}
	if count != 0 {
		return ErrConflict
	}
	if err := s.people.DeleteAddress(ctx, schoolID, id); err != nil {
		return peopleWriteError("delete address", err)
	}
	return s.audit.Write(ctx, &actor.UserID, &schoolID, "DELETE", "addresses", &id, request, nil)
}

func (s *PeopleService) ListResponsibles(ctx context.Context, actor authz.Principal, schoolID uint64, options repository.PeopleListOptions) ([]model.Responsible, error) {
	if !canManageStudents(actor, schoolID) {
		return nil, ErrForbidden
	}
	return s.people.ListResponsibles(ctx, schoolID, options)
}

func (s *PeopleService) GetResponsible(ctx context.Context, actor authz.Principal, schoolID, id uint64) (*model.Responsible, error) {
	if !canManageStudents(actor, schoolID) {
		return nil, ErrForbidden
	}
	return s.people.FindResponsibleByID(ctx, schoolID, id)
}

func (s *PeopleService) CreateResponsible(ctx context.Context, actor authz.Principal, schoolID uint64, input CreateResponsibleInput, request RequestMetadata) (*model.Responsible, error) {
	if !canManageStudents(actor, schoolID) {
		return nil, ErrForbidden
	}
	if err := s.requireActiveSchool(ctx, schoolID); err != nil {
		return nil, err
	}
	name, relationship := strings.TrimSpace(input.Name), strings.TrimSpace(input.Relationship)
	if err := validateName(name, 100); err != nil {
		return nil, err
	}
	if err := validateName(relationship, 40); err != nil {
		return nil, err
	}
	phone := strings.TrimSpace(input.Phone)
	if len(phone) < 5 {
		return nil, ErrInvalidInput
	}
	row := &model.Responsible{SchoolID: schoolID, Name: name, Phone: phone, Relationship: relationship}
	if err := s.people.CreateResponsible(ctx, row); err != nil {
		return nil, peopleWriteError("create guardian", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "INSERT", "responsibles", &row.ID, request, nil); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *PeopleService) UpdateResponsible(ctx context.Context, actor authz.Principal, schoolID, id uint64, input UpdateResponsibleInput, request RequestMetadata) (*model.Responsible, error) {
	if !canManageStudents(actor, schoolID) {
		return nil, ErrForbidden
	}
	if input.Name == nil && input.Phone == nil && input.Relationship == nil {
		return nil, ErrNoChanges
	}
	row, err := s.people.FindResponsibleByID(ctx, schoolID, id)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		row.Name = strings.TrimSpace(*input.Name)
		if err := validateName(row.Name, 100); err != nil {
			return nil, err
		}
	}
	if input.Phone != nil {
		row.Phone = strings.TrimSpace(*input.Phone)
		if len(row.Phone) < 5 {
			return nil, ErrInvalidInput
		}
	}
	if input.Relationship != nil {
		row.Relationship = strings.TrimSpace(*input.Relationship)
		if err := validateName(row.Relationship, 40); err != nil {
			return nil, err
		}
	}
	if err := s.people.UpdateResponsible(ctx, row); err != nil {
		return nil, peopleWriteError("update guardian", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "UPDATE", "responsibles", &row.ID, request, nil); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *PeopleService) DeleteResponsible(ctx context.Context, actor authz.Principal, schoolID, id uint64, request RequestMetadata) error {
	if !canManageStudents(actor, schoolID) {
		return ErrForbidden
	}
	if _, err := s.people.FindResponsibleByID(ctx, schoolID, id); err != nil {
		return err
	}
	count, err := s.people.CountResponsibleReferences(ctx, schoolID, id)
	if err != nil {
		return err
	}
	if count != 0 {
		return ErrConflict
	}
	if err := s.people.DeleteResponsible(ctx, schoolID, id); err != nil {
		return peopleWriteError("delete guardian", err)
	}
	return s.audit.Write(ctx, &actor.UserID, &schoolID, "DELETE", "responsibles", &id, request, nil)
}

func (s *PeopleService) ListStudents(ctx context.Context, actor authz.Principal, schoolID uint64, options repository.PeopleListOptions) ([]model.Student, error) {
	if !canManageStudents(actor, schoolID) {
		return nil, ErrForbidden
	}
	if options.Status != "" && !validStudentStatus(model.StudentStatus(options.Status)) {
		return nil, ErrInvalidInput
	}
	return s.people.ListStudents(ctx, schoolID, options)
}

func (s *PeopleService) GetStudent(ctx context.Context, actor authz.Principal, schoolID, id uint64) (*model.Student, error) {
	if !canManageStudents(actor, schoolID) {
		return nil, ErrForbidden
	}
	return s.people.FindStudentByID(ctx, schoolID, id)
}

func (s *PeopleService) CreateStudent(ctx context.Context, actor authz.Principal, schoolID uint64, input CreateStudentInput, request RequestMetadata) (*model.Student, error) {
	if !canManageStudents(actor, schoolID) {
		return nil, ErrForbidden
	}
	if err := s.requireActiveSchool(ctx, schoolID); err != nil {
		return nil, err
	}
	name, mother := strings.TrimSpace(input.Name), strings.TrimSpace(input.MotherName)
	if err := validateName(name, 100); err != nil {
		return nil, err
	}
	if err := validateName(mother, 100); err != nil {
		return nil, err
	}
	registeredOn, err := requiredDate(input.RegisteredOn)
	if err != nil {
		return nil, err
	}
	if registeredOn.After(todayUTC()) {
		return nil, ErrInvalidDate
	}
	birthDate, err := optionalDate(input.BirthDate)
	if err != nil || (birthDate != nil && !birthDate.Before(todayUTC())) {
		return nil, ErrInvalidDate
	}
	if _, err := s.people.FindResponsibleByID(ctx, schoolID, input.GuardianID); err != nil {
		return nil, err
	}
	if err := s.validateAddress(ctx, schoolID, input.AddressID); err != nil {
		return nil, err
	}
	status := input.Status
	if status == "" {
		status = model.StudentStatusActive
	}
	row := &model.Student{
		SchoolID: schoolID, Name: name, MotherName: mother, Sex: input.Sex, Phone: strings.TrimSpace(input.Phone),
		BirthDate: birthDate, BirthPlace: strings.TrimSpace(input.BirthPlace), AddressID: input.AddressID,
		ResponsibleID: input.GuardianID, Image: strings.TrimSpace(input.Image), RegisteredOn: registeredOn, Status: status,
	}
	if err := s.people.CreateStudent(ctx, row); err != nil {
		return nil, peopleWriteError("create student", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "INSERT", "students", &row.ID, request, nil); err != nil {
		return nil, err
	}
	return s.people.FindStudentByID(ctx, schoolID, row.ID)
}

func (s *PeopleService) UpdateStudent(ctx context.Context, actor authz.Principal, schoolID, id uint64, input UpdateStudentInput, request RequestMetadata) (*model.Student, error) {
	if !canManageStudents(actor, schoolID) {
		return nil, ErrForbidden
	}
	if studentUpdateEmpty(input) {
		return nil, ErrNoChanges
	}
	row, err := s.people.FindStudentByID(ctx, schoolID, id)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		row.Name = strings.TrimSpace(*input.Name)
		if err := validateName(row.Name, 100); err != nil {
			return nil, err
		}
	}
	if input.MotherName != nil {
		row.MotherName = strings.TrimSpace(*input.MotherName)
		if err := validateName(row.MotherName, 100); err != nil {
			return nil, err
		}
	}
	if input.Sex != nil {
		row.Sex = *input.Sex
	}
	if input.Phone != nil {
		row.Phone = strings.TrimSpace(*input.Phone)
	}
	if input.BirthDate != nil {
		row.BirthDate, err = optionalDate(*input.BirthDate)
		if err != nil || (row.BirthDate != nil && !row.BirthDate.Before(todayUTC())) {
			return nil, ErrInvalidDate
		}
	}
	if input.BirthPlace != nil {
		row.BirthPlace = strings.TrimSpace(*input.BirthPlace)
	}
	if input.AddressID != nil && input.ClearAddress {
		return nil, ErrInvalidInput
	}
	if input.ClearAddress {
		row.AddressID = nil
	} else if input.AddressID != nil {
		if err := s.validateAddress(ctx, schoolID, input.AddressID); err != nil {
			return nil, err
		}
		row.AddressID = input.AddressID
	}
	if input.GuardianID != nil {
		if _, err := s.people.FindResponsibleByID(ctx, schoolID, *input.GuardianID); err != nil {
			return nil, err
		}
		row.ResponsibleID = *input.GuardianID
	}
	if input.Image != nil {
		row.Image = strings.TrimSpace(*input.Image)
	}
	if input.RegisteredOn != nil {
		row.RegisteredOn, err = requiredDate(*input.RegisteredOn)
		if err != nil {
			return nil, err
		}
		if row.RegisteredOn.After(todayUTC()) {
			return nil, ErrInvalidDate
		}
	}
	if input.Status != nil {
		row.Status = *input.Status
	}
	if err := s.people.UpdateStudent(ctx, row); err != nil {
		return nil, peopleWriteError("update student", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "UPDATE", "students", &row.ID, request, nil); err != nil {
		return nil, err
	}
	return s.people.FindStudentByID(ctx, schoolID, row.ID)
}

func (s *PeopleService) ArchiveStudent(ctx context.Context, actor authz.Principal, schoolID, id uint64, request RequestMetadata) error {
	if !canManageStudents(actor, schoolID) {
		return ErrForbidden
	}
	row, err := s.people.FindStudentByID(ctx, schoolID, id)
	if err != nil {
		return err
	}
	row.Status = model.StudentStatusDropped
	if err := s.people.UpdateStudent(ctx, row); err != nil {
		return peopleWriteError("archive student", err)
	}
	return s.audit.Write(ctx, &actor.UserID, &schoolID, "DELETE", "students", &row.ID, request, map[string]any{"soft_delete": true, "status": row.Status})
}

func (s *PeopleService) ListStaff(ctx context.Context, actor authz.Principal, schoolID uint64, options repository.PeopleListOptions) ([]model.Staff, error) {
	if !canManageStaff(actor, schoolID) {
		return nil, ErrForbidden
	}
	return s.people.ListStaff(ctx, schoolID, options)
}

func (s *PeopleService) GetStaff(ctx context.Context, actor authz.Principal, schoolID, id uint64) (*model.Staff, error) {
	if !canManageStaff(actor, schoolID) {
		return nil, ErrForbidden
	}
	return s.people.FindStaffByID(ctx, schoolID, id)
}

func (s *PeopleService) CreateStaff(ctx context.Context, actor authz.Principal, schoolID uint64, input CreateStaffInput, request RequestMetadata) (*model.Staff, error) {
	if !canManageStaff(actor, schoolID) {
		return nil, ErrForbidden
	}
	if err := s.requireActiveSchool(ctx, schoolID); err != nil {
		return nil, err
	}
	name := strings.TrimSpace(input.Name)
	if err := validateName(name, 100); err != nil {
		return nil, err
	}
	hiredDate, err := requiredDate(input.HiredDate)
	if err != nil || hiredDate.After(time.Now().UTC()) {
		return nil, ErrInvalidDate
	}
	if err := s.validateAddress(ctx, schoolID, input.AddressID); err != nil {
		return nil, err
	}
	job, err := s.people.FindJobByID(ctx, input.JobID)
	if err != nil || job.Status != model.RecordStatusActive {
		return nil, ErrInvalidScope
	}
	decree, err := s.people.FindDecreeByID(ctx, input.DecreeID)
	if err != nil || decree.Status != model.RecordStatusActive {
		return nil, ErrInvalidScope
	}
	row := &model.Staff{SchoolID: schoolID, Name: name, Sex: input.Sex, Phone: strings.TrimSpace(input.Phone), AddressID: input.AddressID, JobID: job.ID, DecreeID: decree.ID, Salary: input.Salary, HiredDate: hiredDate, Description: strings.TrimSpace(input.Description)}
	if err := s.people.CreateStaff(ctx, row); err != nil {
		return nil, peopleWriteError("create staff", err)
	}
	active, err := s.people.FindStaffStatusTypeByName(ctx, "Active")
	if err != nil || active.Status != model.RecordStatusActive {
		return nil, ErrInvalidScope
	}
	initialStatus := &model.StaffStatus{SchoolID: schoolID, StaffID: row.ID, StaffStatusTypeID: active.ID, StatusType: *active, StatusDate: hiredDate, Description: "Initial employment status"}
	if err := s.people.CreateStaffStatus(ctx, initialStatus); err != nil {
		return nil, peopleWriteError("create initial staff status", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "INSERT", "staff", &row.ID, request, nil); err != nil {
		return nil, err
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "INSERT", "staff_status", &initialStatus.ID, request, map[string]any{"initial": true}); err != nil {
		return nil, err
	}
	return s.people.FindStaffByID(ctx, schoolID, row.ID)
}

func (s *PeopleService) UpdateStaff(ctx context.Context, actor authz.Principal, schoolID, id uint64, input UpdateStaffInput, request RequestMetadata) (*model.Staff, error) {
	if !canManageStaff(actor, schoolID) {
		return nil, ErrForbidden
	}
	if staffUpdateEmpty(input) {
		return nil, ErrNoChanges
	}
	row, err := s.people.FindStaffByID(ctx, schoolID, id)
	if err != nil {
		return nil, err
	}
	if input.Name != nil {
		row.Name = strings.TrimSpace(*input.Name)
		if err := validateName(row.Name, 100); err != nil {
			return nil, err
		}
	}
	if input.Sex != nil {
		row.Sex = *input.Sex
	}
	if input.Phone != nil {
		row.Phone = strings.TrimSpace(*input.Phone)
	}
	if input.AddressID != nil && input.ClearAddress {
		return nil, ErrInvalidInput
	}
	if input.ClearAddress {
		row.AddressID = nil
	} else if input.AddressID != nil {
		if err := s.validateAddress(ctx, schoolID, input.AddressID); err != nil {
			return nil, err
		}
		row.AddressID = input.AddressID
	}
	if input.JobID != nil {
		job, err := s.people.FindJobByID(ctx, *input.JobID)
		if err != nil || job.Status != model.RecordStatusActive {
			return nil, ErrInvalidScope
		}
		row.JobID = job.ID
	}
	if input.DecreeID != nil {
		decree, err := s.people.FindDecreeByID(ctx, *input.DecreeID)
		if err != nil || decree.Status != model.RecordStatusActive {
			return nil, ErrInvalidScope
		}
		row.DecreeID = decree.ID
	}
	if input.Salary != nil {
		row.Salary = *input.Salary
	}
	if input.HiredDate != nil {
		row.HiredDate, err = requiredDate(*input.HiredDate)
		if err != nil || row.HiredDate.After(time.Now().UTC()) {
			return nil, ErrInvalidDate
		}
	}
	if input.Description != nil {
		row.Description = strings.TrimSpace(*input.Description)
	}
	if err := s.people.UpdateStaff(ctx, row); err != nil {
		return nil, peopleWriteError("update staff", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "UPDATE", "staff", &row.ID, request, nil); err != nil {
		return nil, err
	}
	return s.people.FindStaffByID(ctx, schoolID, row.ID)
}

func (s *PeopleService) ListStaffStatuses(ctx context.Context, actor authz.Principal, schoolID, staffID uint64) ([]model.StaffStatus, error) {
	if !canManageStaff(actor, schoolID) {
		return nil, ErrForbidden
	}
	if _, err := s.people.FindStaffByID(ctx, schoolID, staffID); err != nil {
		return nil, err
	}
	return s.people.ListStaffStatuses(ctx, schoolID, staffID)
}

func (s *PeopleService) CreateStaffStatus(ctx context.Context, actor authz.Principal, schoolID, staffID uint64, input CreateStaffStatusInput, request RequestMetadata) (*model.StaffStatus, error) {
	if !canManageStaff(actor, schoolID) {
		return nil, ErrForbidden
	}
	if _, err := s.people.FindStaffByID(ctx, schoolID, staffID); err != nil {
		return nil, err
	}
	statusType, err := s.people.FindStaffStatusTypeByID(ctx, input.StatusTypeID)
	if err != nil || statusType.Status != model.RecordStatusActive {
		return nil, ErrInvalidScope
	}
	statusDate, err := requiredDate(input.StatusDate)
	if err != nil {
		return nil, err
	}
	if statusDate.After(todayUTC()) {
		return nil, ErrInvalidDate
	}
	row := &model.StaffStatus{SchoolID: schoolID, StaffID: staffID, StaffStatusTypeID: statusType.ID, StatusType: *statusType, Description: strings.TrimSpace(input.Description), StatusDate: statusDate}
	if err := s.people.CreateStaffStatus(ctx, row); err != nil {
		return nil, peopleWriteError("create staff status", err)
	}
	if err := s.audit.Write(ctx, &actor.UserID, &schoolID, "INSERT", "staff_status", &row.ID, request, nil); err != nil {
		return nil, err
	}
	return row, nil
}

func (s *PeopleService) ArchiveStaff(ctx context.Context, actor authz.Principal, schoolID, staffID uint64, request RequestMetadata) error {
	if !canManageStaff(actor, schoolID) {
		return ErrForbidden
	}
	if _, err := s.people.FindStaffByID(ctx, schoolID, staffID); err != nil {
		return err
	}
	resigned, err := s.people.FindStaffStatusTypeByName(ctx, "Resigned")
	if err != nil || resigned.Status != model.RecordStatusActive {
		return ErrInvalidScope
	}
	latest, latestErr := s.people.FindLatestStaffStatus(ctx, schoolID, staffID)
	if latestErr == nil && latest.StaffStatusTypeID == resigned.ID {
		return nil
	}
	if latestErr != nil && !errors.Is(latestErr, repository.ErrNotFound) {
		return latestErr
	}
	today := todayUTC()
	row := &model.StaffStatus{SchoolID: schoolID, StaffID: staffID, StaffStatusTypeID: resigned.ID, Description: "Archived through staff endpoint", StatusDate: today}
	if err := s.people.CreateStaffStatus(ctx, row); err != nil {
		return peopleWriteError("archive staff", err)
	}
	return s.audit.Write(ctx, &actor.UserID, &schoolID, "DELETE", "staff", &staffID, request, map[string]any{"soft_delete": true, "staff_status": "Resigned"})
}

func (s *PeopleService) requireActiveSchool(ctx context.Context, schoolID uint64) error {
	school, err := s.foundation.FindSchoolByID(ctx, schoolID)
	if err != nil {
		return err
	}
	if school.Status != model.SchoolStatusActive {
		return ErrInvalidScope
	}
	return nil
}

func (s *PeopleService) validateAddress(ctx context.Context, schoolID uint64, addressID *uint64) error {
	if addressID == nil {
		return nil
	}
	_, err := s.people.FindAddressByID(ctx, schoolID, *addressID)
	return err
}

func canManageStudents(actor authz.Principal, schoolID uint64) bool {
	return canAccessSchool(actor, schoolID) && actor.HasPermission(model.PermissionManageStudents)
}

func canManageStaff(actor authz.Principal, schoolID uint64) bool {
	return canAccessSchool(actor, schoolID) && actor.HasPermission(model.PermissionManageStaff)
}

func canManagePeople(actor authz.Principal, schoolID uint64) bool {
	return canAccessSchool(actor, schoolID) && (actor.HasPermission(model.PermissionManageStudents) || actor.HasPermission(model.PermissionManageStaff))
}

func requiredDate(value string) (time.Time, error) {
	parsed, err := time.Parse("2006-01-02", strings.TrimSpace(value))
	if err != nil {
		return time.Time{}, ErrInvalidDate
	}
	return parsed, nil
}

func todayUTC() time.Time {
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func optionalDate(value string) (*time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return nil, nil
	}
	parsed, err := requiredDate(value)
	if err != nil {
		return nil, err
	}
	return &parsed, nil
}

func validStudentStatus(status model.StudentStatus) bool {
	switch status {
	case model.StudentStatusActive, model.StudentStatusGraduated, model.StudentStatusTransferred, model.StudentStatusDropped:
		return true
	default:
		return false
	}
}

func studentUpdateEmpty(input UpdateStudentInput) bool {
	return input.Name == nil && input.MotherName == nil && input.Sex == nil && input.Phone == nil && input.BirthDate == nil && input.BirthPlace == nil && input.AddressID == nil && !input.ClearAddress && input.GuardianID == nil && input.Image == nil && input.RegisteredOn == nil && input.Status == nil
}

func staffUpdateEmpty(input UpdateStaffInput) bool {
	return input.Name == nil && input.Sex == nil && input.Phone == nil && input.AddressID == nil && !input.ClearAddress && input.JobID == nil && input.DecreeID == nil && input.Salary == nil && input.HiredDate == nil && input.Description == nil
}

func peopleWriteError(operation string, err error) error {
	switch {
	case errors.Is(err, gorm.ErrDuplicatedKey):
		return ErrDuplicateRecord
	case errors.Is(err, gorm.ErrForeignKeyViolated):
		return ErrInvalidScope
	default:
		return fmt.Errorf("%s: %w", operation, err)
	}
}
