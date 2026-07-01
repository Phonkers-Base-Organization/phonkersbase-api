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

	search := c.Query("search")
	if msg := validateSearch(search); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"errors": gin.H{"search": msg}})
		return
	}

	p := domain.ListArtistsParams{
		Locale:    locale,
		Search:    search,
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
		internalErr(c, err, "failed to get artists")
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

	search := c.Query("search")
	if msg := validateSearch(search); msg != "" {
		c.JSON(http.StatusBadRequest, gin.H{"errors": gin.H{"search": msg}})
		return
	}

	artists, total, err := h.artists.GetAdminAll(c.Request.Context(), limit, offset, search)
	if err != nil {
		internalErr(c, err, "failed to get admin artists")
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
		internalErr(c, err, "failed to create artist")
		return
	}
	h.recordChange(c, domain.EntityTypeArtist, artist.ID, artist.Name, domain.ChangeActionCreate, nil, artist)
	c.JSON(http.StatusCreated, artist)
}

func (h *Handler) UpdateArtist(c *gin.Context) {
	id := c.Param("id")
	var input domain.UpsertArtistInput
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": err.Error()})
		return
	}

	var old *domain.Artist
	if aid, convErr := strconv.Atoi(id); convErr == nil {
		fetched, fetchErr := h.artists.GetByID(c.Request.Context(), aid)
		if fetchErr != nil && fetchErr != repository.ErrNotFound {
			log.Warn().Err(fetchErr).Str("id", id).Msg("failed to fetch artist before update, recording change without old data")
		} else {
			old = fetched
		}
	}

	artist, err := h.artists.Update(c.Request.Context(), id, input)
	if err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "artist not found"})
			return
		}
		internalErr(c, err, "failed to update artist")
		return
	}
	if old != nil {
		h.recordChange(c, domain.EntityTypeArtist, artist.ID, artist.Name, domain.ChangeActionUpdate, old, artist)
	} else {
		h.recordChange(c, domain.EntityTypeArtist, artist.ID, artist.Name, domain.ChangeActionUpdate, nil, artist)
	}
	c.JSON(http.StatusOK, artist)
}

func (h *Handler) DeleteArtist(c *gin.Context) {
	id := c.Param("id")

	var old *domain.Artist
	if aid, convErr := strconv.Atoi(id); convErr == nil {
		fetched, fetchErr := h.artists.GetByID(c.Request.Context(), aid)
		if fetchErr != nil && fetchErr != repository.ErrNotFound {
			log.Warn().Err(fetchErr).Str("id", id).Msg("failed to fetch artist before delete, recording change without old data")
		} else {
			old = fetched
		}
	}

	if err := h.artists.Delete(c.Request.Context(), id); err != nil {
		if err == repository.ErrNotFound {
			c.JSON(http.StatusNotFound, gin.H{"message": "artist not found"})
			return
		}
		internalErr(c, err, "failed to delete artist")
		return
	}
	if old != nil {
		h.recordChange(c, domain.EntityTypeArtist, id, old.Name, domain.ChangeActionDelete, old, nil)
	} else {
		h.recordChange(c, domain.EntityTypeArtist, id, "", domain.ChangeActionDelete, nil, nil)
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) GetArtistStats(c *gin.Context) {
	stats, err := h.artists.GetStats(c.Request.Context())
	if err != nil {
		internalErr(c, err, "failed to get artist stats")
		return
	}
	c.JSON(http.StatusOK, stats)
}

const (
	maxSearchLen   = 2048
	maxSearchTerms = 50
)

// validateSearch bounds the `search` query param before any parsing happens, so a single
// oversized string or term-heavy comma list can't force an unbounded allocation or an
// excessively large OR-tree in the repository query. Returns a client-facing message, or ""
// if the value is within bounds.
func validateSearch(search string) string {
	if len(search) > maxSearchLen {
		return "must not exceed " + strconv.Itoa(maxSearchLen) + " characters"
	}
	count := 0
	for term := range strings.SplitSeq(search, ",") {
		if strings.TrimSpace(term) == "" {
			continue
		}
		count++
		if count > maxSearchTerms {
			return "must not contain more than " + strconv.Itoa(maxSearchTerms) + " comma-separated terms"
		}
	}
	return ""
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
