package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLoadValidatesRequiredSecurityConfiguration(t *testing.T) {
	t.Setenv("DATABASE_DSN", ":memory:")
	t.Setenv("JWT_SECRET", "test-secret-for-jwt-min-32-chars!!")
	t.Setenv("CORS_ORIGINS", "https://one.example, https://two.example")
	t.Setenv("TRUSTED_PROXIES", "127.0.0.1")

	configuration, err := Load()
	require.NoError(t, err)
	require.Equal(t, ":8080", configuration.ListenAddress)
	require.Equal(t, []string{"https://one.example", "https://two.example"}, configuration.CORSOrigins)
	require.Equal(t, []string{"127.0.0.1"}, configuration.TrustedProxies)
}

func TestLoadRejectsShortJWTSecret(t *testing.T) {
	t.Setenv("DATABASE_DSN", ":memory:")
	t.Setenv("JWT_SECRET", "short")

	_, err := Load()
	require.EqualError(t, err, "JWT_SECRET must be at least 32 characters")
}
