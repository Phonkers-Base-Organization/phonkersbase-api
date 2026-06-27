package handlers

import (
	"net/http"

	"github.com/PhonkersBase/base-api2/internal/repository"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetFeedbacks(c *gin.Context) {
	feedbacks, err := h.feedbacks.GetAll(c.Request.Context())
	if err != nil {
		internalErr(c, err, "failed to get feedbacks")
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
		internalErr(c, err, "failed to delete feedback")
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
		internalErr(c, err, "failed to create feedback")
		return
	}
	c.JSON(http.StatusCreated, feedback)
}
