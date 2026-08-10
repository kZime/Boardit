package database

import (
	"github.com/stretchr/testify/assert"
	"os"
	"testing"

	"backend/internal/testutils"
)

func TestMain(m *testing.M) {
	// Load environment variables for testing
	testutils.LoadTestEnv()

	// Run test
	code := m.Run()

	// Exit test
	os.Exit(code)
}

func TestInit(t *testing.T) {
	// Run init function
	err := Init()

	// Check if there was an error
	assert.NoError(t, err, "Init should not return an error")

	// Check if DB is initialized
	assert.NotNil(t, DB, "DB should not be nil after initialization")
}
