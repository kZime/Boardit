// internal/testutils/testutils.go
package testutils

import (
	"os"

	"github.com/joho/godotenv"
)

func LoadTestEnv() {
	// Preserve env if already set (e.g. by CI or local export)
	dsn := os.Getenv("DATABASE_DSN")
	jwt := os.Getenv("JWT_SECRET")

	// Try multiple paths: from backend/ or from package dir (internal/handler, etc.)
	for _, path := range []string{".env.test", "../../.env.test", "../../../.env.test"} {
		if err := godotenv.Load(path); err == nil {
			break
		}
	}

	if dsn != "" {
		os.Setenv("DATABASE_DSN", dsn)
	}
	if jwt != "" {
		os.Setenv("JWT_SECRET", jwt)
	}
}
