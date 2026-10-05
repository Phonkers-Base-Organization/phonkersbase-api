package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PhonkersBase/base-api2/internal/domain"
)

type OrganisationMusoRepo struct {
	db *pgxpool.Pool
}

type OrganisationMusoSweepTarget struct {
	ID   int
	Name string
}

func NewOrganisationMusoRepo(db *pgxpool.Pool) *OrganisationMusoRepo {
	return &OrganisationMusoRepo{db: db}
}

func (r *OrganisationMusoRepo) GetOrganisationsForSweep(ctx context.Context) ([]OrganisationMusoSweepTarget, error) {
	rows, err := r.db.Query(ctx, `-- name: organisation_muso.get_organisations_for_sweep
		SELECT o.id, o.name
		FROM organisations o
		LEFT JOIN organisation_muso om ON om.organisation_id = o.id
		WHERE o.origin <> ALL($1)
		ORDER BY om.last_checked_at ASC NULLS FIRST, o.id ASC
	`, domain.MusoExcludedOrigins())
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	targets := make([]OrganisationMusoSweepTarget, 0)
	for rows.Next() {
		var target OrganisationMusoSweepTarget
		if err := rows.Scan(&target.ID, &target.Name); err != nil {
			return nil, err
		}
		targets = append(targets, target)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return targets, nil
}

func (r *OrganisationMusoRepo) GetOrganisationMuso(ctx context.Context, organisationID int) (*domain.OrganisationMuso, error) {
	var state domain.OrganisationMuso
	err := r.db.QueryRow(ctx, `-- name: organisation_muso.get
		SELECT id, organisation_id, muso_profile_id, collaborators_count, credit_count,
		       roster_size, last_checked_at, last_synced_at, sync_status, sync_error,
		       created_at, updated_at
		FROM organisation_muso
		WHERE organisation_id = $1
	`, organisationID).Scan(
		&state.ID, &state.OrganisationID, &state.MusoProfileID,
		&state.CollaboratorsCount, &state.CreditCount, &state.RosterSize,
		&state.LastCheckedAt, &state.LastSyncedAt, &state.SyncStatus,
		&state.SyncError, &state.CreatedAt, &state.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &state, nil
}

// ListStates returns every stored muso sync state, for the admin panel's view of
// what the last sweep managed to fetch.
func (r *OrganisationMusoRepo) ListStates(ctx context.Context) ([]domain.OrganisationMuso, error) {
	rows, err := r.db.Query(ctx, `-- name: organisation_muso.list_states
		SELECT id, organisation_id, muso_profile_id, collaborators_count, credit_count,
		       roster_size, last_checked_at, last_synced_at, sync_status, sync_error,
		       created_at, updated_at
		FROM organisation_muso
		ORDER BY organisation_id
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	states := []domain.OrganisationMuso{}
	for rows.Next() {
		var state domain.OrganisationMuso
		err := rows.Scan(
			&state.ID, &state.OrganisationID, &state.MusoProfileID,
			&state.CollaboratorsCount, &state.CreditCount, &state.RosterSize,
			&state.LastCheckedAt, &state.LastSyncedAt, &state.SyncStatus,
			&state.SyncError, &state.CreatedAt, &state.UpdatedAt,
		)
		if err != nil {
			return nil, err
		}
		states = append(states, state)
	}
	err = rows.Err()
	if err != nil {
		return nil, err
	}
	return states, nil
}

func (r *OrganisationMusoRepo) UpsertOrganisationMuso(ctx context.Context, state *domain.OrganisationMuso) error {
	return r.db.QueryRow(ctx, `-- name: organisation_muso.upsert
		INSERT INTO organisation_muso (
			organisation_id, muso_profile_id, collaborators_count, credit_count,
			roster_size, last_checked_at, last_synced_at, sync_status, sync_error
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (organisation_id) DO UPDATE SET
			muso_profile_id = EXCLUDED.muso_profile_id,
			collaborators_count = EXCLUDED.collaborators_count,
			credit_count = EXCLUDED.credit_count,
			roster_size = EXCLUDED.roster_size,
			last_checked_at = EXCLUDED.last_checked_at,
			last_synced_at = EXCLUDED.last_synced_at,
			sync_status = EXCLUDED.sync_status,
			sync_error = EXCLUDED.sync_error
		RETURNING id, created_at, updated_at
	`, state.OrganisationID, state.MusoProfileID, state.CollaboratorsCount,
		state.CreditCount, state.RosterSize, state.LastCheckedAt, state.LastSyncedAt,
		state.SyncStatus, state.SyncError,
	).Scan(&state.ID, &state.CreatedAt, &state.UpdatedAt)
}

func (r *OrganisationMusoRepo) ResolveArtistIDsBySpotifyIDs(ctx context.Context, spotifyIDs []string) (map[string]int, error) {
	result := make(map[string]int, len(spotifyIDs))
	if len(spotifyIDs) == 0 {
		return result, nil
	}

	rows, err := r.db.Query(ctx, `-- name: organisation_muso.resolve_artist_ids
		SELECT spotify_id, id
		FROM artists
		WHERE spotify_id = ANY($1)
	`, spotifyIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var spotifyID string
		var artistID int
		if err := rows.Scan(&spotifyID, &artistID); err != nil {
			return nil, err
		}
		result[spotifyID] = artistID
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *OrganisationMusoRepo) ReplaceOrganisationArtists(ctx context.Context, organisationID int, artistProfiles map[int]string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `-- name: organisation_muso.replace_artists.delete
		DELETE FROM organisation_artists
		WHERE organisation_id = $1
	`, organisationID)
	if err != nil {
		return err
	}

	if len(artistProfiles) > 0 {
		artistIDs := make([]int, 0, len(artistProfiles))
		profileIDs := make([]string, 0, len(artistProfiles))
		for artistID, profileID := range artistProfiles {
			artistIDs = append(artistIDs, artistID)
			profileIDs = append(profileIDs, profileID)
		}
		_, err = tx.Exec(ctx, `-- name: organisation_muso.replace_artists.insert
			INSERT INTO organisation_artists (organisation_id, artist_id, muso_profile_id)
			SELECT $1, pairs.artist_id, pairs.muso_profile_id
			FROM unnest($2::int[], $3::text[]) AS pairs(artist_id, muso_profile_id)
		`, organisationID, artistIDs, profileIDs)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// Listen labels as stored in labels.name. pride and base only ever accompany
// approved, so the chart folds all three into one bucket.
const (
	listenLabelBlocked = "blocked"
	listenLabelWarning = "warning"
)

var listenLabelsApproved = []string{"approved", "pride", "base"}

// RosterStatsFor returns, for a whole page of organisations in one query, how the
// roster artists pb holds split by listen label and by country.
func (r *OrganisationMusoRepo) RosterStatsFor(ctx context.Context, orgIDs []int) (map[int]domain.RosterStats, error) {
	rosterStats := make(map[int]domain.RosterStats, len(orgIDs))
	if len(orgIDs) == 0 {
		return rosterStats, nil
	}

	rows, err := r.db.Query(ctx, `-- name: organisation_muso.roster_stats_for
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
		)
		SELECT
			om.organisation_id,
			COUNT(ro.artist_id),
			om.roster_size,
			om.last_synced_at,
			-- An artist can carry several labels (approved+warning is common), so each
			-- one lands in the most severe bucket it qualifies for: blocked, then
			-- warning, then approved. Whatever is left, unlabelled artists included,
			-- is unknown.
			COUNT(*) FILTER (WHERE ro.blocked),
			COUNT(*) FILTER (WHERE ro.warning AND NOT ro.blocked),
			COUNT(*) FILTER (WHERE ro.approved AND NOT ro.warning AND NOT ro.blocked),
			COUNT(*) FILTER (WHERE ro.artist_id IS NOT NULL AND NOT ro.approved AND NOT ro.warning AND NOT ro.blocked),
			COUNT(*) FILTER (WHERE pc.code = ANY($5)),
			COUNT(*) FILTER (WHERE pc.code = $6),
			COUNT(*) FILTER (WHERE pc.code <> ALL($5) AND pc.code <> $6),
			COUNT(*) FILTER (WHERE ro.artist_id IS NOT NULL AND pc.code IS NULL)
		FROM organisation_muso om
		JOIN organisations o ON o.id = om.organisation_id
		LEFT JOIN roster ro ON ro.organisation_id = om.organisation_id
		LEFT JOIN primary_country pc ON pc.artist_id = ro.artist_id
		-- An empty roster is nothing to chart, so it reads as absent rather than as
		-- stats full of zeroes. Excluded origins read as absent too, including any
		-- organisation synced before its origin was set to one of them.
		WHERE om.organisation_id = ANY($1) AND om.roster_size > 0 AND o.origin <> ALL($7)
		GROUP BY om.organisation_id, om.roster_size, om.last_synced_at
	`, orgIDs, listenLabelBlocked, listenLabelWarning, listenLabelsApproved,
		[]string{domain.CountryRussia, domain.CountryBelarus}, domain.CountryUkraine,
		domain.MusoExcludedOrigins())
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
			&stats.SyncedAt,
			&stats.ByStatus.Blocked,
			&stats.ByStatus.Warning,
			&stats.ByStatus.Approved,
			&stats.ByStatus.Unknown,
			&stats.ByCountry.RuBy,
			&stats.ByCountry.Ukraine,
			&stats.ByCountry.Other,
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
