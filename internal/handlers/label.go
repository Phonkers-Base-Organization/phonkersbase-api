package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetLabels(c *gin.Context) {
	labels, err := h.labels.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, errJSON(err.Error()))
		return
	}
	c.JSON(http.StatusOK, labels)
}
