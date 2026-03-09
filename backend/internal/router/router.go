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
		// Public note endpoints (no JWT)
		public := v1.Group("/public")
		{
			public.GET("/notes", handler.ListPublicNotes)                    // GET /api/v1/public/notes
			public.GET("/notes/:username/:slug", handler.GetPublicNote)        // GET /api/v1/public/notes/:username/:slug
		}
		notes := v1.Group("/notes")
		{
			notes.POST("", middleware.JWTMiddleware(), handler.CreateNote)           // POST /api/v1/notes
			notes.GET("", middleware.JWTMiddleware(), handler.ListNotes)             // GET /api/v1/notes
			notes.GET("/:id", middleware.JWTMiddleware(), handler.GetNote)           // GET /api/v1/notes/{id}
			notes.PATCH("/:id", middleware.JWTMiddleware(), handler.UpdateNote)     // PATCH /api/v1/notes/{id}
			notes.DELETE("/:id", middleware.JWTMiddleware(), handler.DeleteNote)     // DELETE /api/v1/notes/{id}
		}
		folders := v1.Group("/folders")
		{
			folders.POST("", middleware.JWTMiddleware(), handler.CreateFolder)      // POST /api/v1/folders
			folders.PATCH("/:id", middleware.JWTMiddleware(), handler.UpdateFolder) // PATCH /api/v1/folders/{id}
			folders.DELETE("/:id", middleware.JWTMiddleware(), handler.DeleteFolder) // DELETE /api/v1/folders/{id}
		}
		tree := v1.Group("/tree")
		{
			tree.POST("/reorder", middleware.JWTMiddleware(), handler.ReorderTree) // POST /api/v1/tree/reorder
		}
	}

	r.GET("/api/user", middleware.JWTMiddleware(), handler.GetCurrentUser)

	return r
}
