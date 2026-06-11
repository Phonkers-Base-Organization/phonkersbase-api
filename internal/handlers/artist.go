package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/PhonkersBase/base-api2/internal/domain"
	"github.com/PhonkersBase/base-api2/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

func (h *Handler) GetArtists(c *gin.Context) {
	locale := c.Query("locale")
	if locale == "" {
		locale = "uk"
	}
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
		log.Error().Err(err).Msg("failed to get artists")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) GetAdminArtists(c *gin.Context) {
	limit := 0
	offset := 0
	if v := c.Query("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil || n < 1 || n > 200 {
			c.JSON(http.StatusBadRequest, gin.H{"errors": gin.H{"limit": "must be between 1 and 200"}})
			return
		}
		limit = n
	}
	if v := c.Query("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}

	artists, total, err := h.artists.GetAdminAll(c.Request.Context(), limit, offset)
	if err != nil {
		log.Error().Err(err).Msg("failed to get admin artists")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, gin.H{"artists": artists, "total": total})
}

func (h *Handler) CreateArtist(c *gin.Context) {
	var input domain.UpsertArtistInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	artist, err := h.artists.Create(c.Request.Context(), input)
	if err != nil {
		log.Error().Err(err).Msg("failed to create artist")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusCreated, artist)
}

func (h *Handler) UpdateArtist(c *gin.Context) {
	id := c.Param("id")
	var input domain.UpsertArtistInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	artist, err := h.artists.Update(c.Request.Context(), id, input)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "artist not found"})
			return
		}
		log.Error().Err(err).Msg("failed to update artist")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, artist)
}

func (h *Handler) DeleteArtist(c *gin.Context) {
	id := c.Param("id")
	if err := h.artists.Delete(c.Request.Context(), id); err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "artist not found"})
			return
		}
		log.Error().Err(err).Msg("failed to delete artist")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) GetArtistStats(c *gin.Context) {
	stats, err := h.artists.GetStats(c.Request.Context())
	if err != nil {
		log.Error().Err(err).Msg("failed to get artist stats")
		c.Status(http.StatusInternalServerError)
		return
	}
	c.JSON(http.StatusOK, stats)
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
