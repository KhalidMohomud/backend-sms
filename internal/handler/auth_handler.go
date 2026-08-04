package handler

import (
	"backendapi/internal/middleware"
	"backendapi/internal/service"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthHandler struct{ service *service.AuthService }

func NewAuthHandler(service *service.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

// Login godoc
// @Summary Log in with a username and password
// @Tags auth
// @Accept json
// @Produce json
// @Param payload body service.LoginInput true "Login details"
// @Success 200 {object} service.AuthResult
// @Failure 400 {object} errorResponse
// @Failure 401 {object} errorResponse
// @Router /auth/login [post]
func (h *AuthHandler) Login(c *gin.Context) {
	var input service.LoginInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid login request"})
		return
	}
	result, err := h.service.Login(c.Request.Context(), input, requestMetadata(c))
	if err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) || errors.Is(err, service.ErrAccountUnavailable) {
			c.JSON(http.StatusUnauthorized, errorResponse{Error: "invalid username or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "could not log in"})
		return
	}
	c.JSON(http.StatusOK, result)
}

// Me godoc
// @Summary Get the current authorization identity
// @Tags auth
// @Produce json
// @Security BearerAuth
// @Success 200 {object} authz.Principal
// @Failure 401 {object} errorResponse
// @Router /auth/me [get]
func (h *AuthHandler) Me(c *gin.Context) {
	principal, ok := middleware.Principal(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, errorResponse{Error: "authentication required"})
		return
	}
	c.JSON(http.StatusOK, principal)
}

func requestMetadata(c *gin.Context) service.RequestMetadata {
	return service.RequestMetadata{IPAddress: c.ClientIP(), UserAgent: c.Request.UserAgent()}
}
