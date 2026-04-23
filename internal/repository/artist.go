package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PhonkersBase/base-api2/internal/domain"
)

var ErrNotFound = errors.New("not found")

type ArtistRepo struct {
	db *pgxpool.Pool
}

func NewArtistRepo(db *pgxpool.Pool) *ArtistRepo {
	return &ArtistRepo{db: db}
}

func (r *ArtistRepo) GetByID(ctx context.Context, id int) (*domain.Artist, error) {
	var (
		aid                       int
		link, spotifyID           *string
		avatarURL, descUA, descEN *string
		name                      string
		countries                 []string
		createdAt, updatedAt      time.Time
	)
	err := r.db.QueryRow(ctx, `
		SELECT id, name, link, spotify_id, avatar_url,
		       description_ua, description_en, countries,
		       created_at, updated_at
		FROM artists
		WHERE id = $1
	`, id).Scan(
		&aid, &name, &link, &spotifyID, &avatarURL,
		&descUA, &descEN, &countries,
		&createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	countryLookup, err := r.fetchCountriesByNames(ctx, countries)
	if err != nil {
		return nil, err
	}

	labels, err := r.fetchLabels(ctx, []int{aid})
	if err != nil {
		return nil, err
	}

	a := buildArtist(aid, name, link, spotifyID, avatarURL, descUA, descEN,
		resolvecountries(countries, countryLookup),
		createdAt, updatedAt,
		labels[aid])
	return &a, nil
}

func (r *ArtistRepo) GetAll(ctx context.Context, p domain.ListArtistsParams) (*domain.PaginatedArtists, error) {
	args := []any{}
	conditions := []string{}
	n := 1

	if p.Search != "" {
		descCol := "a.description_ua"
		if p.Locale == "en" {
			descCol = "a.description_en"
		}
		conditions = append(conditions, fmt.Sprintf(
			"(a.name ILIKE $%d OR %s ILIKE $%d)", n, descCol, n,
		))
		args = append(args, "%"+p.Search+"%")
		n++
	}
	if len(p.Countries) > 0 {
		conditions = append(conditions, fmt.Sprintf("a.countries && $%d", n))
		args = append(args, p.Countries)
		n++
	}
	if len(p.Labels) > 0 {
		conditions = append(conditions, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM artist_labels al2 JOIN labels l2 ON l2.id = al2.label_id WHERE al2.artist_id = a.id AND l2.name = ANY($%d))", n,
		))
		args = append(args, p.Labels)
		n++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	orderParts := []string{}
	if p.ByArtist != "" {
		orderParts = append(orderParts, "a.name "+safeDir(p.ByArtist))
	}
	if p.ByCountry != "" {
		nullsPos := "LAST"
		if p.ByCountry == domain.SortAsc {
			nullsPos = "FIRST"
		}
		orderParts = append(orderParts, fmt.Sprintf("a.countries[1] %s NULLS %s", safeDir(p.ByCountry), nullsPos))
	}
	if p.ByListen != "" {
		orderParts = append(orderParts, "a.total_priority "+safeDir(p.ByListen))
	}
	if len(orderParts) == 0 {
		orderParts = []string{"a.total_priority DESC"}
	}
	orderParts = append(orderParts, "a.id ASC")
	orderClause := "ORDER BY " + strings.Join(orderParts, ", ")

	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}

	q := fmt.Sprintf(`
		SELECT
			a.id, a.name, a.link, a.spotify_id, a.avatar_url,
			a.description_ua, a.description_en, a.countries,
			a.created_at, a.updated_at,
			COUNT(*) OVER() AS total_count
		FROM artists a
		%s
		%s
		LIMIT $%d OFFSET $%d
	`, whereClause, orderClause, n, n+1)

	args = append(args, limit, p.Offset)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type rawRow struct {
		id                         int
		name                       string
		link, spotifyID, avatarURL *string
		descUA, descEN             *string
		rawCountries               []string
		createdAt, updatedAt       time.Time
		totalCount                 int
	}

	var (
		raws      []rawRow
		artistIDs []int
		nameSet   = map[string]struct{}{}
		total     int
	)

	for rows.Next() {
		var r rawRow
		if err := rows.Scan(
			&r.id, &r.name, &r.link, &r.spotifyID, &r.avatarURL,
			&r.descUA, &r.descEN, &r.rawCountries,
			&r.createdAt, &r.updatedAt,
			&r.totalCount,
		); err != nil {
			return nil, err
		}
		total = r.totalCount
		raws = append(raws, r)
		artistIDs = append(artistIDs, r.id)
		for _, c := range r.rawCountries {
			nameSet[c] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	allNames := make([]string, 0, len(nameSet))
	for n := range nameSet {
		allNames = append(allNames, n)
	}

	countryLookup, err := r.fetchCountriesByNames(ctx, allNames)
	if err != nil {
		return nil, err
	}

	labelsByArtist := map[int][]domain.Label{}
	if len(artistIDs) > 0 {
		labelsByArtist, err = r.fetchLabels(ctx, artistIDs)
		if err != nil {
			return nil, err
		}
	}

	artists := make([]domain.Artist, len(raws))
	for i, r := range raws {
		artists[i] = buildArtist(
			r.id, r.name, r.link, r.spotifyID, r.avatarURL,
			r.descUA, r.descEN,
			resolvecountries(r.rawCountries, countryLookup),
			r.createdAt, r.updatedAt,
			labelsByArtist[r.id],
		)
	}

	totalPages := 0
	if total > 0 && limit > 0 {
		totalPages = int(math.Ceil(float64(total) / float64(limit)))
	}
	currentPage := 1
	if limit > 0 && p.Offset > 0 {
		currentPage = p.Offset/limit + 1
	}

	return &domain.PaginatedArtists{
		Items: artists,
		Info: domain.Pagination{
			Limit:       limit,
			Offset:      p.Offset,
			Total:       total,
			TotalPages:  totalPages,
			CurrentPage: currentPage,
		},
	}, nil
}

func (r *ArtistRepo) fetchLabels(ctx context.Context, artistIDs []int) (map[int][]domain.Label, error) {
	rows, err := r.db.Query(ctx, `
		SELECT al.artist_id, l.id, l.name, l.original_name, l.priority, l.created_at, l.updated_at
		FROM labels l
		JOIN artist_labels al ON al.label_id = l.id
		WHERE al.artist_id = ANY($1)
		ORDER BY l.priority DESC, l.name ASC
	`, artistIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[int][]domain.Label{}
	for rows.Next() {
		var (
			artistID, id         int
			name, originalName   string
			priority             int
			createdAt, updatedAt time.Time
		)
		if err := rows.Scan(&artistID, &id, &name, &originalName, &priority, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		result[artistID] = append(result[artistID], domain.Label{
			ID:           strconv.Itoa(id),
			Name:         domain.LabelName(name),
			OriginalName: originalName,
			Priority:     priority,
			CreatedAt:    createdAt,
			UpdatedAt:    updatedAt,
		})
	}
	return result, rows.Err()
}

func (r *ArtistRepo) fetchCountriesByNames(ctx context.Context, names []string) (map[string]domain.Country, error) {
	if len(names) == 0 {
		return map[string]domain.Country{}, nil
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, name, original_name, created_at, updated_at
		FROM countries
		WHERE name = ANY($1)
	`, names)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[string]domain.Country{}
	for rows.Next() {
		var (
			id                   int
			name, originalName   string
			createdAt, updatedAt time.Time
		)
		if err := rows.Scan(&id, &name, &originalName, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		result[name] = domain.Country{
			ID:           strconv.Itoa(id),
			Name:         name,
			OriginalName: originalName,
			CreatedAt:    createdAt,
			UpdatedAt:    updatedAt,
		}
	}
	return result, rows.Err()
}

func resolvecountries(names []string, lookup map[string]domain.Country) []domain.Country {
	out := make([]domain.Country, 0, len(names))
	for _, name := range names {
		if c, ok := lookup[name]; ok {
			out = append(out, c)
		} else {
			out = append(out, domain.CountryFromName(name))
		}
	}
	return out
}

func buildArtist(
	id int,
	name string,
	link, spotifyID, avatarURL, descUA, descEN *string,
	countries []domain.Country,
	createdAt, updatedAt time.Time,
	labels []domain.Label,
) domain.Artist {
	if labels == nil {
		labels = []domain.Label{}
	}
	return domain.Artist{
		ID:            strconv.Itoa(id),
		Name:          name,
		Link:          link,
		SpotifyID:     spotifyID,
		AvatarURL:     avatarURL,
		Countries:     countries,
		ListenLabels:  labels,
		Description:   descUA,
		DescriptionEn: descEN,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}

func safeDir(d domain.SortDirection) string {
	if d == domain.SortDesc {
		return "DESC"
	}
	return "ASC"
}
