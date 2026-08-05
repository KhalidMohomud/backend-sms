package handler

import (
	"backendapi/internal/middleware"
	"backendapi/internal/security"
	"backendapi/internal/service"
	"errors"
	"net/http"
	"strings"

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
		if errors.Is(err, security.ErrRateLimited) {
			c.JSON(http.StatusTooManyRequests, errorResponse{Error: err.Error()})
			return
		}
		if errors.Is(err, service.ErrInvalidCredentials) || errors.Is(err, service.ErrAccountUnavailable) {
			c.JSON(http.StatusUnauthorized, errorResponse{Error: "invalid username or password"})
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "could not log in"})
		return
	}
	c.JSON(http.StatusOK, result)
}

// Refresh godoc
// @Summary Rotate a refresh token and issue a new token pair
// @Tags auth
// @Accept json
// @Produce json
// @Param payload body service.RefreshInput true "Refresh token"
// @Success 200 {object} service.AuthResult
// @Router /auth/refresh [post]
func (h *AuthHandler) Refresh(c *gin.Context) {
	var input service.RefreshInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid refresh request"})
		return
	}
	result, err := h.service.Refresh(c.Request.Context(), input, requestMetadata(c))
	if err != nil {
		if errors.Is(err, security.ErrInvalidToken) || errors.Is(err, service.ErrAccountUnavailable) {
			c.JSON(http.StatusUnauthorized, errorResponse{Error: "invalid or expired refresh token"})
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "could not refresh session"})
		return
	}
	c.JSON(http.StatusOK, result)
}

// Logout godoc
// @Summary Revoke the current access token and optional refresh token
// @Tags auth
// @Accept json
// @Security BearerAuth
// @Param payload body service.LogoutInput false "Refresh token to revoke"
// @Success 204
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	var input service.LogoutInput
	if c.Request.ContentLength > 0 && c.ShouldBindJSON(&input) != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid logout request"})
		return
	}
	if err := h.service.Logout(c.Request.Context(), bearerToken(c), input, requestMetadata(c)); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "could not log out"})
		return
	}
	c.Status(http.StatusNoContent)
}

// LogoutAll godoc
// @Summary Revoke all sessions for the current user
// @Tags auth
// @Security BearerAuth
// @Success 204
// @Router /auth/logout-all [post]
func (h *AuthHandler) LogoutAll(c *gin.Context) {
	principal, _ := middleware.Principal(c)
	if err := h.service.LogoutAll(c.Request.Context(), principal, requestMetadata(c)); err != nil {
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "could not revoke sessions"})
		return
	}
	c.Status(http.StatusNoContent)
}

// ChangePassword godoc
// @Summary Change the current user's password
// @Tags auth
// @Accept json
// @Security BearerAuth
// @Param payload body service.ChangePasswordInput true "Current and new password"
// @Success 204
// @Router /auth/change-password [post]
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var input service.ChangePasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid password change request"})
		return
	}
	principal, _ := middleware.Principal(c)
	if err := h.service.ChangePassword(c.Request.Context(), principal, input, requestMetadata(c)); err != nil {
		if errors.Is(err, service.ErrInvalidCredentials) {
			c.JSON(http.StatusUnauthorized, errorResponse{Error: "current password is incorrect"})
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "could not change password"})
		return
	}
	c.Status(http.StatusNoContent)
}

// ResetPassword godoc
// @Summary Use a one-time reset token to set a new password
// @Tags auth
// @Accept json
// @Param payload body service.ResetPasswordInput true "Reset token and new password"
// @Success 204
// @Router /auth/reset-password [post]
func (h *AuthHandler) ResetPassword(c *gin.Context) {
	var input service.ResetPasswordInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid password reset request"})
		return
	}
	if err := h.service.ResetPassword(c.Request.Context(), input, requestMetadata(c)); err != nil {
		if errors.Is(err, security.ErrInvalidToken) {
			c.JSON(http.StatusBadRequest, errorResponse{Error: "invalid or expired reset token"})
			return
		}
		c.JSON(http.StatusInternalServerError, errorResponse{Error: "could not reset password"})
		return
	}
	c.Status(http.StatusNoContent)
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

func bearerToken(c *gin.Context) string {
	parts := strings.Fields(c.GetHeader("Authorization"))
	if len(parts) == 2 {
		return parts[1]
	}
	return ""
}
