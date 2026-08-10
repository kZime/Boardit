package handler

import (
	"net/http"

	"backend/internal/noteapp"
	"backend/internal/response"

	"github.com/gin-gonic/gin"
)

func (handler *ContentHandler) ReorderTree(c *gin.Context) {
	var input noteapp.ReorderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", "invalid payload")
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	if err := handler.service.Reorder(c.Request.Context(), userID, input); err != nil {
		writeUseCaseError(c, err)
		return
	}
	c.Status(http.StatusOK)
}
