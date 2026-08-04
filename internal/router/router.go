package router

import (
	"backendapi/internal/handler"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func New(auth *handler.AuthHandler, health *handler.HealthHandler) *gin.Engine {
	r := gin.New()
	r.Use(gin.Logger(), gin.Recovery())

	r.GET("/health", health.Check)
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	v1 := r.Group("/api/v1")
	{
		v1.GET("/health", health.Check)
		authRoutes := v1.Group("/auth")
		authRoutes.POST("/register", auth.Register)
		authRoutes.POST("/login", auth.Login)
	}
	return r
}
