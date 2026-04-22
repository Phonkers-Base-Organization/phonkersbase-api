package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) ForceSync(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "sync triggered",
	})
}
