package repository

import (
	"context"
	"encoding/json"
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
		aid                                    int
		link, spotifyID                        *string
		avatarURL, descUA, descEN              *string
		primaryCountry, evidenceURL, notes     *string
		name                                   string
		countries                              []string
		sourcesRaw                             []byte
		createdAt, updatedAt                   time.Time
	)
	err := r.db.QueryRow(ctx, `
		SELECT id, name, link, spotify_id, avatar_url,
		       description_ua, description_en, countries,
		       primary_country, evidence_url, notes, sources,
		       created_at, updated_at
		FROM artists
		WHERE id = $1
	`, id).Scan(
		&aid, &name, &link, &spotifyID, &avatarURL,
		&descUA, &descEN, &countries,
		&primaryCountry, &evidenceURL, &notes, &sourcesRaw,
		&createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	labels, err := r.fetchLabels(ctx, []int{aid})
	if err != nil {
		return nil, err
	}

	sources := parseSources(sourcesRaw)
	a := buildArtist(aid, name, link, spotifyID, avatarURL, descUA, descEN,
		countries, primaryCountry, evidenceURL, notes, sources,
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
		searchBase := p.Search
		if i := strings.Index(searchBase, "?"); i != -1 {
			searchBase = searchBase[:i]
		}
		orClauses := []string{
			fmt.Sprintf("a.name ILIKE $%d", n),
			fmt.Sprintf("%s ILIKE $%d", descCol, n),
			fmt.Sprintf("SPLIT_PART(a.link, '?', 1) ILIKE $%d", n),
		}
		args = append(args, "%"+searchBase+"%")
		n++
		if spotifyID := extractSpotifyID(p.Search); spotifyID != "" {
			orClauses = append(orClauses, fmt.Sprintf("a.spotify_id = $%d", n))
			args = append(args, spotifyID)
			n++
		}
		conditions = append(conditions, "("+strings.Join(orClauses, " OR ")+")")
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
		total     int
	)

	for rows.Next() {
		var row rawRow
		if err := rows.Scan(
			&row.id, &row.name, &row.link, &row.spotifyID, &row.avatarURL,
			&row.descUA, &row.descEN, &row.rawCountries,
			&row.createdAt, &row.updatedAt,
			&row.totalCount,
		); err != nil {
			return nil, err
		}
		total = row.totalCount
		raws = append(raws, row)
		artistIDs = append(artistIDs, row.id)
	}
	if err := rows.Err(); err != nil {
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
	for i, row := range raws {
		artists[i] = buildArtist(
			row.id, row.name, row.link, row.spotifyID, row.avatarURL,
			row.descUA, row.descEN,
			row.rawCountries, nil, nil, nil, nil,
			row.createdAt, row.updatedAt,
			labelsByArtist[row.id],
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

func (r *ArtistRepo) GetAdminAll(ctx context.Context) ([]domain.Artist, error) {
	rows, err := r.db.Query(ctx, `
		SELECT
			a.id, a.name, a.link, a.spotify_id, a.avatar_url,
			a.description_ua, a.description_en, a.countries,
			a.primary_country, a.evidence_url, a.notes, a.sources,
			a.created_at, a.updated_at
		FROM artists a
		ORDER BY a.total_priority DESC, a.id ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type rawRow struct {
		id                                 int
		name                               string
		link, spotifyID, avatarURL         *string
		descUA, descEN                     *string
		rawCountries                       []string
		primaryCountry, evidenceURL, notes *string
		sourcesRaw                         []byte
		createdAt, updatedAt               time.Time
	}

	var (
		raws      []rawRow
		artistIDs []int
	)

	for rows.Next() {
		var row rawRow
		if err := rows.Scan(
			&row.id, &row.name, &row.link, &row.spotifyID, &row.avatarURL,
			&row.descUA, &row.descEN, &row.rawCountries,
			&row.primaryCountry, &row.evidenceURL, &row.notes, &row.sourcesRaw,
			&row.createdAt, &row.updatedAt,
		); err != nil {
			return nil, err
		}
		raws = append(raws, row)
		artistIDs = append(artistIDs, row.id)
	}
	if err := rows.Err(); err != nil {
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
	for i, row := range raws {
		sources := parseSources(row.sourcesRaw)
		artists[i] = buildArtist(
			row.id, row.name, row.link, row.spotifyID, row.avatarURL,
			row.descUA, row.descEN,
			row.rawCountries, row.primaryCountry, row.evidenceURL, row.notes, sources,
			row.createdAt, row.updatedAt,
			labelsByArtist[row.id],
		)
	}

	return artists, nil
}

func (r *ArtistRepo) Create(ctx context.Context, input domain.UpsertArtistInput) (*domain.Artist, error) {
	sourcesJSON, err := json.Marshal(input.Sources)
	if err != nil {
		return nil, err
	}

	var (
		id                   int
		createdAt, updatedAt time.Time
	)
	err = r.db.QueryRow(ctx, `
		INSERT INTO artists (name, link, spotify_id, avatar_url, description_ua, description_en,
			countries, primary_country, evidence_url, notes, sources)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, created_at, updated_at
	`,
		input.Name, input.Link, input.SpotifyID, input.AvatarURL,
		input.Description, input.DescriptionEn,
		input.Countries, input.PrimaryCountry, input.EvidenceURL, input.Notes,
		sourcesJSON,
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}

	if err := r.replaceArtistLabels(ctx, id, input.ListenLabels); err != nil {
		return nil, err
	}

	labels, err := r.fetchLabels(ctx, []int{id})
	if err != nil {
		return nil, err
	}

	a := buildArtist(id, input.Name, input.Link, input.SpotifyID, input.AvatarURL,
		input.Description, input.DescriptionEn,
		input.Countries, input.PrimaryCountry, input.EvidenceURL, input.Notes, input.Sources,
		createdAt, updatedAt,
		labels[id])
	return &a, nil
}

func (r *ArtistRepo) Update(ctx context.Context, id string, input domain.UpsertArtistInput) (*domain.Artist, error) {
	sourcesJSON, err := json.Marshal(input.Sources)
	if err != nil {
		return nil, err
	}

	var (
		aid                  int
		createdAt, updatedAt time.Time
	)
	err = r.db.QueryRow(ctx, `
		UPDATE artists SET
			name = $1, link = $2, spotify_id = $3, avatar_url = $4,
			description_ua = $5, description_en = $6, countries = $7,
			primary_country = $8, evidence_url = $9, notes = $10, sources = $11
		WHERE id = $12
		RETURNING id, created_at, updated_at
	`,
		input.Name, input.Link, input.SpotifyID, input.AvatarURL,
		input.Description, input.DescriptionEn, input.Countries,
		input.PrimaryCountry, input.EvidenceURL, input.Notes, sourcesJSON,
		id,
	).Scan(&aid, &createdAt, &updatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}

	if err := r.replaceArtistLabels(ctx, aid, input.ListenLabels); err != nil {
		return nil, err
	}

	labels, err := r.fetchLabels(ctx, []int{aid})
	if err != nil {
		return nil, err
	}

	a := buildArtist(aid, input.Name, input.Link, input.SpotifyID, input.AvatarURL,
		input.Description, input.DescriptionEn,
		input.Countries, input.PrimaryCountry, input.EvidenceURL, input.Notes, input.Sources,
		createdAt, updatedAt,
		labels[aid])
	return &a, nil
}

func (r *ArtistRepo) Delete(ctx context.Context, id string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM artist_labels WHERE artist_id = $1`, id)
	if err != nil {
		return err
	}
	tag, err := r.db.Exec(ctx, `DELETE FROM artists WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *ArtistRepo) GetStats(ctx context.Context) (*domain.ArtistStats, error) {
	var (
		total     int
		lastAdded *string
	)
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*), (SELECT name FROM artists ORDER BY created_at DESC LIMIT 1)
		FROM artists
	`).Scan(&total, &lastAdded)
	if err != nil {
		return nil, err
	}
	return &domain.ArtistStats{Total: total, LastAdded: lastAdded}, nil
}

func (r *ArtistRepo) replaceArtistLabels(ctx context.Context, artistID int, labelIDs []string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM artist_labels WHERE artist_id = $1`, artistID)
	if err != nil {
		return err
	}
	if len(labelIDs) == 0 {
		return nil
	}
	ids := make([]int, 0, len(labelIDs))
	for _, s := range labelIDs {
		n, err := strconv.Atoi(s)
		if err != nil {
			continue
		}
		ids = append(ids, n)
	}
	if len(ids) == 0 {
		return nil
	}
	_, err = r.db.Exec(ctx, `
		INSERT INTO artist_labels (artist_id, label_id)
		SELECT $1, id FROM labels WHERE id = ANY($2)
		ON CONFLICT DO NOTHING
	`, artistID, ids)
	return err
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
			Name:         name,
			OriginalName: originalName,
			Priority:     priority,
			CreatedAt:    createdAt,
			UpdatedAt:    updatedAt,
		})
	}
	return result, rows.Err()
}

func buildArtist(
	id int,
	name string,
	link, spotifyID, avatarURL, descUA, descEN *string,
	countries []string,
	primaryCountry, evidenceURL, notes *string,
	sources []domain.ArtistSource,
	createdAt, updatedAt time.Time,
	labels []domain.Label,
) domain.Artist {
	if labels == nil {
		labels = []domain.Label{}
	}
	if countries == nil {
		countries = []string{}
	}
	if sources == nil {
		sources = []domain.ArtistSource{}
	}
	return domain.Artist{
		ID:             strconv.Itoa(id),
		Name:           name,
		Link:           link,
		SpotifyID:      spotifyID,
		AvatarURL:      avatarURL,
		Countries:      countries,
		ListenLabels:   labels,
		Description:    descUA,
		DescriptionEn:  descEN,
		PrimaryCountry: primaryCountry,
		EvidenceURL:    evidenceURL,
		Notes:          notes,
		Sources:        sources,
		CreatedAt:      createdAt,
		UpdatedAt:      updatedAt,
	}
}

func parseSources(raw []byte) []domain.ArtistSource {
	if len(raw) == 0 {
		return []domain.ArtistSource{}
	}
	var sources []domain.ArtistSource
	if err := json.Unmarshal(raw, &sources); err != nil {
		return []domain.ArtistSource{}
	}
	return sources
}

func extractSpotifyID(s string) string {
	const urlPrefix = "open.spotify.com/artist/"
	if idx := strings.Index(s, urlPrefix); idx != -1 {
		rest := s[idx+len(urlPrefix):]
		if i := strings.IndexAny(rest, "?/"); i != -1 {
			rest = rest[:i]
		}
		return rest
	}
	const uriPrefix = "spotify:artist:"
	if strings.HasPrefix(s, uriPrefix) {
		id := s[len(uriPrefix):]
		if i := strings.Index(id, "?"); i != -1 {
			id = id[:i]
		}
		return id
	}
	return ""
}

func safeDir(d domain.SortDirection) string {
	if d == domain.SortDesc {
		return "DESC"
	}
	return "ASC"
}
