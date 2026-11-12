package handlers

import (
	"net/http"

	"github.com/PhonkersBase/base-api2/internal/domain"
	"github.com/gin-gonic/gin"
)

func (h *Handler) GetSiteSettings(c *gin.Context) {
	settings, err := h.siteSettings.Get(c.Request.Context())
	if err != nil {
		internalErr(c, err, "failed to get site settings")
		return
	}
	c.JSON(http.StatusOK, settings)
}

func (h *Handler) UpdateSiteSettings(c *gin.Context) {
	var input domain.SiteSettings
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	settings, err := h.siteSettings.Update(c.Request.Context(), input)
	if err != nil {
		internalErr(c, err, "failed to update site settings")
		return
	}
	c.JSON(http.StatusOK, settings)
}
