package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PhonkersBase/base-api2/internal/domain"
)

type OrganisationMusoRepo struct {
	db *pgxpool.Pool
}

func NewOrganisationMusoRepo(db *pgxpool.Pool) *OrganisationMusoRepo {
	return &OrganisationMusoRepo{db: db}
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
				om.organisation_id,
				COUNT(ro.artist_id) AS matched,
				om.roster_size,
				om.last_synced_at,
				-- An artist can carry several labels (approved+warning is common), so
				-- each one lands in the most severe bucket it qualifies for: blocked,
				-- then warning, then approved. Whatever is left, unlabelled artists
				-- included, is unknown.
				COUNT(*) FILTER (WHERE ro.blocked) AS blocked,
				COUNT(*) FILTER (WHERE ro.warning AND NOT ro.blocked) AS warning,
				COUNT(*) FILTER (WHERE ro.approved AND NOT ro.warning AND NOT ro.blocked) AS approved,
				COUNT(*) FILTER (WHERE ro.artist_id IS NOT NULL AND NOT ro.approved AND NOT ro.warning AND NOT ro.blocked) AS status_unknown,
				COUNT(*) FILTER (WHERE ro.artist_id IS NOT NULL AND pc.code IS NULL) AS country_unknown
			FROM organisation_muso om
			JOIN organisations o ON o.id = om.organisation_id
			LEFT JOIN roster ro ON ro.organisation_id = om.organisation_id
			LEFT JOIN primary_country pc ON pc.artist_id = ro.artist_id
			-- An empty roster is nothing to chart, so it reads as absent rather than
			-- as stats full of zeroes. Excluded origins read as absent too, including
			-- any organisation synced before its origin was set to one of them.
			WHERE om.organisation_id = ANY($1) AND om.roster_size > 0 AND o.origin <> ALL($5)
			GROUP BY om.organisation_id, om.roster_size, om.last_synced_at
		)
		SELECT
			t.organisation_id, t.matched, t.roster_size, t.last_synced_at,
			t.blocked, t.warning, t.approved, t.status_unknown,
			COALESCE(cl.countries, '[]'::json),
			t.country_unknown
		FROM totals t
		LEFT JOIN country_lists cl ON cl.organisation_id = t.organisation_id
	`, orgIDs, listenLabelBlocked, listenLabelWarning, listenLabelsApproved, domain.MusoExcludedOrigins())
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
