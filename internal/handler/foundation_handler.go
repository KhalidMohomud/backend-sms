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
	case errors.Is(err, service.ErrDuplicateRecord):
		c.JSON(http.StatusConflict, errorResponse{Error: err.Error()})
	case errors.Is(err, service.ErrInvalidDate), errors.Is(err, repository.ErrNotFound):
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})
	default:
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "internal server error"})
	}
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
