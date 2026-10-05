package handlers

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/PhonkersBase/base-api2/internal/domain"
	"github.com/PhonkersBase/base-api2/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// A sync walks the roster 50 collaborators at a time, paced under muso's 300/min
// ceiling. A 4000-artist label costs roughly 20 seconds, but the largest
// distributors run to 45000 collaborators, or some 900 paced requests, so the
// cap has to clear ten minutes of walking to reach them at all.
const organisationMusoSyncTimeout = 20 * time.Minute

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
	organisationID, err := strconv.Atoi(org.ID)
	if h.musoSyncer != nil && err == nil && !domain.ExcludedFromMuso(org.Origin) {
		// The request context dies with the response, so the new organisation's first
		// sync runs on its own deadline rather than blocking the create.
		go func(id int, name string) {
			ctx, cancel := context.WithTimeout(context.Background(), organisationMusoSyncTimeout)
			defer cancel()
			syncErr := h.musoSyncer.SyncOrganisation(ctx, id, name)
			if syncErr != nil {
				log.Error().Err(syncErr).Int("organisation_id", id).Msg("failed to sync new organisation with muso")
			}
		}(organisationID, org.Name)
	}
	c.JSON(http.StatusCreated, org)
}

// GetOrganisationMusoStates backs the admin panel's view of what the last sync
// fetched. sync_status and sync_error stay off the public organisation payload,
// which carries only the distribution the page draws.
func (h *Handler) GetOrganisationMusoStates(c *gin.Context) {
	states, err := h.musoStates.ListStates(c.Request.Context())
	if err != nil {
		internalErr(c, err, "failed to list organisation muso states")
		return
	}
	c.JSON(http.StatusOK, states)
}

func (h *Handler) SyncOrganisationMuso(c *gin.Context) {
	if h.musoSyncer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"message": "muso sync is unavailable"})
		return
	}

	id := c.Param("id")
	org, err := h.organisations.GetByID(c.Request.Context(), id)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "organisation not found"})
			return
		}
		internalErr(c, err, "failed to get organisation for muso sync")
		return
	}

	if domain.ExcludedFromMuso(org.Origin) {
		c.JSON(http.StatusUnprocessableEntity, gin.H{"message": "muso sync is skipped for organisations of this origin"})
		return
	}

	organisationID, err := strconv.Atoi(org.ID)
	if err != nil {
		internalErr(c, err, "invalid organisation id from database")
		return
	}
	// The router caps a request at 30s, which a full roster walk can outlast. Detaching
	// from that cancellation lets the sync finish and commit even when the client is
	// already gone, so no run is wasted halfway through.
	syncCtx, cancel := context.WithTimeout(context.WithoutCancel(c.Request.Context()), organisationMusoSyncTimeout)
	defer cancel()
	err = h.musoSyncer.SyncOrganisation(syncCtx, organisationID, org.Name)
	if err != nil {
		internalErr(c, err, "failed to sync organisation with muso")
		return
	}

	refreshed, err := h.organisations.GetByID(syncCtx, id)
	if err != nil {
		internalErr(c, err, "failed to load organisation distribution after muso sync")
		return
	}
	h.recordChange(c, domain.EntityTypeOrganisation, org.ID, org.Name, domain.ChangeActionMusoSync, org.LabelDistribution, refreshed.LabelDistribution)
	c.JSON(http.StatusOK, refreshed.LabelDistribution)
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
