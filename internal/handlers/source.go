package handlers

import (
	"net/http"

	"github.com/PhonkersBase/base-api2/internal/domain"
	"github.com/PhonkersBase/base-api2/internal/repository"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetSources(c *gin.Context) {
	sources, err := h.sources.GetAll(c.Request.Context())
	if err != nil {
		internalErr(c, err, "failed to get sources")
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
		internalErr(c, err, "failed to create source")
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
		internalErr(c, err, "failed to update source")
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
		internalErr(c, err, "failed to delete source")
		return
	}
	c.Status(http.StatusNoContent)
}
