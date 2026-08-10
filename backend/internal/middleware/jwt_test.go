package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/suite"
)

type JWTMiddlewareTestSuite struct {
	suite.Suite
	router *gin.Engine
	secret string
}

func TestJWTMiddlewareSuite(t *testing.T) {
	suite.Run(t, new(JWTMiddlewareTestSuite))
}

func (suite *JWTMiddlewareTestSuite) SetupSuite() {
	// Set test environment
	suite.secret = "test-secret-key"
	os.Setenv("JWT_SECRET", suite.secret)

	// Set up router
	gin.SetMode(gin.TestMode)
	suite.router = gin.New()

	// Add a test endpoint that uses JWT middleware
	suite.router.GET("/test", JWTMiddleware(), func(c *gin.Context) {
		userID, exists := c.Get("userID")
		if !exists {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "userID not found in context"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"userID": userID})
	})
}

func (suite *JWTMiddlewareTestSuite) TestValidToken() {
	// Create a valid JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": float64(123),
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(suite.secret))
	suite.NoError(err)

	// Make request with valid token
	req, err := http.NewRequest("GET", "/test", nil)
	suite.NoError(err)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Check response
	suite.Equal(http.StatusOK, w.Code)

	var response map[string]interface{}
	err = suite.parseJSON(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal(float64(123), response["userID"])
}

func (suite *JWTMiddlewareTestSuite) TestMissingAuthorizationHeader() {
	req, err := http.NewRequest("GET", "/test", nil)
	suite.NoError(err)
	// No Authorization header

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Check response
	suite.Equal(http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err = suite.parseJSON(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("authorization header required", response["error"])
}

func (suite *JWTMiddlewareTestSuite) TestInvalidAuthorizationFormat() {
	testCases := []string{
		"InvalidFormat",  // No space, no Bearer
		"Basic token123", // Wrong scheme
		"BearerToken",    // No space between Bearer and token
		"TokenOnly",      // No Bearer prefix
		"Bearer",         // Missing token part (len(parts) != 2)
	}

	for _, authHeader := range testCases {
		suite.Run("Format: "+authHeader, func() {
			req, err := http.NewRequest("GET", "/test", nil)
			suite.NoError(err)
			req.Header.Set("Authorization", authHeader)

			w := httptest.NewRecorder()
			suite.router.ServeHTTP(w, req)

			// Check response
			suite.Equal(http.StatusUnauthorized, w.Code)

			var response map[string]interface{}
			err = suite.parseJSON(w.Body.Bytes(), &response)
			suite.NoError(err)
			suite.Equal("invalid authorization header format", response["error"],
				"Expected format error for header: %s", authHeader)
		})
	}
}

func (suite *JWTMiddlewareTestSuite) TestInvalidToken() {
	testCases := []string{
		"invalid.token.here",
		"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.invalid_signature",
		"not.a.valid.jwt",
	}

	for _, token := range testCases {
		req, err := http.NewRequest("GET", "/test", nil)
		suite.NoError(err)
		req.Header.Set("Authorization", "Bearer "+token)

		w := httptest.NewRecorder()
		suite.router.ServeHTTP(w, req)

		// Check response
		suite.Equal(http.StatusUnauthorized, w.Code)

		var response map[string]interface{}
		err = suite.parseJSON(w.Body.Bytes(), &response)
		suite.NoError(err)
		suite.Equal("invalid or expired token", response["error"])
	}
}

func (suite *JWTMiddlewareTestSuite) TestExpiredToken() {
	// Create an expired JWT token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": float64(123),
		"exp": time.Now().Add(-time.Hour).Unix(), // Expired 1 hour ago
	})
	tokenString, err := token.SignedString([]byte(suite.secret))
	suite.NoError(err)

	req, err := http.NewRequest("GET", "/test", nil)
	suite.NoError(err)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Check response
	suite.Equal(http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err = suite.parseJSON(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("invalid or expired token", response["error"])
}

func (suite *JWTMiddlewareTestSuite) TestWrongSigningMethod() {
	// Create a token with wrong signing method
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"sub": float64(123),
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(suite.secret))
	suite.NoError(err)

	req, err := http.NewRequest("GET", "/test", nil)
	suite.NoError(err)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Check response
	suite.Equal(http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err = suite.parseJSON(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("invalid or expired token", response["error"])
}

func (suite *JWTMiddlewareTestSuite) TestMissingSubClaim() {
	// Create a token without sub claim
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"exp": time.Now().Add(time.Hour).Unix(),
		// Missing "sub" claim
	})
	tokenString, err := token.SignedString([]byte(suite.secret))
	suite.NoError(err)

	req, err := http.NewRequest("GET", "/test", nil)
	suite.NoError(err)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Check response
	suite.Equal(http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err = suite.parseJSON(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("invalid sub claim", response["error"])
}

func (suite *JWTMiddlewareTestSuite) TestInvalidSubClaimType() {
	// Create a token with sub claim as string instead of number
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"sub": "123", // String instead of number
		"exp": time.Now().Add(time.Hour).Unix(),
	})
	tokenString, err := token.SignedString([]byte(suite.secret))
	suite.NoError(err)

	req, err := http.NewRequest("GET", "/test", nil)
	suite.NoError(err)
	req.Header.Set("Authorization", "Bearer "+tokenString)

	w := httptest.NewRecorder()
	suite.router.ServeHTTP(w, req)

	// Check response
	suite.Equal(http.StatusUnauthorized, w.Code)

	var response map[string]interface{}
	err = suite.parseJSON(w.Body.Bytes(), &response)
	suite.NoError(err)
	suite.Equal("invalid sub claim", response["error"])
}

// Helper method to parse JSON response
func (suite *JWTMiddlewareTestSuite) parseJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
