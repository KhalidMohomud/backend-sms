package middleware

import (
	"backendapi/internal/authz"
	"backendapi/internal/model"
	"backendapi/internal/repository"
	"backendapi/internal/security"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const PrincipalKey = "authenticated_principal"

func Authenticate(jwt *security.JWTManager, users repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
			return
		}

		tokenPrincipal, err := jwt.Parse(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		user, err := users.FindByID(c.Request.Context(), tokenPrincipal.UserID)
		if errors.Is(err, repository.ErrNotFound) || (err == nil && (user.Status != model.UserStatusActive || (user.SchoolID != nil && (user.School == nil || user.School.Status != model.SchoolStatusActive)))) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "account is unavailable"})
			return
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not validate account"})
			return
		}

		// Database state is authoritative, so permission changes apply immediately.
		c.Set(PrincipalKey, authz.FromUser(user))
		c.Next()
	}
}

func Principal(c *gin.Context) (authz.Principal, bool) {
	value, exists := c.Get(PrincipalKey)
	if !exists {
		return authz.Principal{}, false
	}
	principal, ok := value.(authz.Principal)
	return principal, ok
}
