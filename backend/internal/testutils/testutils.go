// internal/testutils/testutils.go
package testutils

import (
    "github.com/joho/godotenv"
)

func LoadTestEnv() {
    err := godotenv.Load("../../.env.test")
    if err != nil {
        panic("Failed to load .env.test file")
    }
}