package handlers

import (
	"net/http"
	"strconv"

	"github.com/PhonkersBase/base-api2/internal/domain"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetHistory(c *gin.Context) {
	params := domain.ListChangesParams{
		EntityID: c.Query("entityId"),
		Search:   c.Query("search"),
	}

	if v := c.Query("entityType"); v != "" {
		if !domain.ValidEntityType(v) {
			c.JSON(http.StatusBadRequest, gin.H{"errors": gin.H{"entityType": "invalid value"}})
			return
		}
		params.EntityType = v
	}

	if v := c.Query("action"); v != "" {
		if !domain.ValidChangeAction(v) {
			c.JSON(http.StatusBadRequest, gin.H{"errors": gin.H{"action": "invalid value"}})
			return
		}
		params.Action = v
	}

	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			params.Offset = n
		}
	}
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 500 {
			c.JSON(http.StatusBadRequest, gin.H{"errors": gin.H{"limit": "must be between 1 and 500"}})
			return
		}
		params.Limit = n
	}

	result, err := h.changeHistory.List(c.Request.Context(), params)
	if err != nil {
		internalErr(c, err, "failed to get change history")
		return
	}
	c.JSON(http.StatusOK, result)
}
