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

	v1 := r.Group("/api/v1")
	{
		notes := v1.Group("/notes")
		{
			notes.POST("", handler.CreateNote)           // POST /api/v1/notes
			notes.GET("", handler.ListNotes)             // GET /api/v1/notes (列表)
			notes.GET("/:id", handler.GetNote)           // GET /api/v1/notes/{id}
			notes.PATCH("/:id", handler.UpdateNote)      // PATCH /api/v1/notes/{id}
			notes.DELETE("/:id", handler.DeleteNote)     // DELETE /api/v1/notes/{id}
		}
		// folders := v1.Group("/folders")
		// {

		// }
		// tree := v1.Group("/tree")
		// {
		// }
	}

	r.GET("/api/user", middleware.JWTMiddleware(), handler.GetCurrentUser)

	return r
}
