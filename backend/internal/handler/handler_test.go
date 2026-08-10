// internal/handler/auth_test.go

package handler

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"backend/internal/database"
	"backend/internal/model"
	"backend/internal/testutils"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

// AuthTestSuite holds shared state for auth tests
type AuthTestSuite struct {
	suite.Suite
	router         *gin.Engine
	authentication *AuthHandler
	userID         uint
	accessToken    string
	refreshToken   string
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
	suite.authentication = NewAuthHandler(os.Getenv("JWT_SECRET"))
	suite.router.POST("/api/auth/register", Register)
	suite.router.POST("/api/auth/login", suite.authentication.Login)
	suite.router.POST("/api/auth/refresh", suite.authentication.Refresh)
	suite.router.POST("/api/auth/logout", suite.authentication.Logout)
	// Note: For testing GetCurrentUser, we'll handle JWT middleware in the test itself
	suite.router.GET("/api/user", GetCurrentUser)
}

func (suite *AuthTestSuite) TearDownSuite() {
	suite.Require().NoError(database.TruncateAllTables())
}

func (suite *AuthTestSuite) SetupTest() {
	// Clean up before each test to ensure isolation (works with both Postgres and SQLite)
	suite.Require().NoError(database.TruncateAllTables())
	// Reset test state
	suite.userID = 0
	suite.accessToken = ""
	suite.refreshToken = ""
}

// TestRegisterSuccess tests user registration (uses unique email to avoid cross-test leakage in CI)
func (suite *AuthTestSuite) TestRegisterSuccess() {
	uniqueEmail := fmt.Sprintf("test-reg-%d@example.com", time.Now().UnixNano())
	payload := registerRequest{
		Username: "testuser",
		Email:    uniqueEmail,
		Password: "testpassword",
	}

	body, err := json.Marshal(payload)
	suite.NoError(err, "Failed to marshal payload")

	req, err := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
	suite.NoError(err, "Failed to create request")
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusCreated, w.Code, "Expected status code 201 Created")

	var response map[string]interface{}
	err = json.Unmarshal(w.Body.Bytes(), &response)
	suite.NoError(err)

	suite.Equal("testuser", response["username"], "Expected username to be 'testuser'")
	suite.Equal(uniqueEmail, response["email"], "Expected email to match")
	suite.NotEmpty(response["id"], "Expected user ID to be present")

	suite.userID = uint(response["id"].(float64))
}

func (suite *AuthTestSuite) TestRegisterRejectsUnsafeUsernamesAndShortPasswords() {
	testCases := []struct {
		name     string
		username string
		password string
	}{
		{name: "whitespace username", username: "   ", password: "testpassword"},
		{name: "URL-unsafe username", username: "bad/name", password: "testpassword"},
		{name: "username too short", username: "ab", password: "testpassword"},
		{name: "password too short", username: "valid-user", password: "1234567"},
	}

	for index, testCase := range testCases {
		suite.Run(testCase.name, func() {
			body, err := json.Marshal(registerRequest{
				Username: testCase.username,
				Email:    fmt.Sprintf("invalid-%d@example.com", index),
				Password: testCase.password,
			})
			suite.NoError(err)
			req, err := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(body))
			suite.NoError(err)
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			suite.router.ServeHTTP(w, req)

			suite.Equal(http.StatusBadRequest, w.Code)
		})
	}
}

func (suite *AuthTestSuite) TestRegisterRejectsDuplicateUsername() {
	suite.registerTestUser()
	body, err := json.Marshal(registerRequest{
		Username: "testuser",
		Email:    "another@example.com",
		Password: "testpassword",
	})
	suite.NoError(err)
	req, err := http.NewRequest(http.MethodPost, "/api/auth/register", bytes.NewBuffer(body))
	suite.NoError(err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusBadRequest, w.Code)
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
	suite.Equal(float64(accessTokenTTL.Seconds()), response["expires_in"])

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
	suite.Equal(float64(accessTokenTTL.Seconds()), refreshResponse["expires_in"])

	// Check the new tokens are different from the old ones
	suite.NotEqual(suite.refreshToken, refreshResponse["refresh_token"], "Expected new refresh token")
	suite.NotEqual(suite.accessToken, refreshResponse["access_token"], "Expected new access token")

	// Rotation is single-use: replaying the old token must fail.
	replayW := httptest.NewRecorder()
	replayReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewBuffer(refreshBody))
	replayReq.Header.Set("Content-Type", "application/json")
	suite.router.ServeHTTP(replayW, replayReq)
	suite.Equal(http.StatusUnauthorized, replayW.Code)

	// Reuse detection revokes the replacement token as part of the same family.
	replacementBody, err := json.Marshal(refreshRequest{RefreshToken: refreshResponse["refresh_token"].(string)})
	suite.NoError(err)
	replacementReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewBuffer(replacementBody))
	replacementReq.Header.Set("Content-Type", "application/json")
	replacementW := httptest.NewRecorder()
	suite.router.ServeHTTP(replacementW, replacementReq)
	suite.Equal(http.StatusUnauthorized, replacementW.Code)

	var activeSessions int64
	suite.NoError(database.DB.Model(&model.RefreshSession{}).
		Where("user_id = ? AND revoked_at IS NULL", suite.userID).
		Count(&activeSessions).Error)
	suite.Zero(activeSessions)
}

func (suite *AuthTestSuite) TestConcurrentReplayRevokesSessionCreatedByDescendantRefresh() {
	if database.DB.Dialector.Name() != "postgres" {
		suite.T().Skip("PostgreSQL row-lock regression test")
	}
	suite.registerTestUser()
	suite.loginTestUser()
	rootToken := suite.refreshToken
	_, rootTokenID, ok := parseRefreshToken(suite.authentication.jwtSecret, rootToken)
	suite.Require().True(ok)

	rootBody, err := json.Marshal(refreshRequest{RefreshToken: rootToken})
	suite.NoError(err)
	rootReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader(rootBody))
	rootReq.Header.Set("Content-Type", "application/json")
	rootW := httptest.NewRecorder()
	suite.router.ServeHTTP(rootW, rootReq)
	suite.Require().Equal(http.StatusOK, rootW.Code)
	var rotation map[string]any
	suite.Require().NoError(json.Unmarshal(rootW.Body.Bytes(), &rotation))
	childToken := rotation["refresh_token"].(string)
	_, childTokenID, ok := parseRefreshToken(suite.authentication.jwtSecret, childToken)
	suite.Require().True(ok)

	triggerSQL := fmt.Sprintf(`
		CREATE OR REPLACE FUNCTION delay_test_refresh_rotation() RETURNS trigger AS $$
		BEGIN
			IF OLD.token_id = '%s' AND OLD.revoked_at IS NULL AND NEW.revoked_at IS NOT NULL THEN
				PERFORM pg_sleep(0.4);
			END IF;
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql;
		CREATE TRIGGER delay_test_refresh_rotation_trigger
		BEFORE UPDATE ON refresh_sessions
		FOR EACH ROW EXECUTE FUNCTION delay_test_refresh_rotation();`, childTokenID)
	suite.Require().NoError(database.DB.Exec(triggerSQL).Error)
	defer func() {
		suite.NoError(database.DB.Exec("DROP TRIGGER IF EXISTS delay_test_refresh_rotation_trigger ON refresh_sessions").Error)
		suite.NoError(database.DB.Exec("DROP FUNCTION IF EXISTS delay_test_refresh_rotation()").Error)
	}()

	childBody, err := json.Marshal(refreshRequest{RefreshToken: childToken})
	suite.NoError(err)
	childStatus := make(chan int, 1)
	go func() {
		req := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader(childBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)
		childStatus <- w.Code
	}()

	// Do not rely on scheduler timing. The trigger keeps the descendant
	// rotation open while a separate transaction observes that it owns the
	// shared family-root lock.
	lockDeadline := time.Now().Add(3 * time.Second)
	for {
		probe := database.DB.Begin()
		var rootID uint
		lockErr := probe.Raw(
			"SELECT id FROM refresh_sessions WHERE token_id = ? FOR UPDATE NOWAIT",
			rootTokenID,
		).Scan(&rootID).Error
		rollbackErr := probe.Rollback().Error
		if lockErr != nil && strings.Contains(lockErr.Error(), "55P03") {
			break
		}
		suite.Require().NoError(lockErr)
		suite.Require().NoError(rollbackErr)
		if time.Now().After(lockDeadline) {
			suite.T().Fatal("descendant refresh did not acquire the family-root lock before timeout")
		}
		time.Sleep(10 * time.Millisecond)
	}
	replayReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewReader(rootBody))
	replayReq.Header.Set("Content-Type", "application/json")
	replayW := httptest.NewRecorder()
	suite.router.ServeHTTP(replayW, replayReq)

	suite.Equal(http.StatusOK, <-childStatus)
	suite.Equal(http.StatusUnauthorized, replayW.Code)
	var activeSessions int64
	suite.NoError(database.DB.Model(&model.RefreshSession{}).
		Where("user_id = ? AND revoked_at IS NULL", suite.userID).
		Count(&activeSessions).Error)
	suite.Zero(activeSessions)
}

func (suite *AuthTestSuite) TestLogoutRevokesRefreshSession() {
	suite.registerTestUser()
	suite.loginTestUser()
	body, err := json.Marshal(refreshRequest{RefreshToken: suite.refreshToken})
	suite.NoError(err)

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/logout", bytes.NewBuffer(body))
	logoutReq.Header.Set("Content-Type", "application/json")
	logoutW := httptest.NewRecorder()
	suite.router.ServeHTTP(logoutW, logoutReq)
	suite.Equal(http.StatusNoContent, logoutW.Code)

	refreshReq := httptest.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewBuffer(body))
	refreshReq.Header.Set("Content-Type", "application/json")
	refreshW := httptest.NewRecorder()
	suite.router.ServeHTTP(refreshW, refreshReq)
	suite.Equal(http.StatusUnauthorized, refreshW.Code)
}

func (suite *AuthTestSuite) TestRefreshRejectsAccessToken() {
	suite.registerTestUser()
	suite.loginTestUser()

	body, err := json.Marshal(refreshRequest{RefreshToken: suite.accessToken})
	suite.NoError(err)
	req, err := http.NewRequest(http.MethodPost, "/api/auth/refresh", bytes.NewBuffer(body))
	suite.NoError(err)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	suite.router.ServeHTTP(w, req)

	suite.Equal(http.StatusUnauthorized, w.Code)
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
