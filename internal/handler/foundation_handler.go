package handler

import (
	"backendapi/internal/middleware"
	"backendapi/internal/repository"
	"backendapi/internal/service"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

type FoundationHandler struct{ service *service.FoundationService }

func NewFoundationHandler(service *service.FoundationService) *FoundationHandler {
	return &FoundationHandler{service: service}
}

// CreateSchool godoc
// @Summary Create a school
// @Tags foundation
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body service.CreateSchoolInput true "School details"
// @Success 201 {object} model.School
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Router /schools [post]
func (h *FoundationHandler) CreateSchool(c *gin.Context) {
	var input service.CreateSchoolInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid school request"})
		return
	}
	principal, _ := middleware.Principal(c)
	school, err := h.service.CreateSchool(c.Request.Context(), principal, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, school)
}

// ListSchools godoc
// @Summary List schools
// @Tags foundation
// @Produce json
// @Security BearerAuth
// @Success 200 {array} model.School
// @Router /schools [get]
func (h *FoundationHandler) ListSchools(c *gin.Context) {
	principal, _ := middleware.Principal(c)
	schools, err := h.service.ListSchools(c.Request.Context(), principal)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, schools)
}

// GetSchool godoc
// @Summary Get a school by ID
// @Tags foundation
// @Produce json
// @Security BearerAuth
// @Param id path int true "School ID"
// @Success 200 {object} model.School
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /schools/{id} [get]
func (h *FoundationHandler) GetSchool(c *gin.Context) {
	schoolID, ok := positivePathID(c, "id", "school")
	if !ok {
		return
	}
	principal, _ := middleware.Principal(c)
	school, err := h.service.GetSchool(c.Request.Context(), principal, schoolID)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, school)
}

// UpdateSchool godoc
// @Summary Update a school
// @Description Partially updates a school. Setting status to inactive immediately blocks its school accounts.
// @Tags foundation
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "School ID"
// @Param payload body service.UpdateSchoolInput true "Fields to update"
// @Success 200 {object} model.School
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Router /schools/{id} [patch]
func (h *FoundationHandler) UpdateSchool(c *gin.Context) {
	schoolID, ok := positivePathID(c, "id", "school")
	if !ok {
		return
	}
	var input service.UpdateSchoolInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid school update request"})
		return
	}
	principal, _ := middleware.Principal(c)
	school, err := h.service.UpdateSchool(c.Request.Context(), principal, schoolID, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, school)
}

// ArchiveSchool godoc
// @Summary Deactivate a school
// @Description Safe delete: sets the school to inactive instead of removing historical data.
// @Tags foundation
// @Security BearerAuth
// @Param id path int true "School ID"
// @Success 204
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /schools/{id} [delete]
func (h *FoundationHandler) ArchiveSchool(c *gin.Context) {
	schoolID, ok := positivePathID(c, "id", "school")
	if !ok {
		return
	}
	principal, _ := middleware.Principal(c)
	if err := h.service.ArchiveSchool(c.Request.Context(), principal, schoolID, requestMetadata(c)); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// CreateAcademicYear godoc
// @Summary Create an academic year for the resolved school
// @Tags foundation
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param payload body service.CreateAcademicYearInput true "Academic year details"
// @Success 201 {object} model.AcademicYear
// @Router /academic-years [post]
func (h *FoundationHandler) CreateAcademicYear(c *gin.Context) {
	var input service.CreateAcademicYearInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid academic year request"})
		return
	}
	principal, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	year, err := h.service.CreateAcademicYear(c.Request.Context(), principal, schoolID, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, year)
}

// ListAcademicYears godoc
// @Summary List academic years for the resolved school
// @Tags foundation
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Success 200 {array} model.AcademicYear
// @Router /academic-years [get]
func (h *FoundationHandler) ListAcademicYears(c *gin.Context) {
	principal, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	years, err := h.service.ListAcademicYears(c.Request.Context(), principal, schoolID)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, years)
}

// UpdateAcademicYear godoc
// @Summary Update an academic year
// @Tags foundation
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Academic year ID"
// @Param payload body service.UpdateAcademicYearInput true "Fields to update"
// @Success 200 {object} model.AcademicYear
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Router /academic-years/{id} [patch]
func (h *FoundationHandler) UpdateAcademicYear(c *gin.Context) {
	yearID, ok := positivePathID(c, "id", "academic year")
	if !ok {
		return
	}
	var input service.UpdateAcademicYearInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid academic year update request"})
		return
	}
	principal, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	year, err := h.service.UpdateAcademicYear(c.Request.Context(), principal, schoolID, yearID, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, year)
}

// DeleteAcademicYear godoc
// @Summary Delete an academic year
// @Description Permanently deletes an academic year only when no dependent records use it.
// @Tags foundation
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Academic year ID"
// @Success 204
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Router /academic-years/{id} [delete]
func (h *FoundationHandler) DeleteAcademicYear(c *gin.Context) {
	yearID, ok := positivePathID(c, "id", "academic year")
	if !ok {
		return
	}
	principal, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	if err := h.service.DeleteAcademicYear(c.Request.Context(), principal, schoolID, yearID, requestMetadata(c)); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// CreateUser godoc
// @Summary Create a school user
// @Tags foundation
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body service.CreateUserInput true "User details"
// @Success 201 {object} model.User
// @Router /users [post]
func (h *FoundationHandler) CreateUser(c *gin.Context) {
	var input service.CreateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid user request"})
		return
	}
	principal, _ := middleware.Principal(c)
	user, err := h.service.CreateUser(c.Request.Context(), principal, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, user)
}

// UpdateUser godoc
// @Summary Update a user account
// @Description Updates supplied username, staff link, or role fields within the permitted school scope.
// @Tags foundation
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param payload body service.UpdateUserInput true "Fields to update"
// @Success 200 {object} model.User
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Failure 409 {object} errorResponse
// @Router /users/{id} [patch]
func (h *FoundationHandler) UpdateUser(c *gin.Context) {
	userID, ok := positivePathID(c, "id", "user")
	if !ok {
		return
	}
	var input service.UpdateUserInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid user update request"})
		return
	}
	principal, _ := middleware.Principal(c)
	user, err := h.service.UpdateUser(c.Request.Context(), principal, userID, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, user)
}

// UpdateUserStatus godoc
// @Summary Disable or unlock a user account
// @Tags foundation
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Param payload body service.UpdateUserStatusInput true "New account status"
// @Success 200 {object} model.User
// @Router /users/{id}/status [patch]
func (h *FoundationHandler) UpdateUserStatus(c *gin.Context) {
	userID, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil || userID == 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid user ID"})
		return
	}
	var input service.UpdateUserStatusInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid user status request"})
		return
	}
	principal, _ := middleware.Principal(c)
	user, err := h.service.UpdateUserStatus(c.Request.Context(), principal, userID, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, user)
}

// DisableUser godoc
// @Summary Disable a user account
// @Description Safe delete: disables login while preserving ownership and audit history.
// @Tags foundation
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 204
// @Failure 401 {object} errorResponse
// @Failure 403 {object} errorResponse
// @Failure 404 {object} errorResponse
// @Router /users/{id} [delete]
func (h *FoundationHandler) DisableUser(c *gin.Context) {
	userID, ok := positivePathID(c, "id", "user")
	if !ok {
		return
	}
	principal, _ := middleware.Principal(c)
	if err := h.service.DisableUser(c.Request.Context(), principal, userID, requestMetadata(c)); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// CreatePasswordResetToken godoc
// @Summary Create a one-time password reset token for a managed user
// @Tags foundation
// @Produce json
// @Security BearerAuth
// @Param id path int true "User ID"
// @Success 201 {object} service.PasswordResetTokenResult
// @Router /users/{id}/password-reset-token [post]
func (h *FoundationHandler) CreatePasswordResetToken(c *gin.Context) {
	userID, ok := positivePathID(c, "id", "user")
	if !ok {
		return
	}
	principal, _ := middleware.Principal(c)
	result, err := h.service.CreatePasswordResetToken(c.Request.Context(), principal, userID, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, result)
}

// ListUsers godoc
// @Summary List users in the permitted school scope
// @Tags foundation
// @Produce json
// @Security BearerAuth
// @Param school_id query int false "Optional SuperAdmin filter"
// @Success 200 {array} model.User
// @Router /users [get]
func (h *FoundationHandler) ListUsers(c *gin.Context) {
	principal, _ := middleware.Principal(c)
	schoolID, err := optionalUint64(c.Query("school_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "school_id must be a positive integer"})
		return
	}
	users, err := h.service.ListUsers(c.Request.Context(), principal, schoolID)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, users)
}

// ListRoles godoc
// @Summary List roles and their permissions
// @Tags foundation
// @Produce json
// @Security BearerAuth
// @Success 200 {array} model.Role
// @Router /roles [get]
func (h *FoundationHandler) ListRoles(c *gin.Context) {
	principal, _ := middleware.Principal(c)
	roles, err := h.service.ListRoles(c.Request.Context(), principal)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, roles)
}

// CreateRole godoc
// @Summary Create a custom role
// @Tags foundation
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param payload body service.CreateRoleInput true "Role and permissions"
// @Success 201 {object} model.Role
// @Router /roles [post]
func (h *FoundationHandler) CreateRole(c *gin.Context) {
	var input service.CreateRoleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid role request"})
		return
	}
	principal, _ := middleware.Principal(c)
	role, err := h.service.CreateRole(c.Request.Context(), principal, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, role)
}

// UpdateRole godoc
// @Summary Update a custom role
// @Tags foundation
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Role ID"
// @Param payload body service.UpdateRoleInput true "Fields to update"
// @Success 200 {object} model.Role
// @Router /roles/{id} [patch]
func (h *FoundationHandler) UpdateRole(c *gin.Context) {
	roleID, ok := positivePathID(c, "id", "role")
	if !ok {
		return
	}
	var input service.UpdateRoleInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid role update request"})
		return
	}
	principal, _ := middleware.Principal(c)
	role, err := h.service.UpdateRole(c.Request.Context(), principal, roleID, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, role)
}

// ArchiveRole godoc
// @Summary Deactivate a custom role
// @Tags foundation
// @Security BearerAuth
// @Param id path int true "Role ID"
// @Success 204
// @Router /roles/{id} [delete]
func (h *FoundationHandler) ArchiveRole(c *gin.Context) {
	roleID, ok := positivePathID(c, "id", "role")
	if !ok {
		return
	}
	principal, _ := middleware.Principal(c)
	if err := h.service.ArchiveRole(c.Request.Context(), principal, roleID, requestMetadata(c)); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ReplaceRolePermissions godoc
// @Summary Replace a custom role's permissions
// @Tags foundation
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Role ID"
// @Param payload body service.ReplaceRolePermissionsInput true "Complete permission assignment"
// @Success 200 {object} model.Role
// @Router /roles/{id}/permissions [put]
func (h *FoundationHandler) ReplaceRolePermissions(c *gin.Context) {
	roleID, ok := positivePathID(c, "id", "role")
	if !ok {
		return
	}
	var input service.ReplaceRolePermissionsInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid permission assignment"})
		return
	}
	principal, _ := middleware.Principal(c)
	role, err := h.service.ReplaceRolePermissions(c.Request.Context(), principal, roleID, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, role)
}

// ListPermissions godoc
// @Summary List permissions
// @Tags foundation
// @Produce json
// @Security BearerAuth
// @Success 200 {array} model.Permission
// @Router /permissions [get]
func (h *FoundationHandler) ListPermissions(c *gin.Context) {
	principal, _ := middleware.Principal(c)
	permissions, err := h.service.ListPermissions(c.Request.Context(), principal)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, permissions)
}

// ListAuditLogs godoc
// @Summary List append-only audit logs
// @Tags foundation
// @Produce json
// @Security BearerAuth
// @Param school_id query int false "Optional SuperAdmin filter"
// @Param limit query int false "Page size, maximum 100"
// @Param offset query int false "Pagination offset"
// @Success 200 {array} model.AuditLog
// @Router /audit-logs [get]
func (h *FoundationHandler) ListAuditLogs(c *gin.Context) {
	principal, _ := middleware.Principal(c)
	schoolID, err := optionalUint64(c.Query("school_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "school_id must be a positive integer"})
		return
	}
	limit := boundedInt(c.Query("limit"), 50, 1, 100)
	offset := boundedInt(c.Query("offset"), 0, 0, 1_000_000)
	entries, err := h.service.ListAuditLogs(c.Request.Context(), principal, schoolID, limit, offset)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, entries)
}

func writeServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrForbidden), errors.Is(err, service.ErrInvalidScope):
		c.JSON(http.StatusForbidden, errorResponse{Error: err.Error()})
	case errors.Is(err, service.ErrDuplicateRecord), errors.Is(err, service.ErrConflict):
		c.JSON(http.StatusConflict, errorResponse{Error: err.Error()})
	case errors.Is(err, service.ErrInvalidDate), errors.Is(err, service.ErrNoChanges), errors.Is(err, service.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
	case errors.Is(err, repository.ErrNotFound):
		c.JSON(http.StatusNotFound, errorResponse{Error: err.Error()})
	case errors.Is(err, service.ErrProtectedRecord):
		c.JSON(http.StatusForbidden, errorResponse{Error: err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
}

func positivePathID(c *gin.Context, parameter, resource string) (uint64, bool) {
	id, err := strconv.ParseUint(c.Param(parameter), 10, 64)
	if err != nil || id == 0 {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid " + resource + " ID"})
		return 0, false
	}
	return id, true
}

func optionalUint64(value string) (*uint64, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	parsed, err := strconv.ParseUint(value, 10, 64)
	if err != nil || parsed == 0 {
		return nil, errors.New("invalid positive integer")
	}
	return &parsed, nil
}

func boundedInt(value string, fallback, minimum, maximum int) int {
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < minimum {
		return fallback
	}
	if parsed > maximum {
		return maximum
	}
	return parsed
}
