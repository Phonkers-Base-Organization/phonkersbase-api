package handlers

import (
	"github.com/gin-gonic/gin"

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

func errJSON(msg string) gin.H {
	return gin.H{"errors": gin.H{"message": msg}}
}
