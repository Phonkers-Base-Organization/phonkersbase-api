package handlers

import (
	"github.com/PhonkersBase/base-api2/internal/repository"
)

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
