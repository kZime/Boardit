package config

import (
	"fmt"
	"os"
	"strings"
)

type Config struct {
	DatabaseDSN    string
	JWTSecret      string
	GinMode        string
	ListenAddress  string
	CORSOrigins    []string
	TrustedProxies []string
}

func Load() (Config, error) {
	configuration := Config{
		DatabaseDSN:    os.Getenv("DATABASE_DSN"),
		JWTSecret:      os.Getenv("JWT_SECRET"),
		GinMode:        os.Getenv("GIN_MODE"),
		ListenAddress:  envOrDefault("LISTEN_ADDRESS", ":8080"),
		CORSOrigins:    splitList(envOrDefault("CORS_ORIGINS", "http://localhost:5173")),
		TrustedProxies: splitList(os.Getenv("TRUSTED_PROXIES")),
	}
	if configuration.DatabaseDSN == "" {
		return Config{}, fmt.Errorf("DATABASE_DSN is required")
	}
	if len(configuration.JWTSecret) < 32 {
		return Config{}, fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}
	return configuration, nil
}

func envOrDefault(key, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(key)); value != "" {
		return value
	}
	return fallback
}

func splitList(value string) []string {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parts := strings.Split(value, ",")
	result := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}
