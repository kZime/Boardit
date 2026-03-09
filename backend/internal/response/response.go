// Package response provides unified API response helpers aligned with OpenAPI Error schema.
package response

import (
	"github.com/gin-gonic/gin"
)

// Error sends a JSON error response with OpenAPI shape: { "error": code, "message": msg }.
// errorCode should be one of: UNAUTHORIZED, FORBIDDEN, NOT_FOUND, VALIDATION_ERROR, VERSION_CONFLICT, RATE_LIMITED, INTERNAL.
func Error(c *gin.Context, statusCode int, errorCode, message string) {
	c.JSON(statusCode, gin.H{
		"error":   errorCode,
		"message": message,
	})
}
