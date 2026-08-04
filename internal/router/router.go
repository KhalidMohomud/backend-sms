package router

import (
	"backendapi/internal/handler"
	"backendapi/internal/middleware"
	"backendapi/internal/model"
	"backendapi/internal/repository"
	"backendapi/internal/security"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func New(
	auth *handler.AuthHandler,
	foundation *handler.FoundationHandler,
	health *handler.HealthHandler,
	jwt *security.JWTManager,
	users repository.UserRepository,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())
	// Do not trust spoofable forwarding headers until deployment config provides
	// an explicit reverse-proxy allowlist.
	_ = r.SetTrustedProxies(nil)

	r.GET("/health", health.Check)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", health.Check)
		authRoutes := v1.Group("/auth")
		authRoutes.POST("/login", auth.Login)
		authRoutes.GET("/me", middleware.Authenticate(jwt, users), auth.Me)

		secured := v1.Group("")
		secured.Use(middleware.Authenticate(jwt, users))
		{
			schools := secured.Group("/schools", middleware.RequirePermission(model.PermissionManageSchools))
			schools.GET("", foundation.ListSchools)
			schools.POST("", foundation.CreateSchool)

			years := secured.Group("/academic-years", middleware.RequireSchool())
			years.GET("", foundation.ListAcademicYears)
			years.POST("", middleware.RequirePermission(model.PermissionManageAcademicYears), foundation.CreateAcademicYear)

			usersRoutes := secured.Group("/users", middleware.RequirePermission(model.PermissionManageUsers))
			usersRoutes.GET("", foundation.ListUsers)
			usersRoutes.POST("", foundation.CreateUser)
			usersRoutes.PATCH("/:id/status", foundation.UpdateUserStatus)

			secured.GET("/roles", foundation.ListRoles)
			secured.GET("/permissions", middleware.RequirePermission(model.PermissionManageRoles), foundation.ListPermissions)
			secured.GET("/audit-logs", middleware.RequirePermission(model.PermissionViewAuditLogs), foundation.ListAuditLogs)
		}
	}
	return r
}
