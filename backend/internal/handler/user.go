// internal/handler/user.go
package handler

import (
	"net/http"

	"backend/internal/database"
	"backend/internal/model"
	"backend/internal/response"

	"github.com/gin-gonic/gin"
)

func GetCurrentUser(c *gin.Context) {
	// ASSUME 'userID' is written into context in JWTMiddleware
	userIDInterface, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "user not authenticated")
		return
	}

	userID, ok := userIDInterface.(uint)
	if !ok {
		response.Error(c, http.StatusInternalServerError, "INTERNAL", "invalid user ID format")
		return
	}

	var user model.User

	// get user by id
	if err := database.DB.First(&user, userID).Error; err != nil {
		response.Error(c, http.StatusNotFound, "NOT_FOUND", "user not found")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	})
}
