package handler

import (
	"backendapi/internal/middleware"
	"backendapi/internal/repository"
	"backendapi/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type StructureHandler struct{ service *service.StructureService }

func NewStructureHandler(service *service.StructureService) *StructureHandler {
	return &StructureHandler{service: service}
}

// ListLookups godoc
// @Summary List global lookup values
// @Tags structure
// @Produce json
// @Security BearerAuth
// @Param type path string true "Lookup type" Enums(jobs,decrees,subjects,exams,periods,attendance-status,attendance-conditions,staff-status-types,amount-types,expense-types)
// @Success 200 {array} model.LookupItem
// @Router /lookups/{type} [get]
func (h *StructureHandler) ListLookups(c *gin.Context) {
	kind, ok := lookupKind(c)
	if !ok {
		return
	}
	items, err := h.service.ListLookups(c.Request.Context(), kind)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, items)
}

// GetLookup godoc
// @Summary Get a global lookup value
// @Tags structure
// @Produce json
// @Security BearerAuth
// @Param type path string true "Lookup type"
// @Param id path int true "Lookup ID"
// @Success 200 {object} model.LookupItem
// @Failure 404 {object} errorResponse
// @Router /lookups/{type}/{id} [get]
func (h *StructureHandler) GetLookup(c *gin.Context) {
	kind, ok := lookupKind(c)
	if !ok {
		return
	}
	id, ok := positivePathID(c, "id", "lookup")
	if !ok {
		return
	}
	item, err := h.service.GetLookup(c.Request.Context(), kind, id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

// CreateLookup godoc
// @Summary Create a global lookup value
// @Tags structure
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param type path string true "Lookup type"
// @Param payload body service.CreateLookupInput true "Lookup value"
// @Success 201 {object} model.LookupItem
// @Router /lookups/{type} [post]
func (h *StructureHandler) CreateLookup(c *gin.Context) {
	kind, ok := lookupKind(c)
	if !ok {
		return
	}
	var input service.CreateLookupInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid lookup request"})
		return
	}
	actor, _ := middleware.Principal(c)
	item, err := h.service.CreateLookup(c.Request.Context(), actor, kind, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, item)
}

// UpdateLookup godoc
// @Summary Update or reactivate a global lookup value
// @Tags structure
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param type path string true "Lookup type"
// @Param id path int true "Lookup ID"
// @Param payload body service.UpdateLookupInput true "Fields to update"
// @Success 200 {object} model.LookupItem
// @Router /lookups/{type}/{id} [patch]
func (h *StructureHandler) UpdateLookup(c *gin.Context) {
	kind, ok := lookupKind(c)
	if !ok {
		return
	}
	id, ok := positivePathID(c, "id", "lookup")
	if !ok {
		return
	}
	var input service.UpdateLookupInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid lookup update request"})
		return
	}
	actor, _ := middleware.Principal(c)
	item, err := h.service.UpdateLookup(c.Request.Context(), actor, kind, id, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, item)
}

// ArchiveLookup godoc
// @Summary Deactivate a global lookup value
// @Tags structure
// @Security BearerAuth
// @Param type path string true "Lookup type"
// @Param id path int true "Lookup ID"
// @Success 204
// @Router /lookups/{type}/{id} [delete]
func (h *StructureHandler) ArchiveLookup(c *gin.Context) {
	kind, ok := lookupKind(c)
	if !ok {
		return
	}
	id, ok := positivePathID(c, "id", "lookup")
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	if err := h.service.ArchiveLookup(c.Request.Context(), actor, kind, id, requestMetadata(c)); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListLevels godoc
// @Summary List levels for the resolved school
// @Tags structure
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Success 200 {array} model.Level
// @Router /levels [get]
func (h *StructureHandler) ListLevels(c *gin.Context) {
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	levels, err := h.service.ListLevels(c.Request.Context(), actor, schoolID)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, levels)
}

// GetLevel godoc
// @Summary Get a level from the resolved school
// @Tags structure
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Level ID"
// @Success 200 {object} model.Level
// @Router /levels/{id} [get]
func (h *StructureHandler) GetLevel(c *gin.Context) {
	id, ok := positivePathID(c, "id", "level")
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	level, err := h.service.GetLevel(c.Request.Context(), actor, schoolID, id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, level)
}

// CreateLevel godoc
// @Summary Create a level for the resolved school
// @Tags structure
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param payload body service.CreateLevelInput true "Level details"
// @Success 201 {object} model.Level
// @Router /levels [post]
func (h *StructureHandler) CreateLevel(c *gin.Context) {
	var input service.CreateLevelInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid level request"})
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	level, err := h.service.CreateLevel(c.Request.Context(), actor, schoolID, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, level)
}

// UpdateLevel godoc
// @Summary Update a level
// @Tags structure
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Level ID"
// @Param payload body service.UpdateLevelInput true "Fields to update"
// @Success 200 {object} model.Level
// @Router /levels/{id} [patch]
func (h *StructureHandler) UpdateLevel(c *gin.Context) {
	id, ok := positivePathID(c, "id", "level")
	if !ok {
		return
	}
	var input service.UpdateLevelInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid level update request"})
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	level, err := h.service.UpdateLevel(c.Request.Context(), actor, schoolID, id, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, level)
}

// ArchiveLevel godoc
// @Summary Deactivate a level
// @Tags structure
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Level ID"
// @Success 204
// @Router /levels/{id} [delete]
func (h *StructureHandler) ArchiveLevel(c *gin.Context) {
	id, ok := positivePathID(c, "id", "level")
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	if err := h.service.ArchiveLevel(c.Request.Context(), actor, schoolID, id, requestMetadata(c)); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListClasses godoc
// @Summary List classes for the resolved school
// @Tags structure
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Success 200 {array} model.Class
// @Router /classes [get]
func (h *StructureHandler) ListClasses(c *gin.Context) {
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	classes, err := h.service.ListClasses(c.Request.Context(), actor, schoolID)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, classes)
}

// GetClass godoc
// @Summary Get a class from the resolved school
// @Tags structure
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Class ID"
// @Success 200 {object} model.Class
// @Router /classes/{id} [get]
func (h *StructureHandler) GetClass(c *gin.Context) {
	id, ok := positivePathID(c, "id", "class")
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	class, err := h.service.GetClass(c.Request.Context(), actor, schoolID, id)
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, class)
}

// CreateClass godoc
// @Summary Create a class for the resolved school
// @Tags structure
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param payload body service.CreateClassInput true "Class details"
// @Success 201 {object} model.Class
// @Router /classes [post]
func (h *StructureHandler) CreateClass(c *gin.Context) {
	var input service.CreateClassInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid class request"})
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	class, err := h.service.CreateClass(c.Request.Context(), actor, schoolID, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, class)
}

// UpdateClass godoc
// @Summary Update a class
// @Tags structure
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Class ID"
// @Param payload body service.UpdateClassInput true "Fields to update"
// @Success 200 {object} model.Class
// @Router /classes/{id} [patch]
func (h *StructureHandler) UpdateClass(c *gin.Context) {
	id, ok := positivePathID(c, "id", "class")
	if !ok {
		return
	}
	var input service.UpdateClassInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid class update request"})
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	class, err := h.service.UpdateClass(c.Request.Context(), actor, schoolID, id, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, class)
}

// ArchiveClass godoc
// @Summary Deactivate a class
// @Tags structure
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Class ID"
// @Success 204
// @Router /classes/{id} [delete]
func (h *StructureHandler) ArchiveClass(c *gin.Context) {
	id, ok := positivePathID(c, "id", "class")
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	if err := h.service.ArchiveClass(c.Request.Context(), actor, schoolID, id, requestMetadata(c)); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func lookupKind(c *gin.Context) (repository.LookupKind, bool) {
	kind, ok := repository.ParseLookupKind(c.Param("type"))
	if !ok {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid lookup type"})
		return "", false
	}
	return kind, true
}
