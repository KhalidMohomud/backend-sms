package middleware

import (
	"backendapi/internal/authz"
	"backendapi/internal/database"
	"backendapi/internal/model"
	"backendapi/internal/repository"
	"backendapi/internal/security"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const PrincipalKey = "authenticated_principal"

func Authenticate(jwt *security.JWTManager, users repository.UserRepository, db *gorm.DB, sessions security.SessionRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		parts := strings.Fields(c.GetHeader("Authorization"))
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "missing or invalid authorization header"})
			return
		}

		identity, err := jwt.ParseIdentity(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}
		denied, err := sessions.AccessDenied(c.Request.Context(), identity.Principal.UserID, identity.JTI, identity.IssuedAt)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not validate session"})
			return
		}
		if denied {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "session has been revoked"})
			return
		}
		tokenPrincipal := identity.Principal
		requestContext, tx, err := database.BeginRequest(c.Request.Context(), db, database.SecurityScope{Principal: tokenPrincipal})
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not establish database security context"})
			return
		}
		c.Request = c.Request.WithContext(requestContext)
		user, err := users.FindByID(requestContext, tokenPrincipal.UserID)
		if errors.Is(err, repository.ErrNotFound) || (err == nil && (user.Status != model.UserStatusActive || user.Role.Status == model.RoleStatusInactive || (user.SchoolID != nil && (user.School == nil || user.School.Status != model.SchoolStatusActive)))) {
			tx.Rollback()
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "account is unavailable"})
			return
		}
		if err != nil {
			tx.Rollback()
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not validate account"})
			return
		}

		// Database state is authoritative, so permission changes apply immediately.
		principal := authz.FromUser(user)
		if err := database.SetSecurityScope(tx, database.SecurityScope{Principal: principal}); err != nil {
			tx.Rollback()
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not establish database security context"})
			return
		}
		c.Set(PrincipalKey, principal)
		c.Next()
		finishRequestTransaction(c, tx)
	}
}

func AuthenticationDatabase(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestContext, tx, err := database.BeginRequest(c.Request.Context(), db, database.SecurityScope{AuthLookup: true})
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not establish database security context"})
			return
		}
		c.Request = c.Request.WithContext(requestContext)
		c.Next()
		finishRequestTransaction(c, tx)
	}
}

func finishRequestTransaction(c *gin.Context, tx *gorm.DB) {
	if c.Writer.Status() >= http.StatusInternalServerError {
		tx.Rollback()
		return
	}
	if err := tx.Commit().Error; err != nil && !c.Writer.Written() {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": "could not commit request"})
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
