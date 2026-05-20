package handlers

import (
	"net/http"
	"github.com/rs/zerolog/log"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetCountries(c *gin.Context) {
	countries, err := h.countries.GetAll(c.Request.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to get countries")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, countries)
}
