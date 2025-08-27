// internal/handler/note.go
package handler

import (
	"backend/internal/database"
	"backend/internal/model"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ------------------------------------------------------------
// List Notes
// GET /api/v1/notes
// Query params:
// - folder_id: filter by folder ID
// - q: search query
// - status: filter by status (published, draft)
// - limit: number of notes per page
// - offset: page number
// ------------------------------------------------------------

func ListNotes(c *gin.Context) {
	// Get parameters
	folderID := c.Query("folder_id")
	searchQuery := c.Query("q")
	status := c.Query("status")
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")

	// Convert limit and offset to integers
	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		limit = 50 // default value
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil {
		offset = 0 // default value
	}

	// Validate limit
	if limit > 200 {
		limit = 200
	}
	if limit < 1 {
		limit = 50
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
	// Build query
	query := database.DB.Where("user_id = ?", userID)

	// Apply filters
	if folderID != "" {
		query = query.Where("folder_id = ?", folderID)
	}
	if searchQuery != "" {
		query = query.Where("title LIKE ? OR content_md LIKE ?", "%"+searchQuery+"%", "%"+searchQuery+"%")
	}
	
	if status != "" && status != "all" {
		if status == "published" {
			query = query.Where("is_published = ?", true)
		} else if status == "draft" {
			query = query.Where("is_published = ?", false)
		}
	}

	// Get total count
	var total int64
	query.Model(&model.Note{}).Count(&total)

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
		"items":  notes,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}


// ------------------------------------------------------------
// Create Note
// POST /api/v1/notes
// Body:
// - title: string
// - folder_id: uint
// - content_md: string
// ------------------------------------------------------------

type createNoteRequest struct {
	Title     string `json:"title"`
	FolderID  *uint  `json:"folder_id"`
	ContentMd string `json:"content_md"`
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

	// Generate unique slug based on title
	slug := generateUniqueSlug(req.Title, userID.(uint), nil)

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
		Slug:        slug,
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

func generateUniqueSlug(title string, userID uint, excludeNoteID *uint) string {
	baseSlug := generateSlug(title)
	
	// Check if base slug is available
	var count int64
	query := database.DB.Where("user_id = ? AND slug = ?", userID, baseSlug)
	if excludeNoteID != nil {
		query = query.Where("id != ?", *excludeNoteID)
	}
	query.Model(&model.Note{}).Count(&count)
	
	if count == 0 {
		return baseSlug
	}
	
	// If there's a conflict, add numeric suffix
	for i := 2; i <= 999; i++ {
		candidateSlug := fmt.Sprintf("%s-%d", baseSlug, i)
		query := database.DB.Where("user_id = ? AND slug = ?", userID, candidateSlug)
		if excludeNoteID != nil {
			query = query.Where("id != ?", *excludeNoteID)
		}
		query.Model(&model.Note{}).Count(&count)
		
		if count == 0 {
			return candidateSlug
		}
	}
	
	// Fallback: use timestamp suffix if we somehow exhaust numeric options
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s-%d", baseSlug, timestamp)
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

// ------------------------------------------------------------
// Get Note by ID
// GET /api/v1/notes/{id}
// ------------------------------------------------------------

func GetNote(c *gin.Context) {
	// Get note ID from path
	idStr := c.Param("id")

	// Validate ID is a number
	idInt, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "VALIDATION_ERROR",
			"message": "invalid note ID",
		})
		return
	}
	noteID := uint(idInt)
	
	// Get user ID from JWT token
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "UNAUTHORIZED",
			"message": "invalid user ID",
		})
		return
	}

	// Search for note
	var note model.Note
	if err := database.DB.Where("id = ? AND user_id = ?", noteID, userID).First(&note).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "NOT_FOUND",
				"message": "note not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "INTERNAL",
				"message": "Failed to get note",
			})
		}
		return
	}

	// Return the note
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

	c.JSON(http.StatusOK, response)
}

// ------------------------------------------------------------
// Update Note
// PATCH /api/v1/notes/{id}
// Body:
// - title: string
// - content_md: string
// - is_published: bool
// - visibility: string
// ------------------------------------------------------------

type updateNoteRequest struct {
	Title       *string `json:"title"`
	FolderID    *uint   `json:"folder_id"`
	ContentMd   *string `json:"content_md"`
	IsPublished *bool   `json:"is_published"`
	Visibility  *string `json:"visibility"`
	UpdatedAt   string  `json:"updated_at"`
}

func UpdateNote(c *gin.Context) {
	// Get note ID from path
	idStr := c.Param("id")

	// Validate ID is a number
	idInt, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "VALIDATION_ERROR",
			"message": "invalid note ID",
		})
		return
	}
	noteID := uint(idInt)

	// Get user ID from JWT token
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "UNAUTHORIZED",
			"message": "invalid user ID",
		})
		return
	}

	// Parse request body
	var req updateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "VALIDATION_ERROR",
			"message": err.Error(),
		})
		return
	}

	// Get note
	var note model.Note
	if err := database.DB.Where("id = ? AND user_id = ?", noteID, userID).First(&note).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "NOT_FOUND",
				"message": "note not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "INTERNAL",
				"message": "Failed to get note",
			})
		}
		return
	}

	// Check optimistic concurrency
	if req.UpdatedAt != "" {
		expectedTime, err := time.Parse(time.RFC3339, req.UpdatedAt)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"error":   "VALIDATION_ERROR",
				"message": "invalid updated_at format",
			})
			return
		}
		
		if !note.UpdatedAt.Equal(expectedTime) {
			c.JSON(http.StatusConflict, gin.H{
				"error":   "VERSION_CONFLICT",
				"message": "note has been modified by another client",
				"server_updated_at": note.UpdatedAt.Format(time.RFC3339),
				"server_snapshot": gin.H{
					"id":           note.ID,
					"user_id":      note.UserID,
					"folder_id":    note.FolderID,
					"title":        note.Title,
					"slug":         note.Slug,
					"content_md":   note.ContentMd,
					"content_html": note.ContentHtml,
					"is_published": note.IsPublished,
					"visibility":   note.Visibility,
					"sort_order":   note.SortOrder,
					"created_at":   note.CreatedAt.Format(time.RFC3339),
					"updated_at":   note.UpdatedAt.Format(time.RFC3339),
				},
			})
			return
		}
	}

	// Update note fields
	hasChanges := false
	
	if req.Title != nil {
		note.Title = *req.Title
		note.Slug = generateUniqueSlug(*req.Title, userID.(uint), &noteID)
		hasChanges = true
	}
	
	if req.FolderID != nil {
		note.FolderID = req.FolderID
		hasChanges = true
	}
	
	if req.ContentMd != nil && *req.ContentMd != note.ContentMd {
		note.ContentMd = *req.ContentMd
		note.ContentHtml = convertMarkdownToHTML(*req.ContentMd)
		hasChanges = true
	}
	
	if req.IsPublished != nil {
		note.IsPublished = *req.IsPublished
		hasChanges = true
	}
	
	if req.Visibility != nil {
		note.Visibility = *req.Visibility
		hasChanges = true
	}

	// If no changes, return current note (idempotent)
	if !hasChanges {
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
		
		if note.FolderID != nil {
			response["folder_id"] = *note.FolderID
		} else {
			response["folder_id"] = nil
		}
		
		c.JSON(http.StatusOK, response)
		return
	}

	// Update timestamp
	note.UpdatedAt = time.Now()

	// Save to database
	if err := database.DB.Save(&note).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "INTERNAL",
			"message": "Failed to update note",
		})
		return
	}

	// Return updated note
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
	
	if note.FolderID != nil {
		response["folder_id"] = *note.FolderID
	} else {
		response["folder_id"] = nil
	}
	
	c.JSON(http.StatusOK, response)
}

// ------------------------------------------------------------
// Delete Note
// DELETE /api/v1/notes/{id}
// ------------------------------------------------------------

func DeleteNote(c *gin.Context) {
	// Get note ID from path
	idStr := c.Param("id")

	// Validate ID is a number
	idInt, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "VALIDATION_ERROR",
			"message": "invalid note ID",
		})
		return
	}
	noteID := uint(idInt)

	// Get user ID from JWT token
	userID, exists := c.Get("userID")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error":   "UNAUTHORIZED",
			"message": "invalid user ID",
		})
		return
	}

	// Get note
	var note model.Note
	if err := database.DB.Where("id = ? AND user_id = ?", noteID, userID).First(&note).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "NOT_FOUND",
				"message": "note not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "INTERNAL",
				"message": "Failed to get note",
			})
		}
		return
	}

	// Delete note
	if err := database.DB.Delete(&note).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "INTERNAL",
			"message": "Failed to delete note",
		})
		return
	}
	
	// Return success
	c.Status(http.StatusNoContent)
}
