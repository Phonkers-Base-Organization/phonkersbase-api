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
	h.recordChange(c, domain.EntityTypeSource, source.ID, source.Name, domain.ChangeActionCreate, nil, source)
	c.JSON(http.StatusCreated, source)
}

func (h *Handler) UpdateSource(c *gin.Context) {
	id := c.Param("id")
	var input domain.UpsertSourceInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	old, err := h.sources.GetByID(c.Request.Context(), id)
	if err != nil && err != repository.ErrNotFound {
		log.Warn().Err(err).Str("id", id).Msg("failed to fetch source before update, recording change without old data")
		old = nil
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
	h.recordChange(c, domain.EntityTypeSource, source.ID, source.Name, domain.ChangeActionUpdate, old, source)
	c.JSON(http.StatusOK, source)
}

func (h *Handler) DeleteSource(c *gin.Context) {
	id := c.Param("id")

	old, err := h.sources.GetByID(c.Request.Context(), id)
	if err != nil && err != repository.ErrNotFound {
		log.Warn().Err(err).Str("id", id).Msg("failed to fetch source before delete, recording change without old data")
		old = nil
	}

	if err := h.sources.Delete(c.Request.Context(), id); err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "source not found"})
			return
		}
		internalErr(c, err, "failed to delete source")
		return
	}
	deletedName := ""
	if old != nil {
		deletedName = old.Name
	}
	h.recordChange(c, domain.EntityTypeSource, id, deletedName, domain.ChangeActionDelete, old, nil)
	c.Status(http.StatusNoContent)
}
