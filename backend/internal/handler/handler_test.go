// internal/handler/auth_test.go

package handler

import (
	"testing"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"time"

	"backend/internal/database"
	"backend/internal/testutils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestMain(m *testing.M) {
	// Load environment variables for testing
	testutils.LoadTestEnv()

	// Initialize the database
	if err := database.Init(); err != nil {
		panic("Failed to initialize the database: " + err.Error())
	}

	// Run tests
	code := m.Run()

	// Clean up data in the database
	database.DB.Exec("DELETE FROM users")

	// Exit tests
	os.Exit(code)
}

func TestRegisterSuccess(t *testing.T) {
	// Set up router
	gin.SetMode(gin.TestMode)

	// Create router and bind handler
	router := gin.Default()
	router.POST("/api/auth/register", Register)

	// Create a payload for the request
	payload := registerRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "testpassword",
	}

	// Convert payload to JSON
	body, err := json.Marshal(payload)
	if err != nil {
		panic("Failed to marshal payload: " + err.Error())
	}

	// Create a new HTTP request
	req, err := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
	if err != nil {
		panic("Failed to create request: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check the response status code
	assert.Equal(t, http.StatusCreated, w.Code, "Expected status code 201 Created")
	
	// Check the response body
	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.Equal(t, "testuser", response["username"], "Expected username to be 'testuser'")
	assert.Equal(t, "test@example.com", response["email"], "Expected email to be 'test@example.com'")
	assert.NotEmpty(t, response["id"], "Expected user ID to be present")
	
}

func TestLoginSuccess(t *testing.T) {
	// Set up router
	gin.SetMode(gin.TestMode)

	// Create router and bind handler
	router := gin.Default()
	router.POST("/api/auth/login", Login)

	// Create a payload for the login request
	payload := loginRequest{
		Email: "test@example.com",
		Password: "testpassword",
	}

	// Convert payload to JSON
	body, err := json.Marshal(payload)
	if err != nil {
		panic("Failed to marshal payload: " + err.Error())
	}

	// Create a new HTTP request
	req, err := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	if err != nil {
		panic("Failed to create request: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Perform the request
	router.ServeHTTP(w, req)

	// Check the response status code
	assert.Equal(t, http.StatusOK, w.Code, "Expected status code 200 OK")

	// Check the response body
	var response map[string] interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	assert.NotEmpty(t, response["access_token"], "Expected access token to be present")
	assert.NotEmpty(t, response["refresh_token"], "Expected refresh token to be present")
}

func TestRefresh(t *testing.T) {
	// Set up router
	gin.SetMode(gin.TestMode)

	// Create router and bind handler
	router := gin.Default()
	router.POST("/api/auth/login", Login)
	router.POST("/api/auth/refresh", Refresh)

	// First, log in to get tokens
	payload := loginRequest{
		Email:    "test@example.com",
		Password: "testpassword",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		panic("Failed to marshal payload: " + err.Error())
	}

	req, err := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	if err != nil {
		panic("Failed to create request: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Check the response status code
	assert.Equal(t, http.StatusOK, w.Code, "Expected status code 200 OK")

	// Second, parse the response to get the refresh token
	var loginResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &loginResponse)
	refreshToken := loginResponse["refresh_token"].(string)
	accessToken := loginResponse["access_token"].(string)

	// Third, use the refresh token to get a new access token
	refreshPayload := refreshRequest{
		RefreshToken: refreshToken,
	}

	// Time Sleep for 1 second
	time.Sleep(1 * time.Second)

	refreshBody, err := json.Marshal(refreshPayload)
	if err != nil {
		panic("Failed to marshal refresh payload: " + err.Error())
	}
	refreshReq, err := http.NewRequest("POST", "/api/auth/refresh", bytes.NewBuffer(refreshBody))
	if err != nil {
		panic("Failed to create refresh request: " + err.Error())
	}
	refreshReq.Header.Set("Content-Type", "application/json")
	refreshW := httptest.NewRecorder()
	router.ServeHTTP(refreshW, refreshReq)

	// Check the response status code
	assert.Equal(t, http.StatusOK, refreshW.Code, "Expected status code 200 OK")

	// CHeck the response body
	var refreshResponse map[string]interface{}
	json.Unmarshal(refreshW.Body.Bytes(), &refreshResponse)
	assert.NotEmpty(t, refreshResponse["access_token"], "Expected new access token to be present")
	assert.NotEmpty(t, refreshResponse["refresh_token"], "Expected new refresh token to be present")

	// Check the new refresh token is different from the old one
	assert.NotEqual(t, refreshToken, refreshResponse["refresh_token"], "Expected new refresh token")

	// Check the new access token is different from the old one
	assert.NotEqual(t, accessToken, refreshResponse["access_token"], "Expected new access token")
}

func TestGetCurrentUserSuccess(t *testing.T) {
	// Set up router
	gin.SetMode(gin.TestMode)

	// Create router and bind handler
	router := gin.Default()
	router.GET("/api/user", GetCurrentUser)
	router.POST("/api/auth/login", Login)
	
	// First, log in to get tokens
	payload := loginRequest{
		Email:    "test@example.com",
		Password: "testpassword",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		panic("Failed to marshal payload: " + err.Error())
	}
	req, err := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	if err != nil {
		panic("Failed to create request: " + err.Error())
	}
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	// Parse the response to get the access token
	var loginResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &loginResponse)
	accessToken := loginResponse["access_token"].(string)

	// Use the access token to get current user info
	userReq, err := http.NewRequest("GET", "/api/user", nil)
	if err != nil {
		panic("Failed to create user request: " + err.Error())
	}
	userReq.Header.Set("Authorization", "Bearer "+accessToken)
	userW := httptest.NewRecorder()
	router.ServeHTTP(userW, userReq)

	// Check the response status code
	assert.Equal(t, http.StatusOK, userW.Code, "Expected status code 200 OK")

	// Check the response body
	var userResponse map[string] interface{}
	json.Unmarshal(userW.Body.Bytes(), &userResponse)
	assert.Equal(t, "testuser", userResponse["username"], "Expected username to be 'testuser'")
	assert.Equal(t, "test@example.com", userResponse["email"], "Expected email to be 'test@example.com'")
	assert.NotEmpty(t, userResponse["id"], "Expected user ID to be present")
}