package router

import (
	"backend/internal/database"
	"backend/internal/handler"
	"backend/internal/middleware"
	"backend/internal/noteapp"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func Setup() *gin.Engine {
	return SetupWithDB(database.DB)
}

func SetupWithDB(db *gorm.DB) *gin.Engine {
	origins := os.Getenv("CORS_ORIGINS")
	if origins == "" {
		origins = "http://localhost:5173"
	}
	return SetupWithOptions(db, Options{
		JWTSecret:      os.Getenv("JWT_SECRET"),
		CORSOrigins:    splitList(origins),
		TrustedProxies: splitList(os.Getenv("TRUSTED_PROXIES")),
	})
}

type Options struct {
	JWTSecret      string
	CORSOrigins    []string
	TrustedProxies []string
}

func splitList(value string) []string {
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func SetupWithOptions(db *gorm.DB, options Options) *gin.Engine {
	r := gin.Default()
	content := handler.NewContentHandler(noteapp.NewService(noteapp.NewGormRepository(db)))
	authenticate := middleware.JWTMiddlewareWithSecret(options.JWTSecret)

	if len(options.TrustedProxies) > 0 {
		r.SetTrustedProxies(options.TrustedProxies)
	} else {
		r.SetTrustedProxies(nil)
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     options.CORSOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	auth := r.Group("/api/auth")
	{
		auth.POST("/register", handler.Register)
		auth.POST("/login", handler.Login)
		auth.POST("/refresh", handler.Refresh)
	}

	v1 := r.Group("/api/v1")
	{
		// Public note endpoints (no JWT)
		public := v1.Group("/public")
		{
			public.GET("/notes", content.ListPublicNotes)               // GET /api/v1/public/notes
			public.GET("/notes/:username/:slug", content.GetPublicNote) // GET /api/v1/public/notes/:username/:slug
		}
		notes := v1.Group("/notes")
		{
			notes.POST("", authenticate, content.CreateNote)       // POST /api/v1/notes
			notes.GET("", authenticate, content.ListNotes)         // GET /api/v1/notes
			notes.GET("/:id", authenticate, content.GetNote)       // GET /api/v1/notes/{id}
			notes.PATCH("/:id", authenticate, content.UpdateNote)  // PATCH /api/v1/notes/{id}
			notes.DELETE("/:id", authenticate, content.DeleteNote) // DELETE /api/v1/notes/{id}
		}
		folders := v1.Group("/folders")
		{
			folders.GET("", authenticate, content.ListFolders)         // GET /api/v1/folders
			folders.POST("", authenticate, content.CreateFolder)       // POST /api/v1/folders
			folders.PATCH("/:id", authenticate, content.UpdateFolder)  // PATCH /api/v1/folders/{id}
			folders.DELETE("/:id", authenticate, content.DeleteFolder) // DELETE /api/v1/folders/{id}
		}
		tree := v1.Group("/tree")
		{
			tree.POST("/reorder", authenticate, content.ReorderTree) // POST /api/v1/tree/reorder
		}
	}

	r.GET("/api/user", authenticate, handler.GetCurrentUser)

	return r
}
