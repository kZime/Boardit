package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

// JWTMiddleware: validate Authorization: Bearer <token>
// if success, set userID in context, otherwise return 401
func JWTMiddleware() gin.HandlerFunc {
	return JWTMiddlewareWithSecret(os.Getenv("JWT_SECRET"))
}

func JWTMiddlewareWithSecret(secretValue string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "authorization header required"})
			return
		}

		// Validate Bearer token format
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid authorization header format"})
			return
		}
		tokenString := parts[1]

		// Parse and verify signature
		secret := []byte(secretValue)
		token, err := jwt.Parse(tokenString, func(t *jwt.Token) (interface{}, error) {
			// Strongly validate signature algorithm
			if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return secret, nil
		})
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid or expired token"})
			return
		}

		// Get sub from claims (which is userID)
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token claims"})
			return
		}
		tokenType, ok := claims["typ"].(string)
		if !ok || tokenType != "access" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid token type"})
			return
		}
		sub, ok := claims["sub"].(float64) // jwt library will parse number to float64
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "invalid sub claim"})
			return
		}
		userID := uint(sub)

		// Set userID in context, so that subsequent handlers can get it by c.Get("userID")
		c.Set("userID", userID)

		c.Next()
	}
}
