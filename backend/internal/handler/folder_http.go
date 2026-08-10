package handler

import (
	"net/http"

	"backend/internal/noteapp"
	"backend/internal/response"

	"github.com/gin-gonic/gin"
)

func (handler *ContentHandler) ListFolders(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	folders, err := handler.service.ListFolders(c.Request.Context(), userID)
	if err != nil {
		writeUseCaseError(c, err)
		return
	}
	c.JSON(http.StatusOK, folders)
}

func (handler *ContentHandler) CreateFolder(c *gin.Context) {
	var input noteapp.CreateFolderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	folder, err := handler.service.CreateFolder(c.Request.Context(), userID, input)
	if err != nil {
		writeUseCaseError(c, err)
		return
	}
	c.JSON(http.StatusCreated, folder)
}

func (handler *ContentHandler) UpdateFolder(c *gin.Context) {
	folderID, ok := pathID(c, "folder")
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var input noteapp.UpdateFolderInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	folder, err := handler.service.UpdateFolder(c.Request.Context(), userID, folderID, input)
	if err != nil {
		writeUseCaseError(c, err)
		return
	}
	c.JSON(http.StatusOK, folder)
}

func (handler *ContentHandler) DeleteFolder(c *gin.Context) {
	folderID, ok := pathID(c, "folder")
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	if err := handler.service.DeleteFolder(c.Request.Context(), userID, folderID); err != nil {
		writeUseCaseError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
