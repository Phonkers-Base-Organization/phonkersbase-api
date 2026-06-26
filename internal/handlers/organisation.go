package handlers

import (
	"net/http"

	"github.com/PhonkersBase/base-api2/internal/domain"
	"github.com/PhonkersBase/base-api2/internal/repository"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetOrganisations(c *gin.Context) {
	params := domain.ListOrganisationsParams{
		Type:   c.Query("type"),
		Search: c.Query("search"),
	}
	orgs, err := h.organisations.GetAll(c.Request.Context(), params)
	if err != nil {
		internalErr(c, err, "failed to get organisations")
		return
	}
	c.JSON(http.StatusOK, orgs)
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

	org, err := h.organisations.Update(c.Request.Context(), id, input)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "organisation not found"})
			return
		}
		internalErr(c, err, "failed to update organisation")
		return
	}
	c.JSON(http.StatusOK, org)
}

func (h *Handler) DeleteOrganisation(c *gin.Context) {
	id := c.Param("id")
	if err := h.organisations.Delete(c.Request.Context(), id); err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "organisation not found"})
			return
		}
		internalErr(c, err, "failed to delete organisation")
		return
	}
	c.Status(http.StatusNoContent)
}
