package handlers

import (
	"github.com/PhonkersBase/base-api2/internal/repository"
)

type Handler struct {
	artists   *repository.ArtistRepo
	countries *repository.CountryRepo
	labels    *repository.LabelRepo
}

func NewHandler(artists *repository.ArtistRepo, countries *repository.CountryRepo, labels *repository.LabelRepo) *Handler {
	return &Handler{artists: artists, countries: countries, labels: labels}
}
