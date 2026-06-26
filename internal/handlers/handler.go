package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/PhonkersBase/base-api2/internal/repository"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// internalErr logs err at error level and responds 500, but silently aborts
// if the client disconnected (context.Canceled). DeadlineExceeded is still
// logged because it means a server-side timeout fired, not a client drop.
func internalErr(c *gin.Context, err error, msg string) {
	if errors.Is(err, context.Canceled) {
		c.Abort()
		return
	}
	log.Error().Err(err).Msg(msg)
	c.Status(http.StatusInternalServerError)
}

type Handler struct {
	artists       *repository.ArtistRepo
	labels        *repository.LabelRepo
	sources       *repository.EvidenceSourceRepo
	users         *repository.UserRepo
	suggestions   *repository.SuggestionRepo
	feedbacks     *repository.FeedbackRepo
	organisations *repository.OrganisationRepo
	jwtSecret     string
}

func NewHandler(
	artists *repository.ArtistRepo,
	labels *repository.LabelRepo,
	sources *repository.EvidenceSourceRepo,
	users *repository.UserRepo,
	suggestions *repository.SuggestionRepo,
	feedbacks *repository.FeedbackRepo,
	organisations *repository.OrganisationRepo,
	jwtSecret string,
) *Handler {
	return &Handler{
		artists:       artists,
		labels:        labels,
		sources:       sources,
		users:         users,
		suggestions:   suggestions,
		feedbacks:     feedbacks,
		organisations: organisations,
		jwtSecret:     jwtSecret,
	}
}
