package handlers

import (
	"net/http"
	"strconv"

	"github.com/PhonkersBase/base-api2/internal/domain"
	"github.com/PhonkersBase/base-api2/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func (h *Handler) GetLabelOrganisations(c *gin.Context) {
	h.getOrganisationsByType(c, "Label")
}

func (h *Handler) GetDistributorOrganisations(c *gin.Context) {
	h.getOrganisationsByType(c, "Distributor")
}

func (h *Handler) GetCultOrganisations(c *gin.Context) {
	h.getOrganisationsByType(c, "Cult")
}

func (h *Handler) getOrganisationsByType(c *gin.Context, orgType string) {
	params := domain.ListOrganisationsParams{
		Type:   orgType,
		Search: c.Query("search"),
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			params.Offset = n
		}
	}
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 500 {
			c.JSON(http.StatusBadRequest, gin.H{"errors": gin.H{"limit": "must be between 1 and 500"}})
			return
		}
		params.Limit = n
	}

	result, err := h.organisations.GetAll(c.Request.Context(), params)
	if err != nil {
		internalErr(c, err, "failed to get organisations")
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetOrganisations(c *gin.Context) {
	params := domain.ListOrganisationsParams{
		Type:   c.Query("type"),
		Search: c.Query("search"),
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			params.Offset = n
		}
	}
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 500 {
			c.JSON(http.StatusBadRequest, gin.H{"errors": gin.H{"limit": "must be between 1 and 500"}})
			return
		}
		params.Limit = n
	}

	result, err := h.organisations.GetAll(c.Request.Context(), params)
	if err != nil {
		internalErr(c, err, "failed to get organisations")
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) CreateOrganisation(c *gin.Context) {
	var input domain.UpsertOrganisationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	if !input.Recommendation.Valid() {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid recommendation value"})
		return
	}

	org, err := h.organisations.Create(c.Request.Context(), input)
	if err != nil {
		internalErr(c, err, "failed to create organisation")
		return
	}
	h.recordChange(c, domain.EntityTypeOrganisation, org.ID, org.Name, domain.ChangeActionCreate, nil, org)
	c.JSON(http.StatusCreated, org)
}

func (h *Handler) UpdateOrganisation(c *gin.Context) {
	id := c.Param("id")
	var input domain.UpsertOrganisationInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}
	if !input.Recommendation.Valid() {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid recommendation value"})
		return
	}

	old, err := h.organisations.GetByID(c.Request.Context(), id)
	if err != nil && err != repository.ErrNotFound {
		log.Warn().Err(err).Str("id", id).Msg("failed to fetch organisation before update, recording change without old data")
		old = nil
	}

	org, err := h.organisations.Update(c.Request.Context(), id, input)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "organisation not found"})
			return
		}
		internalErr(c, err, "failed to update organisation")
		return
	}
	h.recordChange(c, domain.EntityTypeOrganisation, org.ID, org.Name, domain.ChangeActionUpdate, old, org)
	c.JSON(http.StatusOK, org)
}

func (h *Handler) DeleteOrganisation(c *gin.Context) {
	id := c.Param("id")

	old, err := h.organisations.GetByID(c.Request.Context(), id)
	if err != nil && err != repository.ErrNotFound {
		log.Warn().Err(err).Str("id", id).Msg("failed to fetch organisation before delete, recording change without old data")
		old = nil
	}

	if err := h.organisations.Delete(c.Request.Context(), id); err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "organisation not found"})
			return
		}
		internalErr(c, err, "failed to delete organisation")
		return
	}
	deletedName := ""
	if old != nil {
		deletedName = old.Name
	}
	h.recordChange(c, domain.EntityTypeOrganisation, id, deletedName, domain.ChangeActionDelete, old, nil)
	c.Status(http.StatusNoContent)
}
