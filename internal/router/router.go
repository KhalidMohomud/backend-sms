package router

import (
	"backendapi/internal/config"
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
	structure *handler.StructureHandler,
	people *handler.PeopleHandler,
	health *handler.HealthHandler,
	jwt *security.JWTManager,
	users repository.UserRepository,
	db *gorm.DB,
	sessions security.SessionRepository,
	appConfig config.AppConfig,
) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery(), middleware.LimitRequestBody(appConfig.MaxBodyBytes), middleware.CORS(appConfig.AllowedOrigins))
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
			lookups := secured.Group("/lookups")
			lookups.GET("/:type", structure.ListLookups)
			lookups.GET("/:type/:id", structure.GetLookup)
			lookups.POST("/:type", middleware.RequirePermission(model.PermissionManageLookups), structure.CreateLookup)
			lookups.PATCH("/:type/:id", middleware.RequirePermission(model.PermissionManageLookups), structure.UpdateLookup)
			lookups.DELETE("/:type/:id", middleware.RequirePermission(model.PermissionManageLookups), structure.ArchiveLookup)

			levels := secured.Group("/levels", middleware.RequireSchool())
			levels.GET("", structure.ListLevels)
			levels.GET("/:id", structure.GetLevel)
			levels.POST("", middleware.RequirePermission(model.PermissionManageStructure), structure.CreateLevel)
			levels.PATCH("/:id", middleware.RequirePermission(model.PermissionManageStructure), structure.UpdateLevel)
			levels.DELETE("/:id", middleware.RequirePermission(model.PermissionManageStructure), structure.ArchiveLevel)

			classes := secured.Group("/classes", middleware.RequireSchool())
			classes.GET("", structure.ListClasses)
			classes.GET("/:id", structure.GetClass)
			classes.POST("", middleware.RequirePermission(model.PermissionManageStructure), structure.CreateClass)
			classes.PATCH("/:id", middleware.RequirePermission(model.PermissionManageStructure), structure.UpdateClass)
			classes.DELETE("/:id", middleware.RequirePermission(model.PermissionManageStructure), structure.ArchiveClass)

			addresses := secured.Group("/addresses", middleware.RequireSchool())
			addresses.GET("", people.ListAddresses)
			addresses.GET("/:id", people.GetAddress)
			addresses.POST("", people.CreateAddress)
			addresses.PATCH("/:id", people.UpdateAddress)
			addresses.DELETE("/:id", people.DeleteAddress)

			guardians := secured.Group("/guardians", middleware.RequireSchool(), middleware.RequirePermission(model.PermissionManageStudents))
			guardians.GET("", people.ListGuardians)
			guardians.GET("/:id", people.GetGuardian)
			guardians.POST("", people.CreateGuardian)
			guardians.PATCH("/:id", people.UpdateGuardian)
			guardians.DELETE("/:id", people.DeleteGuardian)

			students := secured.Group("/students", middleware.RequireSchool(), middleware.RequirePermission(model.PermissionManageStudents))
			students.GET("", people.ListStudents)
			students.GET("/:id", people.GetStudent)
			students.POST("", people.CreateStudent)
			students.PATCH("/:id", people.UpdateStudent)
			students.DELETE("/:id", people.ArchiveStudent)

			staff := secured.Group("/staff", middleware.RequireSchool(), middleware.RequirePermission(model.PermissionManageStaff))
			staff.GET("", people.ListStaff)
			staff.GET("/:id", people.GetStaff)
			staff.POST("", people.CreateStaff)
			staff.PATCH("/:id", people.UpdateStaff)
			staff.DELETE("/:id", people.ArchiveStaff)
			staff.GET("/:id/statuses", people.ListStaffStatuses)
			staff.POST("/:id/statuses", people.CreateStaffStatus)

			schools := secured.Group("/schools", middleware.RequirePermission(model.PermissionManageSchools))
			schools.GET("", foundation.ListSchools)
			schools.GET("/:id", foundation.GetSchool)
			schools.POST("", foundation.CreateSchool)
			schools.PATCH("/:id", foundation.UpdateSchool)
			schools.DELETE("/:id", foundation.ArchiveSchool)

			years := secured.Group("/academic-years", middleware.RequireSchool())
			years.GET("", foundation.ListAcademicYears)
			years.GET("/:id", foundation.GetAcademicYear)
			years.POST("", middleware.RequirePermission(model.PermissionManageAcademicYears), foundation.CreateAcademicYear)
			years.PATCH("/:id", middleware.RequirePermission(model.PermissionManageAcademicYears), foundation.UpdateAcademicYear)
			years.DELETE("/:id", middleware.RequirePermission(model.PermissionManageAcademicYears), foundation.DeleteAcademicYear)

			usersRoutes := secured.Group("/users", middleware.RequirePermission(model.PermissionManageUsers))
			usersRoutes.GET("", foundation.ListUsers)
			usersRoutes.GET("/:id", foundation.GetUser)
			usersRoutes.POST("", foundation.CreateUser)
			usersRoutes.PATCH("/:id", foundation.UpdateUser)
			usersRoutes.PATCH("/:id/status", foundation.UpdateUserStatus)
			usersRoutes.DELETE("/:id", foundation.DisableUser)
			usersRoutes.POST("/:id/password-reset-token", foundation.CreatePasswordResetToken)

			roles := secured.Group("/roles")
			roles.GET("", foundation.ListRoles)
			roles.GET("/:id", foundation.GetRole)
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
