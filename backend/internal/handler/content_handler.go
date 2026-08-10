package handler

import (
	"errors"
	"net/http"
	"strconv"

	"backend/internal/noteapp"
	"backend/internal/response"

	"github.com/gin-gonic/gin"
)

type ContentHandler struct {
	service *noteapp.Service
}

func NewContentHandler(service *noteapp.Service) *ContentHandler {
	return &ContentHandler{service: service}
}

func currentUserID(c *gin.Context) (uint, bool) {
	value, exists := c.Get("userID")
	if !exists {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user ID")
		return 0, false
	}
	userID, ok := value.(uint)
	if !ok {
		response.Error(c, http.StatusUnauthorized, "UNAUTHORIZED", "invalid user ID")
		return 0, false
	}
	return userID, true
}

func pathID(c *gin.Context, resource string) (uint, bool) {
	parsed, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil || parsed == 0 {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid "+resource+" ID")
		return 0, false
	}
	return uint(parsed), true
}

func writeUseCaseError(c *gin.Context, err error) {
	var conflict *noteapp.ConflictError
	if errors.As(err, &conflict) {
		c.JSON(http.StatusConflict, gin.H{
			"error": "VERSION_CONFLICT", "message": conflict.Error(),
			"server_updated_at": conflict.ServerUpdatedAt,
			"server_snapshot":   conflict.ServerSnapshot,
		})
		return
	}
	var useCaseErr *noteapp.UseCaseError
	if errors.As(err, &useCaseErr) {
		status := http.StatusInternalServerError
		if useCaseErr.Kind == noteapp.ErrorInvalid {
			status = http.StatusBadRequest
		} else if useCaseErr.Kind == noteapp.ErrorNotFound {
			status = http.StatusNotFound
		}
		response.Error(c, status, string(useCaseErr.Kind), useCaseErr.Message)
		return
	}
	response.Error(c, http.StatusInternalServerError, "INTERNAL", "internal server error")
}
