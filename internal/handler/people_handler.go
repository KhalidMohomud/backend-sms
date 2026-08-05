package handler

import (
	"backendapi/internal/middleware"
	"backendapi/internal/repository"
	"backendapi/internal/service"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

type PeopleHandler struct{ service *service.PeopleService }

func NewPeopleHandler(service *service.PeopleService) *PeopleHandler {
	return &PeopleHandler{service: service}
}

// ListAddresses godoc
// @Summary List addresses for the resolved school
// @Tags people
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param search query string false "District, village, or area search"
// @Param limit query int false "Page size, maximum 100"
// @Param offset query int false "Pagination offset"
// @Success 200 {array} model.Address
// @Router /addresses [get]
func (h *PeopleHandler) ListAddresses(c *gin.Context) {
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	rows, err := h.service.ListAddresses(c.Request.Context(), actor, schoolID, peopleListOptions(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, rows)
}

// GetAddress godoc
// @Summary Get an address by ID
// @Tags people
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Address ID"
// @Success 200 {object} model.Address
// @Router /addresses/{id} [get]
func (h *PeopleHandler) GetAddress(c *gin.Context) {
	id, ok := positivePathID(c, "id", "address")
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.GetAddress(c.Request.Context(), actor, schoolID, id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, row)
}

// CreateAddress godoc
// @Summary Create an address
// @Tags people
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param payload body service.CreateAddressInput true "Address details"
// @Success 201 {object} model.Address
// @Router /addresses [post]
func (h *PeopleHandler) CreateAddress(c *gin.Context) {
	var input service.CreateAddressInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid address request"})
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.CreateAddress(c.Request.Context(), actor, schoolID, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, row)
}

// UpdateAddress godoc
// @Summary Update an address
// @Tags people
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Address ID"
// @Param payload body service.UpdateAddressInput true "Fields to update"
// @Success 200 {object} model.Address
// @Router /addresses/{id} [patch]
func (h *PeopleHandler) UpdateAddress(c *gin.Context) {
	id, ok := positivePathID(c, "id", "address")
	if !ok {
		return
	}
	var input service.UpdateAddressInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid address update request"})
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.UpdateAddress(c.Request.Context(), actor, schoolID, id, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, row)
}

// DeleteAddress godoc
// @Summary Delete an unused address
// @Tags people
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Address ID"
// @Success 204
// @Router /addresses/{id} [delete]
func (h *PeopleHandler) DeleteAddress(c *gin.Context) {
	id, ok := positivePathID(c, "id", "address")
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	if err := h.service.DeleteAddress(c.Request.Context(), actor, schoolID, id, requestMetadata(c)); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListGuardians godoc
// @Summary List guardians for the resolved school
// @Tags people
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param search query string false "Name or phone search"
// @Param limit query int false "Page size, maximum 100"
// @Param offset query int false "Pagination offset"
// @Success 200 {array} model.Responsible
// @Router /guardians [get]
func (h *PeopleHandler) ListGuardians(c *gin.Context) {
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	rows, err := h.service.ListResponsibles(c.Request.Context(), actor, schoolID, peopleListOptions(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, rows)
}

// GetGuardian godoc
// @Summary Get a guardian by ID
// @Tags people
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Guardian ID"
// @Success 200 {object} model.Responsible
// @Router /guardians/{id} [get]
func (h *PeopleHandler) GetGuardian(c *gin.Context) {
	id, ok := positivePathID(c, "id", "guardian")
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.GetResponsible(c.Request.Context(), actor, schoolID, id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, row)
}

// CreateGuardian godoc
// @Summary Create a guardian
// @Tags people
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param payload body service.CreateResponsibleInput true "Guardian details"
// @Success 201 {object} model.Responsible
// @Router /guardians [post]
func (h *PeopleHandler) CreateGuardian(c *gin.Context) {
	var input service.CreateResponsibleInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid guardian request"})
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.CreateResponsible(c.Request.Context(), actor, schoolID, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, row)
}

// UpdateGuardian godoc
// @Summary Update a guardian
// @Tags people
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Guardian ID"
// @Param payload body service.UpdateResponsibleInput true "Fields to update"
// @Success 200 {object} model.Responsible
// @Router /guardians/{id} [patch]
func (h *PeopleHandler) UpdateGuardian(c *gin.Context) {
	id, ok := positivePathID(c, "id", "guardian")
	if !ok {
		return
	}
	var input service.UpdateResponsibleInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid guardian update request"})
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.UpdateResponsible(c.Request.Context(), actor, schoolID, id, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, row)
}

// DeleteGuardian godoc
// @Summary Delete an unused guardian
// @Tags people
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Guardian ID"
// @Success 204
// @Router /guardians/{id} [delete]
func (h *PeopleHandler) DeleteGuardian(c *gin.Context) {
	id, ok := positivePathID(c, "id", "guardian")
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	if err := h.service.DeleteResponsible(c.Request.Context(), actor, schoolID, id, requestMetadata(c)); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListStudents godoc
// @Summary List students for the resolved school
// @Tags people
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param search query string false "Student, mother, or phone search"
// @Param status query string false "active, graduated, transferred, or dropped"
// @Param limit query int false "Page size, maximum 100"
// @Param offset query int false "Pagination offset"
// @Success 200 {array} model.Student
// @Router /students [get]
func (h *PeopleHandler) ListStudents(c *gin.Context) {
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	rows, err := h.service.ListStudents(c.Request.Context(), actor, schoolID, peopleListOptions(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, rows)
}

// GetStudent godoc
// @Summary Get a student by ID
// @Tags people
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Student ID"
// @Success 200 {object} model.Student
// @Router /students/{id} [get]
func (h *PeopleHandler) GetStudent(c *gin.Context) {
	id, ok := positivePathID(c, "id", "student")
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.GetStudent(c.Request.Context(), actor, schoolID, id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, row)
}

// CreateStudent godoc
// @Summary Create a student
// @Tags people
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param payload body service.CreateStudentInput true "Student details"
// @Success 201 {object} model.Student
// @Router /students [post]
func (h *PeopleHandler) CreateStudent(c *gin.Context) {
	var input service.CreateStudentInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid student request"})
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.CreateStudent(c.Request.Context(), actor, schoolID, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, row)
}

// UpdateStudent godoc
// @Summary Update a student
// @Tags people
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Student ID"
// @Param payload body service.UpdateStudentInput true "Fields to update"
// @Success 200 {object} model.Student
// @Router /students/{id} [patch]
func (h *PeopleHandler) UpdateStudent(c *gin.Context) {
	id, ok := positivePathID(c, "id", "student")
	if !ok {
		return
	}
	var input service.UpdateStudentInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid student update request"})
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.UpdateStudent(c.Request.Context(), actor, schoolID, id, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, row)
}

// ArchiveStudent godoc
// @Summary Archive a student as dropped
// @Tags people
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Student ID"
// @Success 204
// @Router /students/{id} [delete]
func (h *PeopleHandler) ArchiveStudent(c *gin.Context) {
	id, ok := positivePathID(c, "id", "student")
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	if err := h.service.ArchiveStudent(c.Request.Context(), actor, schoolID, id, requestMetadata(c)); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListStaff godoc
// @Summary List staff for the resolved school
// @Tags people
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param search query string false "Staff name or phone search"
// @Param limit query int false "Page size, maximum 100"
// @Param offset query int false "Pagination offset"
// @Success 200 {array} model.Staff
// @Router /staff [get]
func (h *PeopleHandler) ListStaff(c *gin.Context) {
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	rows, err := h.service.ListStaff(c.Request.Context(), actor, schoolID, peopleListOptions(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, rows)
}

// GetStaff godoc
// @Summary Get a staff member by ID
// @Tags people
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Staff ID"
// @Success 200 {object} model.Staff
// @Router /staff/{id} [get]
func (h *PeopleHandler) GetStaff(c *gin.Context) {
	id, ok := positivePathID(c, "id", "staff")
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.GetStaff(c.Request.Context(), actor, schoolID, id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, row)
}

// CreateStaff godoc
// @Summary Create a staff member
// @Tags people
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param payload body service.CreateStaffInput true "Staff details"
// @Success 201 {object} model.Staff
// @Router /staff [post]
func (h *PeopleHandler) CreateStaff(c *gin.Context) {
	var input service.CreateStaffInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid staff request"})
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.CreateStaff(c.Request.Context(), actor, schoolID, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, row)
}

// UpdateStaff godoc
// @Summary Update a staff member
// @Tags people
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Staff ID"
// @Param payload body service.UpdateStaffInput true "Fields to update"
// @Success 200 {object} model.Staff
// @Router /staff/{id} [patch]
func (h *PeopleHandler) UpdateStaff(c *gin.Context) {
	id, ok := positivePathID(c, "id", "staff")
	if !ok {
		return
	}
	var input service.UpdateStaffInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid staff update request"})
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.UpdateStaff(c.Request.Context(), actor, schoolID, id, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, row)
}

// ArchiveStaff godoc
// @Summary Archive a staff member by appending Resigned status
// @Tags people
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Staff ID"
// @Success 204
// @Router /staff/{id} [delete]
func (h *PeopleHandler) ArchiveStaff(c *gin.Context) {
	id, ok := positivePathID(c, "id", "staff")
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	if err := h.service.ArchiveStaff(c.Request.Context(), actor, schoolID, id, requestMetadata(c)); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListStaffStatuses godoc
// @Summary List employment status history for a staff member
// @Tags people
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Staff ID"
// @Success 200 {array} model.StaffStatus
// @Router /staff/{id}/statuses [get]
func (h *PeopleHandler) ListStaffStatuses(c *gin.Context) {
	id, ok := positivePathID(c, "id", "staff")
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	rows, err := h.service.ListStaffStatuses(c.Request.Context(), actor, schoolID, id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, rows)
}

// CreateStaffStatus godoc
// @Summary Append an employment status event
// @Tags people
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Staff ID"
// @Param payload body service.CreateStaffStatusInput true "Status event"
// @Success 201 {object} model.StaffStatus
// @Router /staff/{id}/statuses [post]
func (h *PeopleHandler) CreateStaffStatus(c *gin.Context) {
	id, ok := positivePathID(c, "id", "staff")
	if !ok {
		return
	}
	var input service.CreateStaffStatusInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid staff status request"})
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.CreateStaffStatus(c.Request.Context(), actor, schoolID, id, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, row)
}

func peopleListOptions(c *gin.Context) repository.PeopleListOptions {
	return repository.PeopleListOptions{
		Search: strings.TrimSpace(c.Query("search")), Status: strings.ToLower(strings.TrimSpace(c.Query("status"))),
		Limit: boundedInt(c.Query("limit"), 50, 1, 100), Offset: boundedInt(c.Query("offset"), 0, 0, 1_000_000),
	}
}
