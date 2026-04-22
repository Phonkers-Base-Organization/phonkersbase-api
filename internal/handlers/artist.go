package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/PhonkersBase/base-api2/internal/domain"
	"github.com/PhonkersBase/base-api2/internal/repository"
)

func (h *Handler) GetArtists(c *gin.Context) {
	locale := c.Query("locale")
	if locale != "uk" && locale != "en" {
		c.JSON(http.StatusBadRequest, gin.H{"errors": gin.H{"locale": "must be one of: uk, en"}})
		return
	}

	p := domain.ListArtistsParams{
		Locale:    locale,
		Search:    c.Query("search"),
		ByArtist:  domain.SortDirection(c.Query("by_artist")),
		ByCountry: domain.SortDirection(c.Query("by_country")),
		ByListen:  domain.SortDirection(c.Query("by_listen")),
	}

	if v := c.Query("country"); v != "" {
		p.Countries = splitCSV(v)
	}
	if v := c.Query("label"); v != "" {
		p.Labels = splitCSV(v)
	}

	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			p.Offset = n
		}
	}
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 200 {
			c.JSON(http.StatusBadRequest, gin.H{"errors": gin.H{"limit": "must be between 1 and 200"}})
			return
		}
		p.Limit = n
	}

	result, err := h.artists.GetAll(c.Request.Context(), p)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errJSON(err.Error()))
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetArtist(c *gin.Context) {
	id, err := strconv.Atoi(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, errJSON("invalid id"))
		return
	}

	artist, err := h.artists.GetByID(c.Request.Context(), id)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.JSON(http.StatusNotFound, errJSON("artist not found"))
			return
		}
		c.JSON(http.StatusInternalServerError, errJSON(err.Error()))
		return
	}
	c.JSON(http.StatusOK, artist)
}

func splitCSV(s string) []string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if v := strings.TrimSpace(p); v != "" {
			out = append(out, v)
		}
	}
	return out
}
