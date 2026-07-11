package repository

import (
	"context"
	"errors"
	"fmt"
	"math"
	"regexp"
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
		notes                     *string
		name                      string
		createdAt, updatedAt      time.Time
	)
	err := r.db.QueryRow(ctx, `
		-- name: artist.get_by_id
		SELECT id, name, link, spotify_id, avatar_url,
		       description_ua, description_en,
		       notes, created_at, updated_at
		FROM artists
		WHERE id = $1
	`, id).Scan(
		&aid, &name, &link, &spotifyID, &avatarURL,
		&descUA, &descEN,
		&notes, &createdAt, &updatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if errors.Is(err, context.Canceled) {
			return nil, context.Canceled
		}
		return nil, err
	}

	labels, err := r.fetchLabels(ctx, []int{aid})
	if err != nil {
		return nil, err
	}
	countries, err := r.fetchCountries(ctx, []int{aid})
	if err != nil {
		return nil, err
	}

	a := buildArtist(aid, name, link, spotifyID, avatarURL, descUA, descEN,
		countries[aid], notes,
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
		if cond := buildSearchCondition(p.Search, []string{descCol}, &n, &args); cond != "" {
			conditions = append(conditions, cond)
		}
	}
	if len(p.Countries) > 0 {
		conditions = append(conditions, fmt.Sprintf(
			"EXISTS (SELECT 1 FROM artist_countries ac2 WHERE ac2.artist_id = a.id AND ac2.code = ANY($%d))", n,
		))
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
		orderParts = append(orderParts, fmt.Sprintf(
			"(SELECT ac3.code FROM artist_countries ac3 WHERE ac3.artist_id = a.id ORDER BY ac3.position ASC LIMIT 1) %s NULLS %s",
			safeDir(p.ByCountry), nullsPos,
		))
	}
	if p.ByListen != "" {
		orderParts = append(orderParts, "a.total_priority "+safeDir(p.ByListen))
	}
	if len(orderParts) == 0 {
		orderParts = []string{"a.total_priority DESC, a.updated_at DESC"}
	}
	orderParts = append(orderParts, "a.id ASC")
	orderClause := "ORDER BY " + strings.Join(orderParts, ", ")

	limit := p.Limit
	if limit <= 0 {
		limit = 50
	}

	q := fmt.Sprintf(`
		-- name: artist.get_all
		SELECT
			a.id, a.name, a.link, a.spotify_id, a.avatar_url,
			a.description_ua, a.description_en,
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
			&row.descUA, &row.descEN,
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

	labelsByArtist := map[int][]domain.LabelRef{}
	countriesByArtist := map[int][]domain.ArtistCountry{}
	if len(artistIDs) > 0 {
		labelsByArtist, err = r.fetchLabels(ctx, artistIDs)
		if err != nil {
			return nil, err
		}
		countriesByArtist, err = r.fetchCountries(ctx, artistIDs)
		if err != nil {
			return nil, err
		}
	}

	artists := make([]domain.Artist, len(raws))
	for i, row := range raws {
		artists[i] = buildArtist(
			row.id, row.name, row.link, row.spotifyID, row.avatarURL,
			row.descUA, row.descEN,
			countriesByArtist[row.id], nil,
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

func (r *ArtistRepo) GetAdminAll(ctx context.Context, limit, offset int, search string) ([]domain.Artist, int, error) {
	args := []any{}
	conditions := []string{}
	n := 1

	if search != "" {
		if cond := buildSearchCondition(search, []string{"a.description_ua", "a.description_en"}, &n, &args); cond != "" {
			conditions = append(conditions, cond)
		}
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	limitClause := ""
	if limit > 0 {
		limitClause = fmt.Sprintf("LIMIT $%d OFFSET $%d", n, n+1)
		args = append(args, limit, offset)
	}

	q := fmt.Sprintf(`
		-- name: artist.get_admin_all
		SELECT
			a.id, a.name, a.link, a.spotify_id, a.avatar_url,
			a.description_ua, a.description_en,
			a.notes, a.created_at, a.updated_at,
			COUNT(*) OVER() AS total_count
		FROM artists a
		%s
		ORDER BY a.updated_at DESC, a.id ASC
		%s
	`, whereClause, limitClause)

	rows, err := r.db.Query(ctx, q, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	type rawRow struct {
		id                         int
		name                       string
		link, spotifyID, avatarURL *string
		descUA, descEN             *string
		notes                      *string
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
			&row.descUA, &row.descEN,
			&row.notes, &row.createdAt, &row.updatedAt,
			&row.totalCount,
		); err != nil {
			return nil, 0, err
		}
		total = row.totalCount
		raws = append(raws, row)
		artistIDs = append(artistIDs, row.id)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}

	labelsByArtist := map[int][]domain.LabelRef{}
	countriesByArtist := map[int][]domain.ArtistCountry{}
	if len(artistIDs) > 0 {
		labelsByArtist, err = r.fetchLabels(ctx, artistIDs)
		if err != nil {
			return nil, 0, err
		}
		countriesByArtist, err = r.fetchCountries(ctx, artistIDs)
		if err != nil {
			return nil, 0, err
		}
	}

	artists := make([]domain.Artist, len(raws))
	for i, row := range raws {
		artists[i] = buildArtist(
			row.id, row.name, row.link, row.spotifyID, row.avatarURL,
			row.descUA, row.descEN,
			countriesByArtist[row.id], row.notes,
			row.createdAt, row.updatedAt,
			labelsByArtist[row.id],
		)
	}

	return artists, total, nil
}

func (r *ArtistRepo) Create(ctx context.Context, input domain.UpsertArtistInput) (*domain.Artist, error) {
	var (
		id                   int
		createdAt, updatedAt time.Time
	)
	err := r.db.QueryRow(ctx, `
		-- name: artist.create
		INSERT INTO artists (name, link, spotify_id, avatar_url, description_ua, description_en, notes)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at, updated_at
	`,
		input.Name, input.Link, input.SpotifyID, input.AvatarURL,
		input.Description, input.DescriptionEn, input.Notes,
	).Scan(&id, &createdAt, &updatedAt)
	if err != nil {
		return nil, err
	}

	if err := r.replaceArtistLabels(ctx, id, input.ListenLabels); err != nil {
		return nil, err
	}
	if err := r.replaceArtistCountries(ctx, id, input.Countries); err != nil {
		return nil, err
	}

	labels, err := r.fetchLabels(ctx, []int{id})
	if err != nil {
		return nil, err
	}
	countries, err := r.fetchCountries(ctx, []int{id})
	if err != nil {
		return nil, err
	}

	a := buildArtist(id, input.Name, input.Link, input.SpotifyID, input.AvatarURL,
		input.Description, input.DescriptionEn,
		countries[id], input.Notes,
		createdAt, updatedAt,
		labels[id])
	return &a, nil
}

func (r *ArtistRepo) Update(ctx context.Context, id string, input domain.UpsertArtistInput) (*domain.Artist, error) {
	var (
		aid                  int
		createdAt, updatedAt time.Time
	)
	err := r.db.QueryRow(ctx, `
		-- name: artist.update
		UPDATE artists SET
			name = $1, link = $2, spotify_id = $3, avatar_url = $4,
			description_ua = $5, description_en = $6, notes = $7
		WHERE id = $8
		RETURNING id, created_at, updated_at
	`,
		input.Name, input.Link, input.SpotifyID, input.AvatarURL,
		input.Description, input.DescriptionEn,
		input.Notes, id,
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
	if err := r.replaceArtistCountries(ctx, aid, input.Countries); err != nil {
		return nil, err
	}

	labels, err := r.fetchLabels(ctx, []int{aid})
	if err != nil {
		return nil, err
	}
	countries, err := r.fetchCountries(ctx, []int{aid})
	if err != nil {
		return nil, err
	}

	a := buildArtist(aid, input.Name, input.Link, input.SpotifyID, input.AvatarURL,
		input.Description, input.DescriptionEn,
		countries[aid], input.Notes,
		createdAt, updatedAt,
		labels[aid])
	return &a, nil
}

func (r *ArtistRepo) Delete(ctx context.Context, id string) error {
	// junction rows deleted by ON DELETE CASCADE
	tag, err := r.db.Exec(ctx, `
		-- name: artist.delete
		DELETE FROM artists WHERE id = $1
	`, id)
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
		-- name: artist.get_stats
		SELECT COUNT(*), (SELECT name FROM artists ORDER BY created_at DESC LIMIT 1)
		FROM artists
	`).Scan(&total, &lastAdded)
	if err != nil {
		return nil, err
	}
	return &domain.ArtistStats{Total: total, LastAdded: lastAdded}, nil
}

func (r *ArtistRepo) replaceArtistCountries(ctx context.Context, artistID int, countries []domain.ArtistCountryInput) error {
	_, err := r.db.Exec(ctx, `
		-- name: artist.replace_countries.delete
		DELETE FROM artist_countries WHERE artist_id = $1
	`, artistID)
	if err != nil {
		return err
	}
	for i, c := range countries {
		_, err := r.db.Exec(ctx, `
			-- name: artist.replace_countries.insert_country
			INSERT INTO artist_countries (artist_id, code, position)
			VALUES ($1, $2, $3)
			ON CONFLICT (artist_id, code) DO NOTHING
		`, artistID, c.Code, i)
		if err != nil {
			return err
		}
		for _, sid := range c.SourceIDs {
			n, err := strconv.Atoi(sid)
			if err != nil {
				continue
			}
			_, err = r.db.Exec(ctx, `
				-- name: artist.replace_countries.insert_source
				INSERT INTO artist_country_sources (artist_id, code, source_id)
				VALUES ($1, $2, $3)
				ON CONFLICT DO NOTHING
			`, artistID, c.Code, n)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (r *ArtistRepo) replaceArtistLabels(ctx context.Context, artistID int, labelIDs []string) error {
	_, err := r.db.Exec(ctx, `
		-- name: artist.replace_labels.delete
		DELETE FROM artist_labels WHERE artist_id = $1
	`, artistID)
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
		-- name: artist.replace_labels.insert
		INSERT INTO artist_labels (artist_id, label_id)
		SELECT $1, id FROM labels WHERE id = ANY($2)
		ON CONFLICT DO NOTHING
	`, artistID, ids)
	return err
}

func (r *ArtistRepo) fetchCountries(ctx context.Context, artistIDs []int) (map[int][]domain.ArtistCountry, error) {
	rows, err := r.db.Query(ctx, `
		-- name: artist.fetch_countries
		SELECT ac.artist_id, ac.code, es.id, es.name, es.name_en
		FROM artist_countries ac
		LEFT JOIN artist_country_sources acs ON acs.artist_id = ac.artist_id AND acs.code = ac.code
		LEFT JOIN evidence_sources es ON es.id = acs.source_id
		WHERE ac.artist_id = ANY($1)
		ORDER BY ac.position ASC, acs.source_id ASC
	`, artistIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type countryKey struct{ artistID int; code string }
	// maintain insertion order for positions
	order := []countryKey{}
	seen := map[countryKey]bool{}
	result := map[int][]domain.ArtistCountry{}
	sources := map[countryKey][]domain.SourceRef{}

	for rows.Next() {
		var (
			artistID     int
			code         string
			sourceID     *int
			sourceName   *string
			sourceNameEn *string
		)
		if err := rows.Scan(&artistID, &code, &sourceID, &sourceName, &sourceNameEn); err != nil {
			return nil, err
		}
		k := countryKey{artistID, code}
		if !seen[k] {
			seen[k] = true
			order = append(order, k)
			sources[k] = []domain.SourceRef{}
		}
		if sourceID != nil {
			sources[k] = append(sources[k], domain.SourceRef{
				ID:     strconv.Itoa(*sourceID),
				Name:   *sourceName,
				NameEn: derefStr(sourceNameEn),
			})
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, k := range order {
		srcs := sources[k]
		if srcs == nil {
			srcs = []domain.SourceRef{}
		}
		result[k.artistID] = append(result[k.artistID], domain.ArtistCountry{Code: k.code, Sources: srcs})
	}
	return result, nil
}

func (r *ArtistRepo) fetchLabels(ctx context.Context, artistIDs []int) (map[int][]domain.LabelRef, error) {
	rows, err := r.db.Query(ctx, `
		-- name: artist.fetch_labels
		SELECT al.artist_id, l.id, l.name
		FROM labels l
		JOIN artist_labels al ON al.label_id = l.id
		WHERE al.artist_id = ANY($1)
		ORDER BY l.priority DESC, l.name ASC
	`, artistIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[int][]domain.LabelRef{}
	for rows.Next() {
		var (
			artistID, id int
			name         string
		)
		if err := rows.Scan(&artistID, &id, &name); err != nil {
			return nil, err
		}
		result[artistID] = append(result[artistID], domain.LabelRef{
			ID:   strconv.Itoa(id),
			Name: name,
		})
	}
	return result, rows.Err()
}

func buildArtist(
	id int,
	name string,
	link, spotifyID, avatarURL, descUA, descEN *string,
	countries []domain.ArtistCountry,
	notes *string,
	createdAt, updatedAt time.Time,
	labels []domain.LabelRef,
) domain.Artist {
	if labels == nil {
		labels = []domain.LabelRef{}
	}
	if countries == nil {
		countries = []domain.ArtistCountry{}
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
		Notes:         notes,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}

// spotifyIDPattern matches a bare Spotify ID (base62, 22 chars) with no surrounding
// URL/URI, e.g. as passed by external scrapers checking artist existence in bulk.
var spotifyIDPattern = regexp.MustCompile(`^[0-9A-Za-z]{22}$`)

// buildSearchCondition builds the OR-of-terms WHERE clause for a comma-separated artist
// search string. Bare Spotify IDs are routed to a single indexed `spotify_id = ANY(...)`
// lookup instead of the unindexed ILIKE substring scan, since an artist ID never legitimately
// appears as a substring of its own name/description/link.
func buildSearchCondition(search string, descCols []string, n *int, args *[]any) string {
	var termClauses []string
	var spotifyIDs []string
	for term := range strings.SplitSeq(search, ",") {
		term = strings.TrimSpace(term)
		if term == "" {
			continue
		}
		if spotifyIDPattern.MatchString(term) {
			spotifyIDs = append(spotifyIDs, term)
			continue
		}
		searchBase := term
		if i := strings.Index(searchBase, "?"); i != -1 {
			searchBase = searchBase[:i]
		}
		orClauses := []string{fmt.Sprintf("a.name ILIKE $%d", *n)}
		for _, col := range descCols {
			orClauses = append(orClauses, fmt.Sprintf("%s ILIKE $%d", col, *n))
		}
		orClauses = append(orClauses, fmt.Sprintf("SPLIT_PART(a.link, '?', 1) ILIKE $%d", *n))
		*args = append(*args, "%"+escapeLikePattern(searchBase)+"%")
		*n++
		if spotifyID := extractSpotifyID(term); spotifyID != "" {
			orClauses = append(orClauses, fmt.Sprintf("a.spotify_id = $%d", *n))
			*args = append(*args, spotifyID)
			*n++
		}
		termClauses = append(termClauses, "("+strings.Join(orClauses, " OR ")+")")
	}
	if len(spotifyIDs) > 0 {
		termClauses = append(termClauses, fmt.Sprintf("a.spotify_id = ANY($%d)", *n))
		*args = append(*args, spotifyIDs)
		*n++
	}
	if len(termClauses) == 0 {
		return ""
	}
	return "(" + strings.Join(termClauses, " OR ") + ")"
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

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
