// internal/handler/note.go
package handler

import (
	"backend/internal/database"
	"backend/internal/model"
	"backend/internal/response"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type optionalUint struct {
	Set   bool
	Value *uint
}

func (value *optionalUint) UnmarshalJSON(data []byte) error {
	value.Set = true
	if string(data) == "null" {
		value.Value = nil
		return nil
	}
	var parsed uint
	if err := json.Unmarshal(data, &parsed); err != nil {
		return err
	}
	value.Value = &parsed
	return nil
}

func formatTimestamp(value time.Time) string {
	return value.UTC().Format(time.RFC3339Nano)
}

func nowTimestamp() time.Time {
	return time.Now().UTC().Truncate(time.Microsecond)
}

// stripFrontmatter removes the leading --- ... --- frontmatter block from markdown.
// If no frontmatter is present the original string is returned unchanged.
func stripFrontmatter(md string) string {
	trimmed := strings.TrimSpace(md)
	if !strings.HasPrefix(trimmed, "---") {
		return md
	}
	lines := strings.Split(trimmed, "\n")
	if strings.TrimSpace(lines[0]) != "---" {
		return md
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			body := strings.Join(lines[i+1:], "\n")
			return strings.TrimSpace(body)
		}
	}
	// No closing ---; return as-is
	return md
}

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
	if offset < 0 {
		offset = 0
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
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token")
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
		response.Error(c, http.StatusInternalServerError, "INTERNAL", "Failed to get notes")
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
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	// Get user ID from JWT token
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid token")
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
			response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "folder not found or access denied")
			return
		}
	}

	// Convert markdown to HTML (basic implementation)
	contentHtml := convertMarkdownToHTML(req.ContentMd)

	now := nowTimestamp()
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
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := database.DB.Create(&note).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL", "Failed to create note")
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
		"created_at":   formatTimestamp(note.CreatedAt),
		"updated_at":   formatTimestamp(note.UpdatedAt),
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

	// Escape raw HTML before adding the small supported Markdown subset.
	markdown = html.EscapeString(markdown)

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
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid note ID")
		return
	}
	noteID := uint(idInt)

	// Get user ID from JWT token
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user ID")
		return
	}

	// Search for note
	var note model.Note
	if err := database.DB.Where("id = ? AND user_id = ?", noteID, userID).First(&note).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "note not found")
		} else {
			response.Error(c, http.StatusInternalServerError, "INTERNAL", "Failed to get note")
		}
		return
	}

	// Return the note
	response := gin.H{
		"id":           note.ID,
		"user_id":      note.UserID,
		"title":        note.Title,
		"slug":         note.Slug,
		"cover_url":    note.CoverURL,
		"content_md":   note.ContentMd,
		"content_html": note.ContentHtml,
		"is_published": note.IsPublished,
		"visibility":   note.Visibility,
		"sort_order":   note.SortOrder,
		"created_at":   formatTimestamp(note.CreatedAt),
		"updated_at":   formatTimestamp(note.UpdatedAt),
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
	Title       *string      `json:"title"`
	FolderID    optionalUint `json:"folder_id"`
	CoverURL    *string      `json:"cover_url"`
	ContentMd   *string      `json:"content_md"`
	IsPublished *bool        `json:"is_published"`
	Visibility  *string      `json:"visibility"`
	UpdatedAt   string       `json:"updated_at"`
}

func UpdateNote(c *gin.Context) {
	// Get note ID from path
	idStr := c.Param("id")

	// Validate ID is a number
	idInt, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid note ID")
		return
	}
	noteID := uint(idInt)

	// Get user ID from JWT token
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user ID")
		return
	}

	// Parse request body
	var req updateNoteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	// Get note
	var note model.Note
	if err := database.DB.Where("id = ? AND user_id = ?", noteID, userID).First(&note).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "note not found")
		} else {
			response.Error(c, http.StatusInternalServerError, "INTERNAL", "Failed to get note")
		}
		return
	}

	// Check optimistic concurrency
	if req.UpdatedAt != "" {
		expectedTime, err := time.Parse(time.RFC3339, req.UpdatedAt)
		if err != nil {
			response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid updated_at format")
			return
		}

		if !note.UpdatedAt.Equal(expectedTime) {
			c.JSON(http.StatusConflict, gin.H{
				"error":             "VERSION_CONFLICT",
				"message":           "note has been modified by another client",
				"server_updated_at": formatTimestamp(note.UpdatedAt),
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
					"created_at":   formatTimestamp(note.CreatedAt),
					"updated_at":   formatTimestamp(note.UpdatedAt),
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

	if req.FolderID.Set {
		if req.FolderID.Value != nil {
			var folder model.Folder
			if err := database.DB.Where("id = ? AND user_id = ?", *req.FolderID.Value, userID).First(&folder).Error; err != nil {
				response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "folder not found or access denied")
				return
			}
		}
		note.FolderID = req.FolderID.Value
		hasChanges = true
	}

	if req.CoverURL != nil && *req.CoverURL != note.CoverURL {
		note.CoverURL = *req.CoverURL
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
			"cover_url":    note.CoverURL,
			"content_md":   note.ContentMd,
			"content_html": note.ContentHtml,
			"is_published": note.IsPublished,
			"visibility":   note.Visibility,
			"sort_order":   note.SortOrder,
			"created_at":   formatTimestamp(note.CreatedAt),
			"updated_at":   formatTimestamp(note.UpdatedAt),
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
	note.UpdatedAt = nowTimestamp()

	// Save to database
	if err := database.DB.Save(&note).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL", "Failed to update note")
		return
	}

	// Return updated note
	response := gin.H{
		"id":           note.ID,
		"user_id":      note.UserID,
		"title":        note.Title,
		"slug":         note.Slug,
		"cover_url":    note.CoverURL,
		"content_md":   note.ContentMd,
		"content_html": note.ContentHtml,
		"is_published": note.IsPublished,
		"visibility":   note.Visibility,
		"sort_order":   note.SortOrder,
		"created_at":   formatTimestamp(note.CreatedAt),
		"updated_at":   formatTimestamp(note.UpdatedAt),
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
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid note ID")
		return
	}
	noteID := uint(idInt)

	// Get user ID from JWT token
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user ID")
		return
	}

	// Get note
	var note model.Note
	if err := database.DB.Where("id = ? AND user_id = ?", noteID, userID).First(&note).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "note not found")
		} else {
			response.Error(c, http.StatusInternalServerError, "INTERNAL", "Failed to get note")
		}
		return
	}

	// Delete note
	if err := database.DB.Delete(&note).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL", "Failed to delete note")
		return
	}

	// Return success
	c.Status(http.StatusNoContent)
}

// ------------------------------------------------------------
// Create Folder
// GET /api/v1/folders
// ------------------------------------------------------------

func ListFolders(c *gin.Context) {
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user ID")
		return
	}
	var folders []model.Folder
	if err := database.DB.Where("user_id = ?", userID).
		Order("sort_order asc, name asc").Find(&folders).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL", "failed to list folders")
		return
	}
	result := make([]map[string]interface{}, len(folders))
	for i, f := range folders {
		entry := map[string]interface{}{
			"id":         f.ID,
			"user_id":    f.UserID,
			"name":       f.Name,
			"sort_order": f.SortOrder,
			"created_at": formatTimestamp(f.CreatedAt),
			"updated_at": formatTimestamp(f.UpdatedAt),
		}
		if f.ParentID != nil {
			entry["parent_id"] = *f.ParentID
		} else {
			entry["parent_id"] = nil
		}
		result[i] = entry
	}
	c.JSON(http.StatusOK, gin.H{"items": result})
}

// POST /api/v1/folders
// Body:
// - name: string
// - parent_id: uint
// ------------------------------------------------------------

type createFolderRequest struct {
	Name     string `json:"name" binding:"required"`
	ParentID *uint  `json:"parent_id"`
}

func CreateFolder(c *gin.Context) {
	var request createFolderRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	// Get user ID from JWT token
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user ID")
		return
	}

	request.Name = strings.TrimSpace(request.Name)
	if request.Name == "" {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "folder name is required")
		return
	}
	if request.ParentID != nil {
		var parent model.Folder
		if err := database.DB.Where("id = ? AND user_id = ?", *request.ParentID, userID).First(&parent).Error; err != nil {
			response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "parent folder not found or access denied")
			return
		}
	}

	// Name is required, no need for default value

	now := nowTimestamp()
	// Create folder
	folder := model.Folder{
		UserID:    userID.(uint),
		Name:      request.Name,
		ParentID:  request.ParentID,
		SortOrder: 0,
		CreatedAt: now,
		UpdatedAt: now,
	}

	// Save to database
	if err := database.DB.Create(&folder).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL", "Failed to create folder")
		return
	}

	// Return the created folder
	response := gin.H{
		"id":         folder.ID,
		"user_id":    folder.UserID,
		"name":       folder.Name,
		"parent_id":  folder.ParentID,
		"sort_order": folder.SortOrder,
		"created_at": formatTimestamp(folder.CreatedAt),
		"updated_at": formatTimestamp(folder.UpdatedAt),
	}

	c.JSON(http.StatusCreated, response)
}

// ------------------------------------------------------------
// Update Folder
// PATCH /api/v1/folders/{id}
// Body:
// - name: string
// - parent_id: uint
// ------------------------------------------------------------

type updateFolderRequest struct {
	Name     string       `json:"name"`
	ParentID optionalUint `json:"parent_id"`
}

func folderParentWouldCycle(db *gorm.DB, userID, folderID, parentID uint) (bool, error) {
	visited := map[uint]struct{}{folderID: {}}
	currentID := parentID
	for {
		if _, exists := visited[currentID]; exists {
			return true, nil
		}
		visited[currentID] = struct{}{}

		var current model.Folder
		if err := db.Where("id = ? AND user_id = ?", currentID, userID).First(&current).Error; err != nil {
			return false, err
		}
		if current.ParentID == nil {
			return false, nil
		}
		currentID = *current.ParentID
	}
}

func UpdateFolder(c *gin.Context) {
	// Get folder ID from path
	idStr := c.Param("id")

	// Validate ID is a number
	idInt, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid folder ID")
		return
	}
	folderID := uint(idInt)

	// Get user ID from JWT token
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user ID")
		return
	}

	// Get request body
	var req updateFolderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}

	// Get folder
	var folder model.Folder
	if err := database.DB.Where("id = ? AND user_id = ?", folderID, userID).First(&folder).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "folder not found")
		} else {
			response.Error(c, http.StatusInternalServerError, "INTERNAL", "Failed to get folder")
		}
		return
	}

	// Update folder fields
	hasChanges := false

	if req.Name != "" {
		name := strings.TrimSpace(req.Name)
		if name == "" {
			response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "folder name is required")
			return
		}
		folder.Name = name
		hasChanges = true
	}

	if req.ParentID.Set {
		if req.ParentID.Value != nil {
			wouldCycle, err := folderParentWouldCycle(database.DB, userID.(uint), folderID, *req.ParentID.Value)
			if err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "parent folder not found or access denied")
				} else {
					response.Error(c, http.StatusInternalServerError, "INTERNAL", "Failed to validate parent folder")
				}
				return
			}
			if wouldCycle {
				response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "folder parent would create a cycle")
				return
			}
		}
		folder.ParentID = req.ParentID.Value
		hasChanges = true
	}

	// If no changes, return current folder (idempotent)
	if !hasChanges {
		response := gin.H{
			"id":         folder.ID,
			"user_id":    folder.UserID,
			"name":       folder.Name,
			"sort_order": folder.SortOrder,
			"created_at": formatTimestamp(folder.CreatedAt),
			"updated_at": formatTimestamp(folder.UpdatedAt),
		}

		if folder.ParentID != nil {
			response["parent_id"] = *folder.ParentID
		} else {
			response["parent_id"] = nil
		}

		c.JSON(http.StatusOK, response)
		return
	}

	// Update timestamp
	folder.UpdatedAt = nowTimestamp()

	// Save to database
	if err := database.DB.Save(&folder).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL", "Failed to update folder")
		return
	}

	// Return updated folder
	response := gin.H{
		"id":         folder.ID,
		"user_id":    folder.UserID,
		"name":       folder.Name,
		"sort_order": folder.SortOrder,
		"created_at": formatTimestamp(folder.CreatedAt),
		"updated_at": formatTimestamp(folder.UpdatedAt),
	}

	if folder.ParentID != nil {
		response["parent_id"] = *folder.ParentID
	} else {
		response["parent_id"] = nil
	}

	c.JSON(http.StatusOK, response)
}

// ------------------------------------------------------------
// Delete Folder
// DELETE /api/v1/folders/{id}
// ------------------------------------------------------------

func DeleteFolder(c *gin.Context) {
	// Get folder ID from path
	idStr := c.Param("id")

	// Validate ID is a number
	idInt, err := strconv.ParseInt(idStr, 10, 32)
	if err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid folder ID")
		return
	}
	folderID := uint(idInt)

	// Get user ID from JWT token
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user ID")
		return
	}

	// Get folder
	var folder model.Folder
	if err := database.DB.Where("id = ? AND user_id = ?", folderID, userID).First(&folder).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "folder not found")
		} else {
			response.Error(c, http.StatusInternalServerError, "INTERNAL", "Failed to get folder")
		}
		return
	}

	// Check if folder is empty (no subfolders or notes)
	var subfolderCount int64
	database.DB.Model(&model.Folder{}).Where("parent_id = ? AND user_id = ?", folderID, userID).Count(&subfolderCount)

	var noteCount int64
	database.DB.Model(&model.Note{}).Where("folder_id = ? AND user_id = ?", folderID, userID).Count(&noteCount)

	if subfolderCount > 0 || noteCount > 0 {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "folder is not empty")
		return
	}

	// Delete folder
	if err := database.DB.Delete(&folder).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL", "Failed to delete folder")
		return
	}

	// Return success
	c.Status(http.StatusNoContent)
}

// ------------------------------------------------------------
// Reorder Tree
// POST /api/v1/tree/reorder
// Body:
// - folders: array of objects with id and sort_order
// - notes: array of objects with id, sort_order, and folder_id
// ------------------------------------------------------------

type reorderTreeRequest struct {
	Folders []struct {
		ID        uint `json:"id"`
		SortOrder int  `json:"sort_order"`
	} `json:"folders"`
	Notes []struct {
		ID        uint  `json:"id"`
		SortOrder int   `json:"sort_order"`
		FolderID  *uint `json:"folder_id"`
	} `json:"notes"`
}

type treeMutationError struct {
	status  int
	code    string
	message string
}

func (err *treeMutationError) Error() string {
	return err.message
}

func ReorderTree(c *gin.Context) {
	var req reorderTreeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid payload")
		return
	}

	// Get user ID from JWT token
	userID, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user ID")
		return
	}

	err := database.DB.Transaction(func(tx *gorm.DB) error {
		for _, folder := range req.Folders {
			var storedFolder model.Folder
			if err := tx.Where("id = ? AND user_id = ?", folder.ID, userID).First(&storedFolder).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return &treeMutationError{status: http.StatusNotFound, code: "NOT_FOUND", message: "folder not found"}
				}
				return &treeMutationError{status: http.StatusInternalServerError, code: "INTERNAL", message: "Failed to get folder"}
			}
			storedFolder.SortOrder = folder.SortOrder
			if err := tx.Save(&storedFolder).Error; err != nil {
				return &treeMutationError{status: http.StatusInternalServerError, code: "INTERNAL", message: "Failed to update folder"}
			}
		}

		for _, note := range req.Notes {
			var storedNote model.Note
			if err := tx.Where("id = ? AND user_id = ?", note.ID, userID).First(&storedNote).Error; err != nil {
				if errors.Is(err, gorm.ErrRecordNotFound) {
					return &treeMutationError{status: http.StatusNotFound, code: "NOT_FOUND", message: "note not found"}
				}
				return &treeMutationError{status: http.StatusInternalServerError, code: "INTERNAL", message: "Failed to get note"}
			}
			if note.FolderID != nil {
				var folder model.Folder
				if err := tx.Where("id = ? AND user_id = ?", *note.FolderID, userID).First(&folder).Error; err != nil {
					if errors.Is(err, gorm.ErrRecordNotFound) {
						return &treeMutationError{status: http.StatusBadRequest, code: "VALIDATION_ERROR", message: "folder not found or access denied"}
					}
					return &treeMutationError{status: http.StatusInternalServerError, code: "INTERNAL", message: "Failed to validate folder"}
				}
			}
			storedNote.SortOrder = note.SortOrder
			storedNote.FolderID = note.FolderID
			if err := tx.Save(&storedNote).Error; err != nil {
				return &treeMutationError{status: http.StatusInternalServerError, code: "INTERNAL", message: "Failed to update note"}
			}
		}
		return nil
	})
	if err != nil {
		var mutationErr *treeMutationError
		if errors.As(err, &mutationErr) {
			response.Error(c, mutationErr.status, mutationErr.code, mutationErr.message)
		} else {
			response.Error(c, http.StatusInternalServerError, "INTERNAL", "Failed to reorder tree")
		}
		return
	}

	// Return success
	c.Status(http.StatusOK)
}

// ------------------------------------------------------------
// List Public Notes (no auth)
// GET /api/v1/public/notes
// Query: limit, offset
// ------------------------------------------------------------

func ListPublicNotes(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	offsetStr := c.DefaultQuery("offset", "0")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	offset, err := strconv.Atoi(offsetStr)
	if err != nil || offset < 0 {
		offset = 0
	}

	var notes []model.Note
	query := database.DB.Where("visibility = ? AND is_published = ?", "public", true).
		Order("updated_at DESC").Limit(limit).Offset(offset)
	if err := query.Find(&notes).Error; err != nil {
		response.Error(c, http.StatusInternalServerError, "INTERNAL", "Failed to get public notes")
		return
	}

	var total int64
	database.DB.Model(&model.Note{}).Where("visibility = ? AND is_published = ?", "public", true).Count(&total)

	if len(notes) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"items":  []gin.H{},
			"total":  total,
			"limit":  limit,
			"offset": offset,
		})
		return
	}

	userIDs := make([]uint, 0, len(notes))
	for _, n := range notes {
		userIDs = append(userIDs, n.UserID)
	}
	var users []struct {
		ID       uint   `gorm:"column:id"`
		Username string `gorm:"column:username"`
	}
	database.DB.Model(&model.User{}).Where("id IN ?", userIDs).Select("id", "username").Find(&users)
	usernameByID := make(map[uint]string)
	for _, u := range users {
		usernameByID[u.ID] = u.Username
	}

	items := make([]gin.H, 0, len(notes))
	for _, n := range notes {
		// Strip frontmatter block so excerpt shows real article body text
		body := stripFrontmatter(n.ContentMd)
		excerpt := body
		if len([]rune(excerpt)) > 200 {
			excerpt = string([]rune(excerpt)[:200]) + "..."
		}
		items = append(items, gin.H{
			"id":              n.ID,
			"title":           n.Title,
			"slug":            n.Slug,
			"user_id":         n.UserID,
			"author_username": usernameByID[n.UserID],
			"excerpt":         excerpt,
			"cover_url":       n.CoverURL,
			"created_at":      formatTimestamp(n.CreatedAt),
			"updated_at":      formatTimestamp(n.UpdatedAt),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"items":  items,
		"total":  total,
		"limit":  limit,
		"offset": offset,
	})
}

// ------------------------------------------------------------
// Get Public Note by username and slug (no auth)
// GET /api/v1/public/notes/:username/:slug
// ------------------------------------------------------------

func GetPublicNote(c *gin.Context) {
	username := c.Param("username")
	slug := c.Param("slug")
	if username == "" || slug == "" {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "username and slug required")
		return
	}

	var user model.User
	if err := database.DB.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "user not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL", "Failed to get user")
		return
	}

	var note model.Note
	if err := database.DB.Where("user_id = ? AND slug = ? AND visibility IN ? AND is_published = ?",
		user.ID, slug, []string{"public", "unlisted"}, true).First(&note).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			response.Error(c, http.StatusNotFound, "NOT_FOUND", "note not found")
			return
		}
		response.Error(c, http.StatusInternalServerError, "INTERNAL", "Failed to get note")
		return
	}

	resp := gin.H{
		"id":              note.ID,
		"user_id":         note.UserID,
		"title":           note.Title,
		"slug":            note.Slug,
		"cover_url":       note.CoverURL,
		"content_md":      note.ContentMd,
		"content_html":    note.ContentHtml,
		"is_published":    note.IsPublished,
		"visibility":      note.Visibility,
		"created_at":      formatTimestamp(note.CreatedAt),
		"updated_at":      formatTimestamp(note.UpdatedAt),
		"author_username": user.Username,
	}
	if note.FolderID != nil {
		resp["folder_id"] = *note.FolderID
	} else {
		resp["folder_id"] = nil
	}
	c.JSON(http.StatusOK, resp)
}
