package handlers

import (
	"net/http"
	"github.com/rs/zerolog/log"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetLabels(c *gin.Context) {
	labels, err := h.labels.GetAll(c.Request.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to get labels")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, labels)
}
