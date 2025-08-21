// internal/handler/note.go
package handler

import (
	"backend/internal/database"
	"backend/internal/model"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

// ------------------------------------------------------------
// Get Note
// ------------------------------------------------------------

func GetNote(c *gin.Context) {
	// Get parameters
	folderID := c.Query("folder_id")
	searchQuery := c.Query("q")
	status := c.Query("status")
	limit := c.Query("limit")
	offset := c.Query("offset")

	// Get user ID from JWT token
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "UNAUTHORIZED",
			"message": "invalid token",
		})
		return
	}
	// Build query
	query := database.DB.Where("user_id = ?", userID)

	// Apply filters
	if folderID != "" {
		query = query.Where("folder_id = ?", folderID)
	}
	if searchQuery != "" {
		query = query.Where("title LIKE ? OR content_md LIKE ?", "%"+searchQuery+"%", "%"+searchQuery+"%")
	}
	
	if status != "" {
		query = query.Where("is_published = ?", status == "published")
	}

	// Apply pagination
	query = query.Order("created_at DESC").Limit(limit).Offset(offset)

	// Get notes
	var notes []model.Note
	if err := query.Find(&notes).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "INTERNAL",
			"message": "Failed to get notes",
		})
		return
	}

	// Return the notes
	c.JSON(http.StatusOK, gin.H{
		"items": notes,
		"total": len(notes),
		"limit": limit,
		"offset": offset,
	})

}


// ------------------------------------------------------------
// Create Note
// ------------------------------------------------------------

type createNoteRequest struct {
	Title     string `json:"title" binding:"required"`
	FolderID  *uint  `json:"folder_id"`
	ContentMd string `json:"content_md" binding:"required"`
	Slug      string `json:"slug"`
}

func CreateNote(c *gin.Context) {
	var req createNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "VALIDATION_ERROR",
			"message": err.Error(),
		})
		return
	}

	// Get user ID from JWT token
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "UNAUTHORIZED",
			"message": "invalid token",
		})
		return
	}

	// Set default values
	if req.Title == "" {
		req.Title = "Untitled"
	}
	if req.ContentMd == "" {
		req.ContentMd = "# New note"
	}

	// Generate slug if not provided
	if req.Slug == "" {
		req.Slug = generateSlug(req.Title)
	}

	// Validate slug format
	if err := validateSlug(req.Slug); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "VALIDATION_ERROR",
			"message": err.Error(),
		})
		return
	}

	// Check if slug already exists for this user
	var existingNote model.Note
	if err := database.DB.Where("user_id = ? AND slug = ?", userID, req.Slug).First(&existingNote).Error; err == nil {
		// Slug already exists
		c.JSON(http.StatusConflict, gin.H{
			"error":   "VALIDATION_ERROR",
			"message": "slug already exists",
		})
		return
	}

	// Validate that the folder exists and belongs to the current user
	var folder model.Folder
	if req.FolderID != nil {
		if err := database.DB.Where("id = ? AND user_id = ?", *req.FolderID, userID).First(&folder).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "VALIDATION_ERROR",
				"message": "folder not found or access denied",
			})
			return
		}
	}

	// Convert markdown to HTML (basic implementation)
	contentHtml := convertMarkdownToHTML(req.ContentMd)

	// Create note
	note := model.Note{
		UserID:      userID.(uint),
		FolderID:    req.FolderID,
		Title:       req.Title,
		Slug:        req.Slug,
		ContentMd:   req.ContentMd,
		ContentHtml: contentHtml,
		IsPublished: false,
		Visibility:  "private",
		SortOrder:   0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := database.DB.Create(&note).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "INTERNAL",
			"message": "Failed to create note",
		})
		return
	}

	// Return the created note
	response := gin.H{
		"id":           note.ID,
		"user_id":      note.UserID,
		"title":        note.Title,
		"slug":         note.Slug,
		"content_md":   note.ContentMd,
		"content_html": note.ContentHtml,
		"is_published": note.IsPublished,
		"visibility":   note.Visibility,
		"sort_order":   note.SortOrder,
		"created_at":   note.CreatedAt.Format(time.RFC3339),
		"updated_at":   note.UpdatedAt.Format(time.RFC3339),
	}

	// Handle nullable folder_id
	if note.FolderID != nil {
		response["folder_id"] = *note.FolderID
	} else {
		response["folder_id"] = nil
	}

	c.JSON(http.StatusCreated, response)
}

// Helper functions

func generateSlug(title string) string {
	// Convert to lowercase
	slug := strings.ToLower(title)
	
	// Replace spaces with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	
	// Remove special characters, keep only letters, numbers, and hyphens
	reg := regexp.MustCompile("[^a-z0-9-]")
	slug = reg.ReplaceAllString(slug, "")
	
	// Replace multiple hyphens with single hyphen
	reg = regexp.MustCompile("-+")
	slug = reg.ReplaceAllString(slug, "-")
	
	// Remove leading and trailing hyphens
	slug = strings.Trim(slug, "-")
	
	// If slug is empty, use default
	if slug == "" {
		slug = "untitled"
	}
	
	return slug
}

func validateSlug(slug string) error {
	if len(slug) < 3 {
		return fmt.Errorf("slug too short (minimum 3 characters)")
	}
	if len(slug) > 100 {
		return fmt.Errorf("slug too long (maximum 100 characters)")
	}
	
	// Check if slug contains only valid characters
	reg := regexp.MustCompile("^[a-z0-9-]+$")
	if !reg.MatchString(slug) {
		return fmt.Errorf("slug contains invalid characters (only lowercase letters, numbers, and hyphens allowed)")
	}
	
	return nil
}

func convertMarkdownToHTML(markdown string) string {
	// Basic markdown to HTML conversion
	// In production, you should use a proper markdown parser like goldmark
	
	// Replace # with <h1>
	reg := regexp.MustCompile(`^# (.+)$`)
	html := reg.ReplaceAllString(markdown, "<h1>$1</h1>")
	
	// Replace ## with <h2>
	reg = regexp.MustCompile(`^## (.+)$`)
	html = reg.ReplaceAllString(html, "<h2>$1</h2>")
	
	// Replace ### with <h3>
	reg = regexp.MustCompile(`^### (.+)$`)
	html = reg.ReplaceAllString(html, "<h3>$1</h3>")
	
	// Replace **text** with <strong>text</strong>
	reg = regexp.MustCompile(`\*\*(.+?)\*\*`)
	html = reg.ReplaceAllString(html, "<strong>$1</strong>")
	
	// Replace *text* with <em>text</em>
	reg = regexp.MustCompile(`\*(.+?)\*`)
	html = reg.ReplaceAllString(html, "<em>$1</em>")
	
	// Replace line breaks with <br>
	html = strings.ReplaceAll(html, "\n", "<br>")
	
	return html
}

