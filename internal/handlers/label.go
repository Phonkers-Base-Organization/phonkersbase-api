package handlers

import (
	"net/http"

	"github.com/PhonkersBase/base-api2/internal/domain"
	"github.com/PhonkersBase/base-api2/internal/repository"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetLabels(c *gin.Context) {
	labels, err := h.labels.GetAll(c.Request.Context())
	if err != nil {
		internalErr(c, err, "failed to get labels")
		return
	}
	c.JSON(http.StatusOK, labels)
}

func (h *Handler) CreateLabel(c *gin.Context) {
	var input domain.UpsertLabelInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	label, err := h.labels.Create(c.Request.Context(), input)
	if err != nil {
		internalErr(c, err, "failed to create label")
		return
	}
	c.JSON(http.StatusCreated, label)
}

func (h *Handler) UpdateLabel(c *gin.Context) {
	id := c.Param("id")
	var input domain.UpsertLabelInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	label, err := h.labels.Update(c.Request.Context(), id, input)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "label not found"})
			return
		}
		internalErr(c, err, "failed to update label")
		return
	}
	c.JSON(http.StatusOK, label)
}

func (h *Handler) DeleteLabel(c *gin.Context) {
	id := c.Param("id")
	if err := h.labels.Delete(c.Request.Context(), id); err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "label not found"})
			return
		}
		internalErr(c, err, "failed to delete label")
		return
	}
	c.Status(http.StatusNoContent)
}
