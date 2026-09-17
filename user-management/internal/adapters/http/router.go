package http

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/adapters/http/handler"
	"github.com/thitiphum-dev/7-solutions-backend-challenge/user-management/internal/adapters/http/middleware"
)

func NewRouter(
	authHandler *handler.AuthHandler,
	userHandler *handler.UserHandler,
	authMiddleware *middleware.Auth,
) *gin.Engine {
	router := gin.New()

	router.Use(middleware.Logging(), gin.Recovery())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ok",
		})
	})

	authRoutes := router.Group("/auth")
	{
		authRoutes.POST("/register", authHandler.Register)
		authRoutes.POST("/login", authHandler.Login)
	}

	userRoutes := router.Group("/users")
	userRoutes.Use(authMiddleware.RequireAuth())
	{
		userRoutes.GET("", userHandler.List)
		userRoutes.GET("/:id", userHandler.GetByID)
		userRoutes.PATCH("/:id", userHandler.UpdateByID)
		userRoutes.DELETE("/:id", userHandler.DeleteByID)
	}

	return router
}
