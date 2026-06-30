package domain

import "time"

type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

type Label struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Priority  int       `json:"priority"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// LabelRef is the slim representation of a label nested inside an artist's
// listenLabels. Full metadata (priority, timestamps) lives in /label/all and
// can be correlated by ID.
type LabelRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type EvidenceSource struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	NameEn    string    `json:"nameEn"`
	CreatedAt time.Time `json:"createdAt"`
}

// SourceRef is the slim representation of an evidence source nested inside an
// artist country entry. Full metadata lives in /source/all and can be
// correlated by ID.
type SourceRef struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	NameEn string `json:"nameEn"`
}

type ArtistCountry struct {
	Code    string      `json:"code"`
	Sources []SourceRef `json:"sources"`
}

type Artist struct {
	ID            string          `json:"id"`
	Name          string          `json:"name"`
	Link          *string         `json:"link"`
	AvatarURL     *string         `json:"avatarUrl"`
	SpotifyID     *string         `json:"spotifyId"`
	Countries     []ArtistCountry `json:"countries"`
	ListenLabels  []LabelRef      `json:"listenLabels"`
	Description   *string         `json:"description"`
	DescriptionEn *string         `json:"descriptionEn"`
	Notes         *string         `json:"notes"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
}

type Pagination struct {
	Limit       int `json:"limit"`
	Offset      int `json:"offset"`
	Total       int `json:"total"`
	TotalPages  int `json:"totalPages"`
	CurrentPage int `json:"currentPage"`
}

type PaginatedArtists struct {
	Items []Artist   `json:"items"`
	Info  Pagination `json:"info"`
}

type ListArtistsParams struct {
	Locale    string
	Offset    int
	Limit     int
	Search    string
	Countries []string
	Labels    []string
	ByArtist  SortDirection
	ByCountry SortDirection
	ByListen  SortDirection
}

type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Role     string `json:"role"`
}

type Suggestion struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Link         *string   `json:"link"`
	Countries    []string  `json:"countries"`
	ListenLabels []string  `json:"listenLabels"`
	Evidence     *string   `json:"evidence"`
	Description  *string   `json:"description"`
	CreatedAt    time.Time `json:"createdAt"`
}

type Feedback struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"`
	Text      string    `json:"text"`
	Email     *string   `json:"email"`
	CreatedAt time.Time `json:"createdAt"`
}

type ArtistStats struct {
	Total     int     `json:"total"`
	LastAdded *string `json:"lastAdded"`
}

type ArtistCountryInput struct {
	Code      string   `json:"code" binding:"required"`
	SourceIDs []string `json:"sourceIds"`
}

type UpsertArtistInput struct {
	Name          string               `json:"name" binding:"required"`
	Link          *string              `json:"link"`
	AvatarURL     *string              `json:"avatarUrl"`
	SpotifyID     *string              `json:"spotifyId"`
	Description   *string              `json:"description"`
	DescriptionEn *string              `json:"descriptionEn"`
	Countries     []ArtistCountryInput `json:"countries"`
	ListenLabels  []string             `json:"listenLabels"`
	Notes         *string              `json:"notes"`
}

type UpsertSourceInput struct {
	Name   string `json:"name" binding:"required"`
	NameEn string `json:"nameEn"`
}

type UpsertLabelInput struct {
	Name     string `json:"name" binding:"required"`
	Priority int    `json:"priority"`
}

type OrgRecommendation string

const (
	OrgRecNoUse     OrgRecommendation = "Не використовуй"
	OrgRecNoListen  OrgRecommendation = "Не слухай це"
	OrgRecCareful   OrgRecommendation = "Будь обережний"
	OrgRecCanUse    OrgRecommendation = "Можеш використовувати"
	OrgRecCanListen OrgRecommendation = "Можеш слухати"
)

func (r OrgRecommendation) Valid() bool {
	switch r {
	case OrgRecNoUse, OrgRecNoListen, OrgRecCareful, OrgRecCanUse, OrgRecCanListen:
		return true
	}
	return false
}

type Organisation struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Link           *string           `json:"link"`
	Origin         string            `json:"origin"`
	DescriptionUk  *string           `json:"description"`
	DescriptionEn  *string           `json:"descriptionEn"`
	Notes          *string           `json:"notes"`
	Type           string            `json:"type"`
	Recommendation OrgRecommendation `json:"recommendation"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
}

type PaginatedOrganisations struct {
	Items []Organisation `json:"items"`
	Info  Pagination     `json:"info"`
}

type UpsertOrganisationInput struct {
	Name           string            `json:"name" binding:"required"`
	Link           *string           `json:"link"`
	Origin         string            `json:"origin" binding:"required"`
	DescriptionUk  *string           `json:"description"`
	DescriptionEn  *string           `json:"descriptionEn"`
	Notes          *string           `json:"notes"`
	Type           string            `json:"type" binding:"required"`
	Recommendation OrgRecommendation `json:"recommendation" binding:"required"`
}

type ListOrganisationsParams struct {
	Type   string
	Search string
	Limit  int
	Offset int
}
