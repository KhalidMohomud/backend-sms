package handler

import (
	"backendapi/internal/middleware"
	"backendapi/internal/repository"
	"backendapi/internal/service"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AcademicHandler struct{ service *service.AcademicService }

func NewAcademicHandler(service *service.AcademicService) *AcademicHandler {
	return &AcademicHandler{service: service}
}

// ListStudentClasses godoc
// @Summary List student enrollments
// @Tags academic
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param academic_year_id query int false "Academic year ID"
// @Param class_id query int false "Class ID"
// @Param student_id query int false "Student ID"
// @Param search query string false "Student name search"
// @Param limit query int false "Page size, maximum 100"
// @Param offset query int false "Pagination offset"
// @Success 200 {array} model.StudentClass
// @Router /student-classes [get]
func (h *AcademicHandler) ListStudentClasses(c *gin.Context) {
	options, ok := academicListOptions(c)
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	rows, err := h.service.ListStudentClasses(c.Request.Context(), actor, schoolID, options)
	writeAcademicResult(c, rows, err)
}

// GetStudentClass godoc
// @Summary Get a student enrollment
// @Tags academic
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Enrollment ID"
// @Success 200 {object} model.StudentClass
// @Router /student-classes/{id} [get]
func (h *AcademicHandler) GetStudentClass(c *gin.Context) {
	id, ok := positivePathID(c, "id", "enrollment")
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.GetStudentClass(c.Request.Context(), actor, schoolID, id)
	writeAcademicResult(c, row, err)
}

// CreateStudentClass godoc
// @Summary Enroll a student in a class and academic year
// @Tags academic
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param payload body service.CreateStudentClassInput true "Enrollment details"
// @Success 201 {object} model.StudentClass
// @Router /student-classes [post]
func (h *AcademicHandler) CreateStudentClass(c *gin.Context) {
	var input service.CreateStudentClassInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid enrollment request"})
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.CreateStudentClass(c.Request.Context(), actor, schoolID, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, row)
}

// UpdateStudentClass godoc
// @Summary Update a student enrollment
// @Tags academic
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Enrollment ID"
// @Param payload body service.UpdateStudentClassInput true "Fields to update"
// @Success 200 {object} model.StudentClass
// @Router /student-classes/{id} [patch]
func (h *AcademicHandler) UpdateStudentClass(c *gin.Context) {
	id, ok := positivePathID(c, "id", "enrollment")
	if !ok {
		return
	}
	var input service.UpdateStudentClassInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid enrollment update request"})
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.UpdateStudentClass(c.Request.Context(), actor, schoolID, id, input, requestMetadata(c))
	writeAcademicResult(c, row, err)
}

// DeleteStudentClass godoc
// @Summary Delete an unused student enrollment
// @Tags academic
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Enrollment ID"
// @Success 204
// @Router /student-classes/{id} [delete]
func (h *AcademicHandler) DeleteStudentClass(c *gin.Context) {
	id, ok := positivePathID(c, "id", "enrollment")
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	if err := h.service.DeleteStudentClass(c.Request.Context(), actor, schoolID, id, requestMetadata(c)); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListSubjectClasses godoc
// @Summary List class subject and teacher assignments
// @Tags academic
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param academic_year_id query int false "Academic year ID"
// @Param class_id query int false "Class ID"
// @Param subject_id query int false "Subject ID"
// @Param staff_id query int false "Teacher staff ID"
// @Success 200 {array} model.SubjectClass
// @Router /subject-classes [get]
func (h *AcademicHandler) ListSubjectClasses(c *gin.Context) {
	options, ok := academicListOptions(c)
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	rows, err := h.service.ListSubjectClasses(c.Request.Context(), actor, schoolID, options)
	writeAcademicResult(c, rows, err)
}

// GetSubjectClass godoc
// @Summary Get a class subject assignment
// @Tags academic
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Assignment ID"
// @Success 200 {object} model.SubjectClass
// @Router /subject-classes/{id} [get]
func (h *AcademicHandler) GetSubjectClass(c *gin.Context) {
	id, ok := positivePathID(c, "id", "subject assignment")
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.GetSubjectClass(c.Request.Context(), actor, schoolID, id)
	writeAcademicResult(c, row, err)
}

// CreateSubjectClass godoc
// @Summary Assign a subject and teacher to a class
// @Tags academic
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param payload body service.CreateSubjectClassInput true "Assignment details"
// @Success 201 {object} model.SubjectClass
// @Router /subject-classes [post]
func (h *AcademicHandler) CreateSubjectClass(c *gin.Context) {
	var input service.CreateSubjectClassInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid subject assignment request"})
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.CreateSubjectClass(c.Request.Context(), actor, schoolID, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, row)
}

// UpdateSubjectClass godoc
// @Summary Update a class subject assignment
// @Tags academic
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Assignment ID"
// @Param payload body service.UpdateSubjectClassInput true "Fields to update"
// @Success 200 {object} model.SubjectClass
// @Router /subject-classes/{id} [patch]
func (h *AcademicHandler) UpdateSubjectClass(c *gin.Context) {
	id, ok := positivePathID(c, "id", "subject assignment")
	if !ok {
		return
	}
	var input service.UpdateSubjectClassInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid subject assignment update request"})
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.UpdateSubjectClass(c.Request.Context(), actor, schoolID, id, input, requestMetadata(c))
	writeAcademicResult(c, row, err)
}

// DeleteSubjectClass godoc
// @Summary Delete an unused class subject assignment
// @Tags academic
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Assignment ID"
// @Success 204
// @Router /subject-classes/{id} [delete]
func (h *AcademicHandler) DeleteSubjectClass(c *gin.Context) {
	id, ok := positivePathID(c, "id", "subject assignment")
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	if err := h.service.DeleteSubjectClass(c.Request.Context(), actor, schoolID, id, requestMetadata(c)); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListExamRegistrations godoc
// @Summary List scheduled exams
// @Tags academic
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param academic_year_id query int false "Academic year ID"
// @Param exam_id query int false "Exam type ID"
// @Success 200 {array} model.ExamRegistration
// @Router /exam-registrations [get]
func (h *AcademicHandler) ListExamRegistrations(c *gin.Context) {
	options, ok := academicListOptions(c)
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	rows, err := h.service.ListExamRegistrations(c.Request.Context(), actor, schoolID, options)
	writeAcademicResult(c, rows, err)
}

// GetExamRegistration godoc
// @Summary Get a scheduled exam
// @Tags academic
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Exam registration ID"
// @Success 200 {object} model.ExamRegistration
// @Router /exam-registrations/{id} [get]
func (h *AcademicHandler) GetExamRegistration(c *gin.Context) {
	id, ok := positivePathID(c, "id", "exam registration")
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.GetExamRegistration(c.Request.Context(), actor, schoolID, id)
	writeAcademicResult(c, row, err)
}

// CreateExamRegistration godoc
// @Summary Schedule an exam inside an academic year
// @Tags academic
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param payload body service.CreateExamRegistrationInput true "Exam schedule"
// @Success 201 {object} model.ExamRegistration
// @Router /exam-registrations [post]
func (h *AcademicHandler) CreateExamRegistration(c *gin.Context) {
	var input service.CreateExamRegistrationInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid exam registration request"})
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.CreateExamRegistration(c.Request.Context(), actor, schoolID, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, row)
}

// UpdateExamRegistration godoc
// @Summary Update a scheduled exam
// @Tags academic
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Exam registration ID"
// @Param payload body service.UpdateExamRegistrationInput true "Fields to update"
// @Success 200 {object} model.ExamRegistration
// @Router /exam-registrations/{id} [patch]
func (h *AcademicHandler) UpdateExamRegistration(c *gin.Context) {
	id, ok := positivePathID(c, "id", "exam registration")
	if !ok {
		return
	}
	var input service.UpdateExamRegistrationInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid exam registration update request"})
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.UpdateExamRegistration(c.Request.Context(), actor, schoolID, id, input, requestMetadata(c))
	writeAcademicResult(c, row, err)
}

// DeleteExamRegistration godoc
// @Summary Delete a scheduled exam with no results
// @Tags academic
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Exam registration ID"
// @Success 204
// @Router /exam-registrations/{id} [delete]
func (h *AcademicHandler) DeleteExamRegistration(c *gin.Context) {
	id, ok := positivePathID(c, "id", "exam registration")
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	if err := h.service.DeleteExamRegistration(c.Request.Context(), actor, schoolID, id, requestMetadata(c)); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// ListExamResults godoc
// @Summary List exam results
// @Tags academic
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param exam_registration_id query int false "Exam registration ID"
// @Param student_class_id query int false "Enrollment ID"
// @Param subject_class_id query int false "Subject assignment ID"
// @Param student_id query int false "Student ID"
// @Success 200 {array} model.ExamResult
// @Router /exam-results [get]
func (h *AcademicHandler) ListExamResults(c *gin.Context) {
	options, ok := academicListOptions(c)
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	rows, err := h.service.ListExamResults(c.Request.Context(), actor, schoolID, options)
	writeAcademicResult(c, rows, err)
}

// GetExamResult godoc
// @Summary Get an exam result
// @Tags academic
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Exam result ID"
// @Success 200 {object} model.ExamResult
// @Router /exam-results/{id} [get]
func (h *AcademicHandler) GetExamResult(c *gin.Context) {
	id, ok := positivePathID(c, "id", "exam result")
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.GetExamResult(c.Request.Context(), actor, schoolID, id)
	writeAcademicResult(c, row, err)
}

// CreateExamResult godoc
// @Summary Record a student's subject mark
// @Tags academic
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param payload body service.CreateExamResultInput true "Result details"
// @Success 201 {object} model.ExamResult
// @Router /exam-results [post]
func (h *AcademicHandler) CreateExamResult(c *gin.Context) {
	var input service.CreateExamResultInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid exam result request"})
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.CreateExamResult(c.Request.Context(), actor, schoolID, input, requestMetadata(c))
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusCreated, row)
}

// UpdateExamResult godoc
// @Summary Correct an exam mark
// @Tags academic
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Exam result ID"
// @Param payload body service.UpdateExamResultInput true "Corrected mark"
// @Success 200 {object} model.ExamResult
// @Router /exam-results/{id} [patch]
func (h *AcademicHandler) UpdateExamResult(c *gin.Context) {
	id, ok := positivePathID(c, "id", "exam result")
	if !ok {
		return
	}
	var input service.UpdateExamResultInput
	if c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid exam result update request"})
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	row, err := h.service.UpdateExamResult(c.Request.Context(), actor, schoolID, id, input, requestMetadata(c))
	writeAcademicResult(c, row, err)
}

// DeleteExamResult godoc
// @Summary Delete an exam result
// @Tags academic
// @Security BearerAuth
// @Param X-School-ID header int false "Required for SuperAdmin"
// @Param id path int true "Exam result ID"
// @Success 204
// @Router /exam-results/{id} [delete]
func (h *AcademicHandler) DeleteExamResult(c *gin.Context) {
	id, ok := positivePathID(c, "id", "exam result")
	if !ok {
		return
	}
	actor, _ := middleware.Principal(c)
	schoolID, _ := middleware.SchoolID(c)
	if err := h.service.DeleteExamResult(c.Request.Context(), actor, schoolID, id, requestMetadata(c)); err != nil {
		writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func academicListOptions(c *gin.Context) (repository.AcademicListOptions, bool) {
	options := repository.AcademicListOptions{
		Search: c.Query("search"), Limit: boundedInt(c.Query("limit"), 50, 1, 100),
		Offset: boundedInt(c.Query("offset"), 0, 0, 1_000_000),
	}
	fields := []struct {
		name   string
		target **uint64
	}{
		{"academic_year_id", &options.AcademicYearID}, {"class_id", &options.ClassID},
		{"student_id", &options.StudentID}, {"subject_id", &options.SubjectID}, {"staff_id", &options.StaffID},
		{"exam_id", &options.ExamID}, {"exam_registration_id", &options.ExamRegistrationID},
		{"student_class_id", &options.StudentClassID}, {"subject_class_id", &options.SubjectClassID},
	}
	for _, field := range fields {
		value, err := optionalUint64(c.Query(field.name))
		if err != nil {
			c.JSON(http.StatusBadRequest, errorResponse{Error: field.name + " must be a positive integer"})
			return repository.AcademicListOptions{}, false
		}
		*field.target = value
	}
	return options, true
}

func writeAcademicResult(c *gin.Context, value any, err error) {
	if err != nil {
		writeServiceError(c, err)
		return
	}
	c.JSON(http.StatusOK, value)
}
