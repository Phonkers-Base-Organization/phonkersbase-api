package handlers

import (
	"net/http"

	"github.com/PhonkersBase/base-api2/internal/domain"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func (h *Handler) GetSources(c *gin.Context) {
	sources, err := h.sources.GetAll(c.Request.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to get sources")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, sources)
}

func (h *Handler) CreateSource(c *gin.Context) {
	var input domain.UpsertSourceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	source, err := h.sources.Create(c.Request.Context(), input.Name)
	if err != nil {
		log.Error().Err(err).Msg("failed to create source")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusCreated, source)
}
