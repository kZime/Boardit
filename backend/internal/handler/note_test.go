package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"backend/internal/database"
	"backend/internal/model"
	"backend/internal/testutils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

type NoteHandlerTestSuite struct {
	suite.Suite
	router *gin.Engine
	user1  model.User
	user2  model.User
}

func TestNoteHandlerSuite(t *testing.T) {
	suite.Run(t, new(NoteHandlerTestSuite))
}

func (suite *NoteHandlerTestSuite) SetupSuite() {
	testutils.LoadTestEnv()
	suite.Require().NoError(database.Init())

	gin.SetMode(gin.TestMode)
	suite.router = gin.New()
	suite.router.Use(func(c *gin.Context) {
		if rawID := c.GetHeader("X-Test-User-ID"); rawID != "" {
			id, err := strconv.ParseUint(rawID, 10, 64)
			if err == nil {
				c.Set("userID", uint(id))
			}
		}
		c.Next()
	})
	suite.router.GET("/api/v1/notes", ListNotes)
	suite.router.POST("/api/v1/notes", CreateNote)
	suite.router.GET("/api/v1/notes/:id", GetNote)
	suite.router.PATCH("/api/v1/notes/:id", UpdateNote)
	suite.router.DELETE("/api/v1/notes/:id", DeleteNote)
	suite.router.GET("/api/v1/folders", ListFolders)
	suite.router.POST("/api/v1/folders", CreateFolder)
	suite.router.PATCH("/api/v1/folders/:id", UpdateFolder)
	suite.router.DELETE("/api/v1/folders/:id", DeleteFolder)
	suite.router.POST("/api/v1/tree/reorder", ReorderTree)
	suite.router.GET("/api/v1/public/notes", ListPublicNotes)
	suite.router.GET("/api/v1/public/notes/:username/:slug", GetPublicNote)
}

func (suite *NoteHandlerTestSuite) SetupTest() {
	suite.Require().NoError(database.TruncateAllTables())
	now := time.Now()
	suite.user1 = model.User{
		Username: "writer-one", Email: "writer-one@example.com", PasswordHash: "test", CreatedAt: now, UpdatedAt: now,
	}
	suite.user2 = model.User{
		Username: "writer-two", Email: "writer-two@example.com", PasswordHash: "test", CreatedAt: now, UpdatedAt: now,
	}
	suite.Require().NoError(database.DB.Create(&suite.user1).Error)
	suite.Require().NoError(database.DB.Create(&suite.user2).Error)
}

func (suite *NoteHandlerTestSuite) TearDownSuite() {
	suite.Require().NoError(database.TruncateAllTables())
}

func (suite *NoteHandlerTestSuite) request(method, path string, userID uint, payload any) *httptest.ResponseRecorder {
	var body bytes.Buffer
	if payload != nil {
		suite.Require().NoError(json.NewEncoder(&body).Encode(payload))
	}
	req := httptest.NewRequest(method, path, &body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if userID != 0 {
		req.Header.Set("X-Test-User-ID", strconv.FormatUint(uint64(userID), 10))
	}
	recorder := httptest.NewRecorder()
	suite.router.ServeHTTP(recorder, req)
	return recorder
}

func (suite *NoteHandlerTestSuite) decodeObject(recorder *httptest.ResponseRecorder) map[string]any {
	var result map[string]any
	suite.Require().NoError(json.Unmarshal(recorder.Body.Bytes(), &result))
	return result
}

func (suite *NoteHandlerTestSuite) TestNoteAndFolderLifecycle() {
	folderResponse := suite.request(http.MethodPost, "/api/v1/folders", suite.user1.ID, map[string]any{
		"name": "Projects",
	})
	suite.Equal(http.StatusCreated, folderResponse.Code)
	folderID := uint(suite.decodeObject(folderResponse)["id"].(float64))

	createResponse := suite.request(http.MethodPost, "/api/v1/notes", suite.user1.ID, map[string]any{
		"title": "AI roadmap", "folder_id": folderID, "content_md": "# First draft",
	})
	suite.Equal(http.StatusCreated, createResponse.Code)
	created := suite.decodeObject(createResponse)
	noteID := uint(created["id"].(float64))
	suite.Equal("ai-roadmap", created["slug"])

	getResponse := suite.request(http.MethodGet, fmt.Sprintf("/api/v1/notes/%d", noteID), suite.user1.ID, nil)
	suite.Equal(http.StatusOK, getResponse.Code)
	suite.Equal("# First draft", suite.decodeObject(getResponse)["content_md"])

	updateResponse := suite.request(http.MethodPatch, fmt.Sprintf("/api/v1/notes/%d", noteID), suite.user1.ID, map[string]any{
		"title": "AI roadmap updated", "content_md": "# Second draft",
	})
	suite.Equal(http.StatusOK, updateResponse.Code)
	suite.Equal("ai-roadmap-updated", suite.decodeObject(updateResponse)["slug"])

	listResponse := suite.request(http.MethodGet, "/api/v1/notes?q=Second", suite.user1.ID, nil)
	suite.Equal(http.StatusOK, listResponse.Code)
	listed := suite.decodeObject(listResponse)
	suite.Equal(float64(1), listed["total"])

	foldersResponse := suite.request(http.MethodGet, "/api/v1/folders", suite.user1.ID, nil)
	suite.Equal(http.StatusOK, foldersResponse.Code)
	suite.Len(suite.decodeObject(foldersResponse)["items"], 1)

	deleteResponse := suite.request(http.MethodDelete, fmt.Sprintf("/api/v1/notes/%d", noteID), suite.user1.ID, nil)
	suite.Equal(http.StatusNoContent, deleteResponse.Code)
	suite.Equal(http.StatusNotFound, suite.request(http.MethodGet, fmt.Sprintf("/api/v1/notes/%d", noteID), suite.user1.ID, nil).Code)
}

func (suite *NoteHandlerTestSuite) TestPrivateResourcesAreIsolatedByUser() {
	createResponse := suite.request(http.MethodPost, "/api/v1/notes", suite.user1.ID, map[string]any{
		"title": "Private", "content_md": "secret",
	})
	suite.Equal(http.StatusCreated, createResponse.Code)
	noteID := uint(suite.decodeObject(createResponse)["id"].(float64))

	suite.Equal(http.StatusNotFound, suite.request(http.MethodGet, fmt.Sprintf("/api/v1/notes/%d", noteID), suite.user2.ID, nil).Code)
	otherUserList := suite.decodeObject(suite.request(http.MethodGet, "/api/v1/notes", suite.user2.ID, nil))
	suite.Equal(float64(0), otherUserList["total"])
}

func (suite *NoteHandlerTestSuite) TestListNotesClampsNegativeOffset() {
	response := suite.request(http.MethodGet, "/api/v1/notes?offset=-10", suite.user1.ID, nil)

	suite.Equal(http.StatusOK, response.Code)
	suite.Equal(float64(0), suite.decodeObject(response)["offset"])
}

func (suite *NoteHandlerTestSuite) TestMarkdownConversionEscapesRawHTML() {
	converted := convertMarkdownToHTML(`<script>alert("xss")</script> **safe**`)

	suite.NotContains(converted, "<script>")
	suite.Contains(converted, "&lt;script&gt;")
	suite.Contains(converted, "<strong>safe</strong>")
}

func (suite *NoteHandlerTestSuite) TestUpdatedAtCanBeRoundTrippedForOptimisticConcurrency() {
	createResponse := suite.request(http.MethodPost, "/api/v1/notes", suite.user1.ID, map[string]any{
		"title": "Concurrent", "content_md": "v1",
	})
	suite.Equal(http.StatusCreated, createResponse.Code)
	created := suite.decodeObject(createResponse)
	noteID := uint(created["id"].(float64))
	version := created["updated_at"].(string)

	updateResponse := suite.request(http.MethodPatch, fmt.Sprintf("/api/v1/notes/%d", noteID), suite.user1.ID, map[string]any{
		"content_md": "v2", "updated_at": version,
	})
	suite.Equal(http.StatusOK, updateResponse.Code)
	suite.NotEqual(version, suite.decodeObject(updateResponse)["updated_at"])

	conflictResponse := suite.request(http.MethodPatch, fmt.Sprintf("/api/v1/notes/%d", noteID), suite.user1.ID, map[string]any{
		"content_md": "stale write", "updated_at": version,
	})
	suite.Equal(http.StatusConflict, conflictResponse.Code)
}

func (suite *NoteHandlerTestSuite) TestFolderAssignmentsRequireOwnershipAndAllowMovingToRoot() {
	now := time.Now()
	ownFolder := model.Folder{UserID: suite.user1.ID, Name: "Mine", CreatedAt: now, UpdatedAt: now}
	foreignFolder := model.Folder{UserID: suite.user2.ID, Name: "Theirs", CreatedAt: now, UpdatedAt: now}
	suite.Require().NoError(database.DB.Create(&ownFolder).Error)
	suite.Require().NoError(database.DB.Create(&foreignFolder).Error)

	createResponse := suite.request(http.MethodPost, "/api/v1/notes", suite.user1.ID, map[string]any{
		"title": "Move me", "folder_id": ownFolder.ID,
	})
	suite.Equal(http.StatusCreated, createResponse.Code)
	noteID := uint(suite.decodeObject(createResponse)["id"].(float64))

	foreignResponse := suite.request(http.MethodPatch, fmt.Sprintf("/api/v1/notes/%d", noteID), suite.user1.ID, map[string]any{
		"folder_id": foreignFolder.ID,
	})
	suite.Equal(http.StatusBadRequest, foreignResponse.Code)

	rootResponse := suite.request(http.MethodPatch, fmt.Sprintf("/api/v1/notes/%d", noteID), suite.user1.ID, map[string]any{
		"folder_id": nil,
	})
	suite.Equal(http.StatusOK, rootResponse.Code)
	var note model.Note
	suite.Require().NoError(database.DB.First(&note, noteID).Error)
	suite.Nil(note.FolderID)
}

func (suite *NoteHandlerTestSuite) TestFolderParentsRequireOwnershipAndCannotFormCycles() {
	now := time.Now()
	parent := model.Folder{UserID: suite.user1.ID, Name: "Parent", CreatedAt: now, UpdatedAt: now}
	suite.Require().NoError(database.DB.Create(&parent).Error)
	child := model.Folder{UserID: suite.user1.ID, Name: "Child", ParentID: &parent.ID, CreatedAt: now, UpdatedAt: now}
	suite.Require().NoError(database.DB.Create(&child).Error)
	foreign := model.Folder{UserID: suite.user2.ID, Name: "Foreign", CreatedAt: now, UpdatedAt: now}
	suite.Require().NoError(database.DB.Create(&foreign).Error)

	createResponse := suite.request(http.MethodPost, "/api/v1/folders", suite.user1.ID, map[string]any{
		"name": "Invalid", "parent_id": foreign.ID,
	})
	suite.Equal(http.StatusBadRequest, createResponse.Code)

	for name, parentID := range map[string]uint{
		"self":       parent.ID,
		"descendant": child.ID,
		"foreign":    foreign.ID,
	} {
		suite.Run(name, func() {
			response := suite.request(http.MethodPatch, fmt.Sprintf("/api/v1/folders/%d", parent.ID), suite.user1.ID, map[string]any{
				"parent_id": parentID,
			})
			suite.Equal(http.StatusBadRequest, response.Code)
		})
	}
}

func (suite *NoteHandlerTestSuite) TestReorderIsAtomicAndRejectsForeignFolders() {
	now := time.Now()
	folder := model.Folder{UserID: suite.user1.ID, Name: "Mine", CreatedAt: now, UpdatedAt: now}
	foreign := model.Folder{UserID: suite.user2.ID, Name: "Foreign", CreatedAt: now, UpdatedAt: now}
	suite.Require().NoError(database.DB.Create(&folder).Error)
	suite.Require().NoError(database.DB.Create(&foreign).Error)
	note := model.Note{UserID: suite.user1.ID, Title: "Note", Slug: "note", ContentMd: "text", ContentHtml: "text", Visibility: "private", CreatedAt: now, UpdatedAt: now}
	suite.Require().NoError(database.DB.Create(&note).Error)

	atomicResponse := suite.request(http.MethodPost, "/api/v1/tree/reorder", suite.user1.ID, map[string]any{
		"folders": []map[string]any{{"id": folder.ID, "sort_order": 7}},
		"notes":   []map[string]any{{"id": 999999, "sort_order": 1, "folder_id": folder.ID}},
	})
	suite.Equal(http.StatusNotFound, atomicResponse.Code)
	suite.Require().NoError(database.DB.First(&folder, folder.ID).Error)
	suite.Equal(0, folder.SortOrder)

	foreignResponse := suite.request(http.MethodPost, "/api/v1/tree/reorder", suite.user1.ID, map[string]any{
		"notes": []map[string]any{{"id": note.ID, "sort_order": 1, "folder_id": foreign.ID}},
	})
	suite.Equal(http.StatusBadRequest, foreignResponse.Code)
	suite.Require().NoError(database.DB.First(&note, note.ID).Error)
	suite.Nil(note.FolderID)
}

func (suite *NoteHandlerTestSuite) TestPublicListingAndPermalinks() {
	now := time.Now()
	notes := []model.Note{
		{UserID: suite.user1.ID, Title: "Public", Slug: "public", ContentMd: "---\ncover: test\n---\nPublic body", ContentHtml: "Public body", IsPublished: true, Visibility: "public", CreatedAt: now, UpdatedAt: now},
		{UserID: suite.user1.ID, Title: "Unlisted", Slug: "unlisted", ContentMd: "Unlisted body", ContentHtml: "Unlisted body", IsPublished: true, Visibility: "unlisted", CreatedAt: now, UpdatedAt: now},
		{UserID: suite.user1.ID, Title: "Private", Slug: "private", ContentMd: "Private body", ContentHtml: "Private body", IsPublished: true, Visibility: "private", CreatedAt: now, UpdatedAt: now},
	}
	suite.Require().NoError(database.DB.Create(&notes).Error)

	listResponse := suite.request(http.MethodGet, "/api/v1/public/notes", 0, nil)
	suite.Equal(http.StatusOK, listResponse.Code)
	list := suite.decodeObject(listResponse)
	suite.Equal(float64(1), list["total"])
	items := list["items"].([]any)
	suite.Len(items, 1)
	suite.Equal("Public body", items[0].(map[string]any)["excerpt"])

	for _, slug := range []string{"public", "unlisted"} {
		response := suite.request(http.MethodGet, "/api/v1/public/notes/writer-one/"+slug, 0, nil)
		suite.Equal(http.StatusOK, response.Code)
	}
	suite.Equal(http.StatusNotFound, suite.request(http.MethodGet, "/api/v1/public/notes/writer-one/private", 0, nil).Code)
}
