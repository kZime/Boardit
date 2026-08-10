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
