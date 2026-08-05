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
	"gorm.io/gorm"
)

func New(
	auth *handler.AuthHandler,
	foundation *handler.FoundationHandler,
	health *handler.HealthHandler,
	jwt *security.JWTManager,
	users repository.UserRepository,
	db *gorm.DB,
	sessions security.SessionRepository,
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
		authRoutes.POST("/login", middleware.AuthenticationDatabase(db), auth.Login)
		authRoutes.POST("/refresh", middleware.AuthenticationDatabase(db), auth.Refresh)
		authRoutes.POST("/reset-password", middleware.AuthenticationDatabase(db), auth.ResetPassword)
		authRoutes.GET("/me", middleware.Authenticate(jwt, users, db, sessions), auth.Me)
		authRoutes.POST("/logout", middleware.Authenticate(jwt, users, db, sessions), auth.Logout)
		authRoutes.POST("/logout-all", middleware.Authenticate(jwt, users, db, sessions), auth.LogoutAll)
		authRoutes.POST("/change-password", middleware.Authenticate(jwt, users, db, sessions), auth.ChangePassword)

		secured := v1.Group("")
		secured.Use(middleware.Authenticate(jwt, users, db, sessions))
		{
			schools := secured.Group("/schools", middleware.RequirePermission(model.PermissionManageSchools))
			schools.GET("", foundation.ListSchools)
			schools.POST("", foundation.CreateSchool)
			schools.PATCH("/:id", foundation.UpdateSchool)
			schools.DELETE("/:id", foundation.ArchiveSchool)

			years := secured.Group("/academic-years", middleware.RequireSchool())
			years.GET("", foundation.ListAcademicYears)
			years.POST("", middleware.RequirePermission(model.PermissionManageAcademicYears), foundation.CreateAcademicYear)
			years.PATCH("/:id", middleware.RequirePermission(model.PermissionManageAcademicYears), foundation.UpdateAcademicYear)
			years.DELETE("/:id", middleware.RequirePermission(model.PermissionManageAcademicYears), foundation.DeleteAcademicYear)

			usersRoutes := secured.Group("/users", middleware.RequirePermission(model.PermissionManageUsers))
			usersRoutes.GET("", foundation.ListUsers)
			usersRoutes.POST("", foundation.CreateUser)
			usersRoutes.PATCH("/:id", foundation.UpdateUser)
			usersRoutes.PATCH("/:id/status", foundation.UpdateUserStatus)
			usersRoutes.DELETE("/:id", foundation.DisableUser)
			usersRoutes.POST("/:id/password-reset-token", foundation.CreatePasswordResetToken)

			roles := secured.Group("/roles")
			roles.GET("", foundation.ListRoles)
			roles.POST("", middleware.RequirePermission(model.PermissionManageRoles), foundation.CreateRole)
			roles.PATCH("/:id", middleware.RequirePermission(model.PermissionManageRoles), foundation.UpdateRole)
			roles.DELETE("/:id", middleware.RequirePermission(model.PermissionManageRoles), foundation.ArchiveRole)
			roles.PUT("/:id/permissions", middleware.RequirePermission(model.PermissionManageRoles), foundation.ReplaceRolePermissions)
			secured.GET("/permissions", middleware.RequirePermission(model.PermissionManageRoles), foundation.ListPermissions)
			secured.GET("/audit-logs", middleware.RequirePermission(model.PermissionViewAuditLogs), foundation.ListAuditLogs)
		}
	}
	return r
}
