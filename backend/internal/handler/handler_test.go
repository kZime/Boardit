// internal/handler/auth_test.go

package handler

import (
	"testing"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"time"

	"backend/internal/database"
	"backend/internal/testutils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

// AuthTestSuite holds shared state for auth tests
type AuthTestSuite struct {
	suite.Suite
	router      *gin.Engine
	userID      uint
	accessToken string
	refreshToken string
}

func TestAuthSuite(t *testing.T) {
	suite.Run(t, new(AuthTestSuite))
}

func (suite *AuthTestSuite) SetupSuite() {
	// Load environment variables for testing
	testutils.LoadTestEnv()

	// Initialize the database
	if err := database.Init(); err != nil {
		panic("Failed to initialize the database: " + err.Error())
	}

	// Set up router once for all tests
	gin.SetMode(gin.TestMode)
	suite.router = gin.Default()
	suite.router.POST("/api/auth/register", Register)
	suite.router.POST("/api/auth/login", Login)
	suite.router.POST("/api/auth/refresh", Refresh)
	// Note: For testing GetCurrentUser, we'll handle JWT middleware in the test itself
	suite.router.GET("/api/user", GetCurrentUser)
}

func (suite *AuthTestSuite) TearDownSuite() {
	// Clean up data in the database
	database.DB.Exec("DELETE FROM users")
}

func (suite *AuthTestSuite) SetupTest() {
	// Clean up before each test to ensure isolation
	database.DB.Exec("DELETE FROM users")
	// Reset test state
	suite.userID = 0
	suite.accessToken = ""
	suite.refreshToken = ""
}

// TestRegisterSuccess tests user registration
func (suite *AuthTestSuite) TestRegisterSuccess() {
	// Create a payload for the request
	payload := registerRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "testpassword",
	}

	// Convert payload to JSON
	body, err := json.Marshal(payload)
	suite.NoError(err, "Failed to marshal payload")

	// Create a new HTTP request
	req, err := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
	suite.NoError(err, "Failed to create request")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Perform the request
	suite.router.ServeHTTP(w, req)

	// Check the response status code
	suite.Equal(http.StatusCreated, w.Code, "Expected status code 201 Created")
	
	// Check the response body
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	
	suite.Equal("testuser", response["username"], "Expected username to be 'testuser'")
	suite.Equal("test@example.com", response["email"], "Expected email to be 'test@example.com'")
	suite.NotEmpty(response["id"], "Expected user ID to be present")
	
	// Store user ID for other tests
	suite.userID = uint(response["id"].(float64))
}

// TestLoginSuccess tests user login
func (suite *AuthTestSuite) TestLoginSuccess() {
	// First register a user
	suite.registerTestUser()

	// Create a payload for the login request
	payload := loginRequest{
		Email:    "test@example.com",
		Password: "testpassword",
	}

	// Convert payload to JSON
	body, err := json.Marshal(payload)
	suite.NoError(err, "Failed to marshal payload")

	// Create a new HTTP request
	req, err := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	suite.NoError(err, "Failed to create request")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Perform the request
	suite.router.ServeHTTP(w, req)

	// Check the response status code
	suite.Equal(http.StatusOK, w.Code, "Expected status code 200 OK")

	// Check the response body
	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	
	suite.NotEmpty(response["access_token"], "Expected access token to be present")
	suite.NotEmpty(response["refresh_token"], "Expected refresh token to be present")
	
	// Store tokens for other tests
	suite.accessToken = response["access_token"].(string)
	suite.refreshToken = response["refresh_token"].(string)
}

// TestRefresh tests token refresh
func (suite *AuthTestSuite) TestRefresh() {
	// First register and login to get tokens
	suite.registerTestUser()
	suite.loginTestUser()

	// Use the refresh token to get a new access token
	refreshPayload := refreshRequest{
		RefreshToken: suite.refreshToken,
	}

	// Time Sleep for 1 second to ensure token timestamp difference
	time.Sleep(1 * time.Second)

	refreshBody, err := json.Marshal(refreshPayload)
	suite.NoError(err, "Failed to marshal refresh payload")
	
	refreshReq, err := http.NewRequest("POST", "/api/auth/refresh", bytes.NewBuffer(refreshBody))
	suite.NoError(err, "Failed to create refresh request")
	refreshReq.Header.Set("Content-Type", "application/json")
	refreshW := httptest.NewRecorder()
	suite.router.ServeHTTP(refreshW, refreshReq)

	// Check the response status code
	suite.Equal(http.StatusOK, refreshW.Code, "Expected status code 200 OK")

	// Check the response body
	var refreshResponse map[string]interface{}
	err = json.Unmarshal(refreshW.Body.Bytes(), &refreshResponse)
	suite.NoError(err)
	
	suite.NotEmpty(refreshResponse["access_token"], "Expected new access token to be present")
	suite.NotEmpty(refreshResponse["refresh_token"], "Expected new refresh token to be present")

	// Check the new tokens are different from the old ones
	suite.NotEqual(suite.refreshToken, refreshResponse["refresh_token"], "Expected new refresh token")
	suite.NotEqual(suite.accessToken, refreshResponse["access_token"], "Expected new access token")
}

// TestGetCurrentUserSuccess tests getting current user info
func (suite *AuthTestSuite) TestGetCurrentUserSuccess() {
	// First register a user to get userID
	suite.registerTestUser()

	// Create a test handler that manually sets userID in context and calls GetCurrentUser
	testHandler := func(c *gin.Context) {
		c.Set("userID", suite.userID)
		GetCurrentUser(c)
	}

	// Set up a temporary route for this test
	testRouter := gin.New()
	testRouter.GET("/api/user", testHandler)

	// Create request to get current user info
	userReq, err := http.NewRequest("GET", "/api/user", nil)
	suite.NoError(err, "Failed to create user request")
	userW := httptest.NewRecorder()
	testRouter.ServeHTTP(userW, userReq)

	// Check the response status code
	suite.Equal(http.StatusOK, userW.Code, "Expected status code 200 OK")

	// Check the response body
	var userResponse map[string]interface{}
	err = json.Unmarshal(userW.Body.Bytes(), &userResponse)
	suite.NoError(err)
	
	suite.Equal("testuser", userResponse["username"], "Expected username to be 'testuser'")
	suite.Equal("test@example.com", userResponse["email"], "Expected email to be 'test@example.com'")
	suite.NotEmpty(userResponse["id"], "Expected user ID to be present")
}

// Helper methods for shared functionality

func (suite *AuthTestSuite) registerTestUser() {
	payload := registerRequest{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "testpassword",
	}

	body, err := json.Marshal(payload)
	suite.NoError(err)

	req, err := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
	suite.NoError(err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)
	suite.Equal(http.StatusCreated, w.Code)

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)
	
	// Check if response contains expected fields
	suite.NotNil(response["id"], "Expected user ID to be present in response")
	
	// Safe type assertion with check
	userIDFloat, ok := response["id"].(float64)
	suite.True(ok, "Expected user ID to be a number")
	suite.userID = uint(userIDFloat)
}

func (suite *AuthTestSuite) loginTestUser() {
	payload := loginRequest{
		Email:    "test@example.com",
		Password: "testpassword",
	}

	body, err := json.Marshal(payload)
	suite.NoError(err)

	req, err := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	suite.NoError(err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)
	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	suite.accessToken = response["access_token"].(string)
	suite.refreshToken = response["refresh_token"].(string)
}