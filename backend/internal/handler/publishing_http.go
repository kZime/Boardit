package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

func (handler *ContentHandler) ListPublicNotes(c *gin.Context) {
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "50"))
	if err != nil {
		limit = 50
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil {
		offset = 0
	}
	page, err := handler.service.ListPublicNotes(c.Request.Context(), limit, offset)
	if err != nil {
		writeUseCaseError(c, err)
		return
	}
	c.JSON(http.StatusOK, page)
}

func (handler *ContentHandler) GetPublicNote(c *gin.Context) {
	note, err := handler.service.GetPublicNote(c.Request.Context(), c.Param("username"), c.Param("slug"))
	if err != nil {
		writeUseCaseError(c, err)
		return
	}
	c.JSON(http.StatusOK, note)
}
