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

type ArtistSource struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type Artist struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Link           *string        `json:"link"`
	AvatarURL      *string        `json:"avatarUrl"`
	SpotifyID      *string        `json:"spotifyId"`
	Countries      []string       `json:"countries"`
	ListenLabels   []Label        `json:"listenLabels"`
	Description    *string        `json:"description"`
	DescriptionEn  *string        `json:"descriptionEn"`
	PrimaryCountry *string        `json:"primaryCountry"`
	EvidenceURL    *string        `json:"evidenceUrl"`
	Notes          *string        `json:"notes"`
	Sources        []ArtistSource `json:"sources"`
	CreatedAt      time.Time      `json:"createdAt"`
	UpdatedAt      time.Time      `json:"updatedAt"`
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
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
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
	UpdatedAt    time.Time `json:"updatedAt"`
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

type UpsertArtistInput struct {
	Name           string         `json:"name" binding:"required"`
	Link           *string        `json:"link"`
	AvatarURL      *string        `json:"avatarUrl"`
	SpotifyID      *string        `json:"spotifyId"`
	Description    *string        `json:"description"`
	DescriptionEn  *string        `json:"descriptionEn"`
	PrimaryCountry *string        `json:"primaryCountry"`
	Countries      []string       `json:"countries"`
	ListenLabels   []string       `json:"listenLabels"`
	EvidenceURL    *string        `json:"evidenceUrl"`
	Notes          *string        `json:"notes"`
	Sources        []ArtistSource `json:"sources"`
}

type UpsertLabelInput struct {
	Name         string `json:"name" binding:"required"`
	OriginalName string `json:"originalName" binding:"required"`
	Priority     int    `json:"priority"`
}
