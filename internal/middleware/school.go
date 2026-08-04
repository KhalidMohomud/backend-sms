package middleware

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

const SchoolIDKey = "school_id"

func RequireSchool() gin.HandlerFunc {
	return func(c *gin.Context) {
		principal, ok := Principal(c)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authentication required"})
			return
		}

		header := strings.TrimSpace(c.GetHeader("X-School-ID"))
		if principal.IsSuperAdmin() {
			if header == "" {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "X-School-ID is required for this operation"})
				return
			}
			schoolID, err := strconv.ParseUint(header, 10, 64)
			if err != nil || schoolID == 0 {
				c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "X-School-ID must be a positive integer"})
				return
			}
			c.Set(SchoolIDKey, schoolID)
			c.Next()
			return
		}

		if principal.SchoolID == nil {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "account has no school access"})
			return
		}
		if header != "" && header != strconv.FormatUint(*principal.SchoolID, 10) {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"error": "cannot access another school"})
			return
		}
		c.Set(SchoolIDKey, *principal.SchoolID)
		c.Next()
	}
}

func SchoolID(c *gin.Context) (uint64, bool) {
	value, exists := c.Get(SchoolIDKey)
	if !exists {
		return 0, false
	}
	id, ok := value.(uint64)
	return id, ok
}
