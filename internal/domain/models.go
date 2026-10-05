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
	RosterStats    *RosterStats      `json:"rosterStats"`
	CreatedAt      time.Time         `json:"createdAt"`
	UpdatedAt      time.Time         `json:"updatedAt"`
}

// Country codes the organisation chart singles out. Everything else is "other".
const (
	CountryRussia  = "RU"
	CountryBelarus = "BY"
	CountryUkraine = "UA"
)

// MusoExcludedOrigins lists the origins whose organisations get no muso data at
// all. The site stamps those organisations instead of charting them, so fetching
// their rosters would spend API credits on numbers nobody is shown.
func MusoExcludedOrigins() []string {
	return []string{CountryRussia, CountryBelarus}
}

// ExcludedFromMuso reports whether an organisation of this origin is neither synced
// with muso nor served a distribution.
func ExcludedFromMuso(origin string) bool {
	return origin == CountryRussia || origin == CountryBelarus
}

// RosterStats describes the artists of an organisation's muso roster that pb holds,
// split two ways. Both splits cover the same Matched artists, each counted once, so
// each sums to Matched. RosterSize is everyone muso credits on the organisation;
// most of them are not in pb, which is why the two numbers travel together.
type RosterStats struct {
	Matched    int          `json:"matched"`
	RosterSize int          `json:"rosterSize"`
	SyncedAt   *time.Time   `json:"syncedAt"`
	ByStatus   StatusSplit  `json:"byStatus"`
	ByCountry  CountrySplit `json:"byCountry"`
}

// StatusSplit counts artists by listen label, each under its most severe one.
type StatusSplit struct {
	Blocked  int `json:"blocked"`
	Warning  int `json:"warning"`
	Approved int `json:"approved"`
	Unknown  int `json:"unknown"`
}

// CountrySplit counts artists by their primary curated country. Unknown holds the
// artists pb has no country for.
type CountrySplit struct {
	RuBy    int `json:"ruBy"`
	Ukraine int `json:"ukraine"`
	Other   int `json:"other"`
	Unknown int `json:"unknown"`
}

type OrganisationMuso struct {
	ID                 int64      `json:"id"`
	OrganisationID     int        `json:"organisationId"`
	MusoProfileID      string     `json:"musoProfileId"`
	CollaboratorsCount int        `json:"collaboratorsCount"`
	CreditCount        int        `json:"creditCount"`
	RosterSize         int        `json:"rosterSize"`
	LastCheckedAt      *time.Time `json:"lastCheckedAt"`
	LastSyncedAt       *time.Time `json:"lastSyncedAt"`
	SyncStatus         string     `json:"syncStatus"`
	SyncError          *string    `json:"syncError"`
	CreatedAt          time.Time  `json:"createdAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
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
