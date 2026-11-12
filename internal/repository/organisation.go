package repository

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PhonkersBase/base-api2/internal/domain"
)

type OrganisationRepo struct {
	db *pgxpool.Pool
}

func NewOrganisationRepo(db *pgxpool.Pool) *OrganisationRepo {
	return &OrganisationRepo{db: db}
}

func (r *OrganisationRepo) GetAll(ctx context.Context, params domain.ListOrganisationsParams) (*domain.PaginatedOrganisations, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	var total int
	err := r.db.QueryRow(ctx, `
		-- name: organisation.get_all.count
		SELECT COUNT(*) FROM organisations
		WHERE ($1 = '' OR type = $1)
		  AND ($2 = '' OR name ILIKE '%' || $2 || '%')
	`, params.Type, escapeLikePattern(params.Search)).Scan(&total)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
		-- name: organisation.get_all.items
		SELECT id, name, link, origin, description_uk, description_en, notes, type, recommendation, created_at, updated_at
		FROM organisations
		WHERE ($1 = '' OR type = $1)
		  AND ($2 = '' OR name ILIKE '%' || $2 || '%')
		ORDER BY name ASC
		LIMIT $3 OFFSET $4
	`, params.Type, escapeLikePattern(params.Search), limit, params.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.Organisation{}
	orgIDs := []int{}
	for rows.Next() {
		var (
			id int
			o  domain.Organisation
		)
		if err := rows.Scan(&id, &o.Name, &o.Link, &o.Origin, &o.DescriptionUk, &o.DescriptionEn, &o.Notes, &o.Type, &o.Recommendation, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		o.ID = strconv.Itoa(id)
		items = append(items, o)
		orgIDs = append(orgIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	rosterStats, err := r.rosterStatsFor(ctx, orgIDs)
	if err != nil {
		return nil, err
	}
	for i, id := range orgIDs {
		if stats, ok := rosterStats[id]; ok {
			items[i].RosterStats = &stats
		}
	}

	totalPages := total / limit
	if total%limit != 0 {
		totalPages++
	}
	currentPage := params.Offset/limit + 1

	return &domain.PaginatedOrganisations{
		Items: items,
		Info: domain.Pagination{
			Limit:       limit,
			Offset:      params.Offset,
			Total:       total,
			TotalPages:  totalPages,
			CurrentPage: currentPage,
		},
	}, nil
}

func (r *OrganisationRepo) GetByID(ctx context.Context, id string) (*domain.Organisation, error) {
	var o domain.Organisation
	var oid int
	err := r.db.QueryRow(ctx, `
		-- name: organisation.get_by_id
		SELECT id, name, link, origin, description_uk, description_en, notes, type, recommendation, created_at, updated_at
		FROM organisations
		WHERE id = $1
	`, id).Scan(
		&oid, &o.Name, &o.Link, &o.Origin, &o.DescriptionUk, &o.DescriptionEn, &o.Notes, &o.Type, &o.Recommendation, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	o.ID = strconv.Itoa(oid)
	rosterStats, err := r.rosterStatsFor(ctx, []int{oid})
	if err != nil {
		return nil, err
	}
	if stats, ok := rosterStats[oid]; ok {
		o.RosterStats = &stats
	}
	return &o, nil
}

func (r *OrganisationRepo) Create(ctx context.Context, input domain.UpsertOrganisationInput) (*domain.Organisation, error) {
	var o domain.Organisation
	var id int
	err := r.db.QueryRow(ctx, `
		-- name: organisation.create
		INSERT INTO organisations (name, link, origin, description_uk, description_en, notes, type, recommendation)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, name, link, origin, description_uk, description_en, notes, type, recommendation, created_at, updated_at
	`, input.Name, input.Link, input.Origin, input.DescriptionUk, input.DescriptionEn, input.Notes, input.Type, input.Recommendation).Scan(
		&id, &o.Name, &o.Link, &o.Origin, &o.DescriptionUk, &o.DescriptionEn, &o.Notes, &o.Type, &o.Recommendation, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	o.ID = strconv.Itoa(id)
	return &o, nil
}

func (r *OrganisationRepo) Update(ctx context.Context, id string, input domain.UpsertOrganisationInput) (*domain.Organisation, error) {
	var o domain.Organisation
	var oid int
	err := r.db.QueryRow(ctx, `
		-- name: organisation.update
		UPDATE organisations
		SET name = $1, link = $2, origin = $3, description_uk = $4, description_en = $5, notes = $6, type = $7, recommendation = $8
		WHERE id = $9
		RETURNING id, name, link, origin, description_uk, description_en, notes, type, recommendation, created_at, updated_at
	`, input.Name, input.Link, input.Origin, input.DescriptionUk, input.DescriptionEn, input.Notes, input.Type, input.Recommendation, id).Scan(
		&oid, &o.Name, &o.Link, &o.Origin, &o.DescriptionUk, &o.DescriptionEn, &o.Notes, &o.Type, &o.Recommendation, &o.CreatedAt, &o.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	o.ID = strconv.Itoa(oid)
	return &o, nil
}

func (r *OrganisationRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `
		-- name: organisation.delete
		DELETE FROM organisations WHERE id = $1
	`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

const pgForeignKeyViolation = "23503"

var ErrUnknownArtist = errors.New("unknown artist")

func (r *OrganisationRepo) GetArtists(ctx context.Context, id string) ([]domain.ArtistRef, error) {
	orgID, err := strconv.Atoi(id)
	if err != nil {
		return nil, ErrNotFound
	}
	return r.listArtists(ctx, r.db, orgID)
}

type artistLister interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

func (r *OrganisationRepo) listArtists(ctx context.Context, q artistLister, orgID int) ([]domain.ArtistRef, error) {
	rows, err := q.Query(ctx, `
		-- name: organisation.get_artists
		SELECT a.id, a.name, a.avatar_url, a.spotify_id
		FROM organisations o
		LEFT JOIN organisation_artists oa ON oa.organisation_id = o.id
		LEFT JOIN artists a ON a.id = oa.artist_id
		WHERE o.id = $1
		ORDER BY lower(a.name), a.id
	`, orgID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	artists := []domain.ArtistRef{}
	found := false
	for rows.Next() {
		found = true
		var aid *int
		var name *string
		var a domain.ArtistRef
		err := rows.Scan(&aid, &name, &a.AvatarURL, &a.SpotifyID)
		if err != nil {
			return nil, err
		}
		if aid == nil {
			continue
		}
		a.ID = strconv.Itoa(*aid)
		a.Name = *name
		artists = append(artists, a)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	if !found {
		return nil, ErrNotFound
	}
	return artists, nil
}

func (r *OrganisationRepo) ReplaceArtists(ctx context.Context, id string, artistIDs []int) ([]domain.ArtistRef, error) {
	orgID, err := strconv.Atoi(id)
	if err != nil {
		return nil, ErrNotFound
	}
	if artistIDs == nil {
		artistIDs = []int{}
	}

	var artists []domain.ArtistRef
	err = pgx.BeginFunc(ctx, r.db, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, `
			-- name: organisation.replace_artists
			WITH removed AS (
				DELETE FROM organisation_artists
				WHERE organisation_id = $1 AND NOT (artist_id = ANY($2))
			)
			INSERT INTO organisation_artists (organisation_id, artist_id)
			SELECT $1, unnest($2::int[])
			ON CONFLICT DO NOTHING
		`, orgID, artistIDs)
		if err != nil {
			return err
		}
		artists, err = r.listArtists(ctx, tx, orgID)
		return err
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgForeignKeyViolation {
			return nil, ErrUnknownArtist
		}
		return nil, err
	}
	return artists, nil
}

// Listen labels as stored in labels.name. pride and base only ever accompany
// approved, so the chart folds all three into one bucket.
const (
	listenLabelBlocked = "blocked"
	listenLabelWarning = "warning"
)

var listenLabelsApproved = []string{"approved", "pride", "base"}

// rosterStatsFor returns, for a whole page of organisations in one query, how the
// artists on each organisation's list split by listen label and by country.
// Organisations with an empty list and no recorded roster size are absent from the result.
func (r *OrganisationRepo) rosterStatsFor(ctx context.Context, orgIDs []int) (map[int]domain.RosterStats, error) {
	rosterStats := make(map[int]domain.RosterStats, len(orgIDs))
	if len(orgIDs) == 0 {
		return rosterStats, nil
	}

	rows, err := r.db.Query(ctx, `-- name: organisation.roster_stats
		WITH roster AS (
			SELECT
				oa.organisation_id,
				oa.artist_id,
				COALESCE(bool_or(l.name = $2), false)      AS blocked,
				COALESCE(bool_or(l.name = $3), false)      AS warning,
				COALESCE(bool_or(l.name = ANY($4)), false) AS approved
			FROM organisation_artists oa
			LEFT JOIN artist_labels al ON al.artist_id = oa.artist_id
			LEFT JOIN labels l ON l.id = al.label_id
			WHERE oa.organisation_id = ANY($1)
			GROUP BY oa.organisation_id, oa.artist_id
		),
		primary_country AS (
			-- An artist's primary country is its lowest position, then code so ties
			-- resolve the same way every time.
			SELECT DISTINCT ON (ac.artist_id) ac.artist_id, ac.code
			FROM artist_countries ac
			WHERE ac.artist_id IN (SELECT artist_id FROM roster)
			ORDER BY ac.artist_id, ac.position, ac.code
		),
		country_counts AS (
			SELECT ro.organisation_id, pc.code, COUNT(*) AS artists
			FROM roster ro
			JOIN primary_country pc ON pc.artist_id = ro.artist_id
			GROUP BY ro.organisation_id, pc.code
		),
		country_lists AS (
			SELECT
				organisation_id,
				json_agg(json_build_object('code', code, 'count', artists) ORDER BY artists DESC, code) AS countries
			FROM country_counts
			GROUP BY organisation_id
		),
		totals AS (
			SELECT
				o.id AS organisation_id,
				COUNT(ro.artist_id) AS matched,
				GREATEST(o.roster_size, COUNT(ro.artist_id)) AS roster_size,
				-- An artist can carry several labels (approved+warning is common), so
				-- each one lands in the most severe bucket it qualifies for: blocked,
				-- then warning, then approved. Whatever is left, unlabelled artists
				-- included, is unknown.
				COUNT(*) FILTER (WHERE ro.blocked) AS blocked,
				COUNT(*) FILTER (WHERE ro.warning AND NOT ro.blocked) AS warning,
				COUNT(*) FILTER (WHERE ro.approved AND NOT ro.warning AND NOT ro.blocked) AS approved,
				COUNT(*) FILTER (WHERE ro.artist_id IS NOT NULL AND NOT ro.approved AND NOT ro.warning AND NOT ro.blocked) AS status_unknown,
				COUNT(*) FILTER (WHERE ro.artist_id IS NOT NULL AND pc.code IS NULL) AS country_unknown
			FROM organisations o
			LEFT JOIN roster ro ON ro.organisation_id = o.id
			LEFT JOIN primary_country pc ON pc.artist_id = ro.artist_id
			-- An empty list with no recorded roster size is nothing to chart, so it reads as
			-- absent rather than as stats full of zeroes. Excluded origins read as
			-- absent too, including any organisation synced before its origin was
			-- set to one of them.
			WHERE o.id = ANY($1) AND o.origin <> ALL($5)
			GROUP BY o.id
			HAVING COUNT(ro.artist_id) > 0 OR o.roster_size > 0
		)
		SELECT
			t.organisation_id, t.matched, t.roster_size,
			t.blocked, t.warning, t.approved, t.status_unknown,
			COALESCE(cl.countries, '[]'::json),
			t.country_unknown
		FROM totals t
		LEFT JOIN country_lists cl ON cl.organisation_id = t.organisation_id
	`, orgIDs, listenLabelBlocked, listenLabelWarning, listenLabelsApproved, domain.UnchartedOrigins())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var organisationID int
		var stats domain.RosterStats
		err := rows.Scan(
			&organisationID,
			&stats.Matched,
			&stats.RosterSize,
			&stats.ByStatus.Blocked,
			&stats.ByStatus.Warning,
			&stats.ByStatus.Approved,
			&stats.ByStatus.Unknown,
			&stats.ByCountry.Countries,
			&stats.ByCountry.Unknown,
		)
		if err != nil {
			return nil, err
		}
		rosterStats[organisationID] = stats
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return rosterStats, nil
}
