package domain

import "time"

type SortDirection string

const (
	SortAsc  SortDirection = "asc"
	SortDesc SortDirection = "desc"
)

type LabelName string

const (
	LabelApproved LabelName = "approved"
	LabelBlocked  LabelName = "blocked"
	LabelWarning  LabelName = "warning"
	LabelUnknown  LabelName = "unknown"
	LabelPride    LabelName = "pride"
	LabelBase     LabelName = "base"
)

type Country struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	OriginalName string    `json:"originalName"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

func CountryFromName(name string) Country {
	return Country{ID: name, Name: name, OriginalName: name}
}

type Label struct {
	ID           string    `json:"id"`
	Name         LabelName `json:"name"`
	OriginalName string    `json:"originalName"`
	Priority     int       `json:"priority"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

type Artist struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Link          *string   `json:"link"`
	AvatarURL     *string   `json:"avatarUrl"`
	SpotifyID     *string   `json:"spotifyId"`
	Countries     []Country `json:"countries"`
	ListenLabels  []Label   `json:"listenLabels"`
	Description   *string   `json:"description"`
	DescriptionEn *string   `json:"descriptionEn"`
	CreatedAt     time.Time `json:"createdAt"`
	UpdatedAt     time.Time `json:"updatedAt"`
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
