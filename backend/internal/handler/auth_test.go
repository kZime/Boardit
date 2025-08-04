// internal/handler/auth_test.go

package handler

import (
	"testing"
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"

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

func TestRegister(t *testing.T) {
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

func TestLogin(t *testing.T) {

}

func TestRefresh(t *testing.T) {

}