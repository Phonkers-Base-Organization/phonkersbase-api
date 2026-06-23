package handlers

import (
	"net/http"

	"github.com/PhonkersBase/base-api2/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func (h *Handler) GetSuggestions(c *gin.Context) {
	suggestions, err := h.suggestions.GetAll(c.Request.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to get suggestions")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, gin.H{"suggestions": suggestions})
}

func (h *Handler) DeleteSuggestion(c *gin.Context) {
	id := c.Param("id")
	if err := h.suggestions.Delete(c.Request.Context(), id); err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "suggestion not found"})
			return
		}
		log.Error().Err(err).Msg("failed to delete suggestion")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) CreateSuggestion(c *gin.Context) {
	var body struct {
		Name         string   `json:"name" binding:"required"`
		Link         *string  `json:"link"`
		Countries    []string `json:"countries"`
		ListenLabels []string `json:"listenLabels"`
		Evidence     *string  `json:"evidence"`
		Description  *string  `json:"description"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}

	suggestion, err := h.suggestions.Create(
		c.Request.Context(),
		body.Name, body.Link,
		body.Countries, body.ListenLabels,
		body.Evidence, body.Description,
	)
	if err != nil {
		log.Error().Err(err).Msg("failed to create suggestion")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusCreated, suggestion)
}
