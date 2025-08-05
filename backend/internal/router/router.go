package router

import (
	"backend/internal/handler"
	"backend/internal/middleware"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func Setup() *gin.Engine {
	r := gin.Default()

	// Configure CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"http://localhost:5173"}, // Vite and other common dev ports
		AllowMethods:     []string{"GET", "POST", "PUT", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	auth := r.Group("/api/auth")
	{
		auth.POST("/register", handler.Register)
		// TODO: add login, refresh
		auth.POST("/login", handler.Login)
		auth.POST("/refresh", handler.Refresh)
	}

	r.GET("/api/user", middleware.JWTMiddleware(), handler.GetCurrentUser)

	return r
}
