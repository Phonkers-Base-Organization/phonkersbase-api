package handlers

import (
	"net/http"

	"github.com/PhonkersBase/base-api2/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func (h *Handler) GetFeedbacks(c *gin.Context) {
	feedbacks, err := h.feedbacks.GetAll(c.Request.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to get feedbacks")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, gin.H{"feedbacks": feedbacks})
}

func (h *Handler) DeleteFeedback(c *gin.Context) {
	id := c.Param("id")
	if err := h.feedbacks.Delete(c.Request.Context(), id); err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "feedback not found"})
			return
		}
		log.Error().Err(err).Msg("failed to delete feedback")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) CreateFeedback(c *gin.Context) {
	var body struct {
		Type  string  `json:"type" binding:"required"`
		Text  string  `json:"text" binding:"required"`
		Email *string `json:"email"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "Invalid request"})
		return
	}

	feedback, err := h.feedbacks.Create(c.Request.Context(), body.Type, body.Text, body.Email)
	if err != nil {
		log.Error().Err(err).Msg("failed to create feedback")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusCreated, feedback)
}
