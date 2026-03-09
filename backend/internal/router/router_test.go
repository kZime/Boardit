package router

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/database"
	"backend/internal/testutils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/suite"
)

type RouterTestSuite struct {
	suite.Suite
	router *gin.Engine
}

func TestRouterSuite(t *testing.T) {
	suite.Run(t, new(RouterTestSuite))
}

func (suite *RouterTestSuite) SetupSuite() {
	// Load test environment variables
	testutils.LoadTestEnv()

	// Initialize the database
	if err := database.Init(); err != nil {
		panic("Failed to initialize the database: " + err.Error())
	}

	// Set up router
	gin.SetMode(gin.TestMode)
	suite.router = Setup()
}

func (suite *RouterTestSuite) TearDownSuite() {
	suite.Require().NoError(database.TruncateAllTables())
}

func (suite *RouterTestSuite) SetupTest() {
	suite.Require().NoError(database.TruncateAllTables())
}

func (suite *RouterTestSuite) TestAuthRoutes() {
	testCases := []struct {
		name     string
		method   string
		path     string
		expected int
		body     interface{}
	}{
		{
			name:     "Register endpoint exists",
			method:   "POST",
			path:     "/api/auth/register",
			expected: http.StatusBadRequest, // Will fail validation but proves route exists
		},
		{
			name:     "Login endpoint exists",
			method:   "POST",
			path:     "/api/auth/login",
			expected: http.StatusBadRequest, // Will fail validation but proves route exists
		},
		{
			name:     "Refresh endpoint exists",
			method:   "POST",
			path:     "/api/auth/refresh",
			expected: http.StatusBadRequest, // Will fail validation but proves route exists
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			var body []byte
			var err error
			if tc.body != nil {
				body, err = json.Marshal(tc.body)
				suite.NoError(err)
			}

			req, err := http.NewRequest(tc.method, tc.path, bytes.NewBuffer(body))
			suite.NoError(err)
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			// Route should exist (not 404)
			suite.NotEqual(http.StatusNotFound, w.Code, "Route should exist")
		})
	}
}

func (suite *RouterTestSuite) TestUserRouteWithAuth() {
	// Test that /api/user requires authentication
	req, err := http.NewRequest("GET", "/api/user", nil)
	suite.NoError(err)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Should return 401 Unauthorized because no token provided
	suite.Equal(http.StatusUnauthorized, w.Code)
}

func (suite *RouterTestSuite) TestCORSHeaders() {
	// Test CORS headers are set correctly
	req, err := http.NewRequest("OPTIONS", "/api/auth/register", nil)
	suite.NoError(err)
	req.Header.Set("Origin", "http://localhost:5173")
	req.Header.Set("Access-Control-Request-Method", "POST")
	req.Header.Set("Access-Control-Request-Headers", "Content-Type")

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Check CORS headers
	suite.Equal("http://localhost:5173", w.Header().Get("Access-Control-Allow-Origin"))
	suite.Contains(w.Header().Get("Access-Control-Allow-Methods"), "POST")
	suite.Contains(w.Header().Get("Access-Control-Allow-Headers"), "Content-Type")
}

func (suite *RouterTestSuite) TestNonExistentRoute() {
	// Test that non-existent routes return 404
	req, err := http.NewRequest("GET", "/api/nonexistent", nil)
	suite.NoError(err)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Should return 404 Not Found
	suite.Equal(http.StatusNotFound, w.Code)
}

func (suite *RouterTestSuite) TestMethodNotAllowed() {
	// Test that wrong HTTP method returns 404 (Gin's default behavior)
	req, err := http.NewRequest("GET", "/api/auth/register", nil)
	suite.NoError(err)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Should return 404 because GET is not allowed on POST route
	suite.Equal(http.StatusNotFound, w.Code)
}

func (suite *RouterTestSuite) TestRouteStructure() {
	// Test that all expected route groups exist by making actual requests
	// Note: We can't use OPTIONS because Gin doesn't handle them by default
	
	// Test register route
	suite.Run("Route exists: /api/auth/register", func() {
		payload := map[string]interface{}{
			"username": "testuser1",
			"email":    "test1@example.com",
			"password": "password",
		}
		body, err := json.Marshal(payload)
		suite.NoError(err)

		req, err := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
		suite.NoError(err)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		// Route should exist (not 404)
		suite.NotEqual(http.StatusNotFound, w.Code, "Route should exist: /api/auth/register")
	})

	// Test login route
	suite.Run("Route exists: /api/auth/login", func() {
		payload := map[string]interface{}{
			"email":    "test2@example.com",
			"password": "password",
		}
		body, err := json.Marshal(payload)
		suite.NoError(err)

		req, err := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
		suite.NoError(err)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		// Route should exist (not 404)
		suite.NotEqual(http.StatusNotFound, w.Code, "Route should exist: /api/auth/login")
	})

	// Test refresh route
	suite.Run("Route exists: /api/auth/refresh", func() {
		payload := map[string]interface{}{
			"refresh_token": "invalid_token",
		}
		body, err := json.Marshal(payload)
		suite.NoError(err)

		req, err := http.NewRequest("POST", "/api/auth/refresh", bytes.NewBuffer(body))
		suite.NoError(err)
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		// Route should exist (not 404)
		suite.NotEqual(http.StatusNotFound, w.Code, "Route should exist: /api/auth/refresh")
	})

	// Test user route
	suite.Run("Route exists: /api/user", func() {
		req, err := http.NewRequest("GET", "/api/user", nil)
		suite.NoError(err)

		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		// Route should exist (not 404) - will return 401 due to missing auth
		suite.NotEqual(http.StatusNotFound, w.Code, "Route should exist: /api/user")
	})
}

func (suite *RouterTestSuite) TestContentTypeHeaders() {
	// Test that Content-Type headers are handled correctly
	testCases := []struct {
		name           string
		contentType    string
		expectedStatus int
	}{
		{
			name:           "Valid JSON Content-Type",
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest, // Will fail validation but proves route accepts JSON
		},
		{
			name:           "Invalid Content-Type",
			contentType:    "text/plain",
			expectedStatus: http.StatusBadRequest, // Should still process but fail validation
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			payload := map[string]interface{}{
				"username": "test",
				"email":    "test@example.com",
				"password": "password",
			}
			body, err := json.Marshal(payload)
			suite.NoError(err)

			req, err := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
			suite.NoError(err)
			req.Header.Set("Content-Type", tc.contentType)

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			// Route should process the request (not 404 or 415)
			suite.NotEqual(http.StatusNotFound, w.Code)
			suite.NotEqual(http.StatusUnsupportedMediaType, w.Code)
		})
	}
}
