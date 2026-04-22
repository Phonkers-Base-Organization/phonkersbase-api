package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) GetCountries(c *gin.Context) {
	countries, err := h.countries.GetAll(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, errJSON(err.Error()))
		return
	}
	c.JSON(http.StatusOK, countries)
}
