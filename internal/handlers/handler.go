package handlers

import (
	"github.com/PhonkersBase/base-api2/internal/repository"
)

type Handler struct {
	artists             *repository.ArtistRepo
	labels              *repository.LabelRepo
	users               *repository.UserRepo
	suggestions         *repository.SuggestionRepo
	feedbacks           *repository.FeedbackRepo
	jwtSecret           string
	spotifyClientID     string
	spotifyClientSecret string
}

func NewHandler(
	artists *repository.ArtistRepo,
	labels *repository.LabelRepo,
	users *repository.UserRepo,
	suggestions *repository.SuggestionRepo,
	feedbacks *repository.FeedbackRepo,
	jwtSecret, spotifyClientID, spotifyClientSecret string,
) *Handler {
	return &Handler{
		artists:             artists,
		labels:              labels,
		users:               users,
		suggestions:         suggestions,
		feedbacks:           feedbacks,
		jwtSecret:           jwtSecret,
		spotifyClientID:     spotifyClientID,
		spotifyClientSecret: spotifyClientSecret,
	}
}
