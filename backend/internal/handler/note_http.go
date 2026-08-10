package handler

import (
	"net/http"
	"strconv"

	"backend/internal/noteapp"
	"backend/internal/response"

	"github.com/gin-gonic/gin"
)

func (handler *ContentHandler) ListNotes(c *gin.Context) {
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil {
		limit = 50
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil {
		offset = 0
	}
	page, err := handler.service.ListNotes(c.Request.Context(), userID, noteapp.ListNotesFilter{
		FolderID: c.Query("folder_id"), Query: c.Query("q"), Status: c.Query("status"),
		Limit: limit, Offset: offset,
	})
	if err != nil {
		writeUseCaseError(c, err)
		return
	}
	c.JSON(http.StatusOK, page)
}

func (handler *ContentHandler) CreateNote(c *gin.Context) {
	var input noteapp.CreateNoteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	note, err := handler.service.CreateNote(c.Request.Context(), userID, input)
	if err != nil {
		writeUseCaseError(c, err)
		return
	}
	c.JSON(http.StatusCreated, note)
}

func (handler *ContentHandler) GetNote(c *gin.Context) {
	noteID, ok := pathID(c, "note")
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	note, err := handler.service.GetNote(c.Request.Context(), userID, noteID)
	if err != nil {
		writeUseCaseError(c, err)
		return
	}
	c.JSON(http.StatusOK, note)
}

func (handler *ContentHandler) ListNoteRevisions(c *gin.Context) {
	noteID, ok := pathID(c, "note")
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	revisions, err := handler.service.ListNoteRevisions(c.Request.Context(), userID, noteID)
	if err != nil {
		writeUseCaseError(c, err)
		return
	}
	c.JSON(http.StatusOK, revisions)
}

func (handler *ContentHandler) UpdateNote(c *gin.Context) {
	noteID, ok := pathID(c, "note")
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	var input noteapp.UpdateNoteInput
	if err := c.ShouldBindJSON(&input); err != nil {
		response.Error(c, http.StatusBadRequest, "VALIDATION_ERROR", err.Error())
		return
	}
	note, err := handler.service.UpdateNote(c.Request.Context(), userID, noteID, input)
	if err != nil {
		writeUseCaseError(c, err)
		return
	}
	c.JSON(http.StatusOK, note)
}

func (handler *ContentHandler) DeleteNote(c *gin.Context) {
	noteID, ok := pathID(c, "note")
	if !ok {
		return
	}
	userID, ok := currentUserID(c)
	if !ok {
		return
	}
	if err := handler.service.DeleteNote(c.Request.Context(), userID, noteID); err != nil {
		writeUseCaseError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}
