package domain

import "time"

type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

type Label struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	OriginalName string    `json:"originalName"`
	Priority     int       `json:"priority"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// LabelRef is the slim representation of a label nested inside an artist's
// listenLabels. Full metadata (priority, timestamps) lives in /label/all and
// can be correlated by ID.
type LabelRef struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	OriginalName string `json:"originalName"`
}

type EvidenceSource struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	CreatedAt time.Time `json:"createdAt"`
}

// SourceRef is the slim representation of an evidence source nested inside an
// artist country entry. Full metadata lives in /source/all and can be
// correlated by ID.
type SourceRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ArtistCountry struct {
	Code   string     `json:"code"`
	Source *SourceRef `json:"source"`
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
	Status       string    `json:"status"`
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
	Code     string  `json:"code" binding:"required"`
	SourceID *string `json:"sourceId"`
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
	Name string `json:"name" binding:"required"`
}

type UpsertLabelInput struct {
	Name         string `json:"name" binding:"required"`
	OriginalName string `json:"originalName" binding:"required"`
	Priority     int    `json:"priority"`
}
