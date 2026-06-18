package handlers

import (
	"net/http"

	"github.com/PhonkersBase/base-api2/internal/domain"
	"github.com/PhonkersBase/base-api2/internal/repository"
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

	source, err := h.sources.Create(c.Request.Context(), input)
	if err != nil {
		log.Error().Err(err).Msg("failed to create source")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusCreated, source)
}

func (h *Handler) UpdateSource(c *gin.Context) {
	id := c.Param("id")
	var input domain.UpsertSourceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	source, err := h.sources.Update(c.Request.Context(), id, input)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "source not found"})
			return
		}
		log.Error().Err(err).Msg("failed to update source")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, source)
}

func (h *Handler) DeleteSource(c *gin.Context) {
	id := c.Param("id")
	if err := h.sources.Delete(c.Request.Context(), id); err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "source not found"})
			return
		}
		log.Error().Err(err).Msg("failed to delete source")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusNoContent)
}
