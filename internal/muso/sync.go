package muso

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/PhonkersBase/base-api2/internal/domain"
)

// Search and the collaborators endpoint disagree by a unit or two on the same roster,
// so only a larger move means the organisation actually signed or dropped someone.
const collaboratorCountDrift = 2

// Long enough to write one row, short enough that a wedged database cannot hold
// the sweep open.
const recordFailureTimeout = 10 * time.Second

const (
	statusPending  = "pending"
	statusOK       = "ok"
	statusError    = "error"
	statusNotFound = "not_found"
)

type organisationRosterStore interface {
	// GetOrganisationMuso returns nil when the organisation has never been checked.
	GetOrganisationMuso(ctx context.Context, organisationID int) (*domain.OrganisationMuso, error)
	UpsertOrganisationMuso(ctx context.Context, state *domain.OrganisationMuso) error
	ResolveArtistIDsBySpotifyIDs(ctx context.Context, spotifyIDs []string) (map[string]int, error)
	ReplaceOrganisationArtists(ctx context.Context, organisationID int, artistProfiles map[int]string) error
}

type Syncer struct {
	client *Client
	store  organisationRosterStore
}

func NewSyncer(client *Client, store organisationRosterStore) *Syncer {
	return &Syncer{client: client, store: store}
}

// SyncOrganisation walks the organisation's full muso roster and replaces its stored
// artist matches. Hundreds of requests for a large label, so it runs only when
// CheckOrganisation says the roster moved, or when an admin forces it.
func (s *Syncer) SyncOrganisation(ctx context.Context, orgID int, name string) error {
	state := &domain.OrganisationMuso{OrganisationID: orgID}

	organisation, err := s.client.SearchOrganisation(ctx, name)
	if err != nil {
		s.recordFailure(ctx, state, err)
		// An organisation muso has never heard of is a gap in muso, not a failure here.
		if errors.Is(err, ErrNotFound) {
			return nil
		}
		log.Error().Err(err).Int("organisation_id", orgID).Msg("muso sync failed")
		return fmt.Errorf("search organisation: %w", err)
	}

	state.MusoProfileID = organisation.ID
	state.CollaboratorsCount = organisation.CollaboratorsCount
	state.CreditCount = organisation.CreditCount

	artistProfiles, rosterSize, err := s.collectRoster(ctx, organisation.ID)
	if err != nil {
		s.recordFailure(ctx, state, err)
		log.Error().Err(err).Int("organisation_id", orgID).Msg("muso sync failed")
		return err
	}
	state.RosterSize = rosterSize

	err = s.store.ReplaceOrganisationArtists(ctx, orgID, artistProfiles)
	if err != nil {
		log.Error().Err(err).Int("organisation_id", orgID).Msg("muso sync failed")
		return fmt.Errorf("replace organisation artists: %w", err)
	}

	now := time.Now().UTC()
	state.LastCheckedAt = &now
	state.LastSyncedAt = &now
	state.SyncStatus = statusOK
	err = s.store.UpsertOrganisationMuso(ctx, state)
	if err != nil {
		log.Error().Err(err).Int("organisation_id", orgID).Msg("muso sync failed")
		return fmt.Errorf("record successful muso sync: %w", err)
	}

	log.Info().
		Int("organisation_id", orgID).
		Int("roster_size", rosterSize).
		Int("matched", len(artistProfiles)).
		Msg("muso roster synced")
	return nil
}

// CheckOrganisation costs one muso request and reports whether the expensive roster
// walk is worth running.
func (s *Syncer) CheckOrganisation(ctx context.Context, orgID int, name string) (bool, error) {
	stored, err := s.store.GetOrganisationMuso(ctx, orgID)
	if err != nil {
		log.Error().Err(err).Int("organisation_id", orgID).Msg("muso check failed")
		return false, fmt.Errorf("load muso organisation: %w", err)
	}

	state := stored
	if state == nil {
		state = &domain.OrganisationMuso{OrganisationID: orgID, SyncStatus: statusPending}
	}

	organisation, err := s.client.SearchOrganisation(ctx, name)
	if err != nil {
		s.recordFailure(ctx, state, err)
		if errors.Is(err, ErrNotFound) {
			return false, nil
		}
		log.Error().Err(err).Int("organisation_id", orgID).Msg("muso check failed")
		return false, fmt.Errorf("check muso organisation: %w", err)
	}

	drift := organisation.CollaboratorsCount - state.CollaboratorsCount
	if drift < 0 {
		drift = -drift
	}
	warranted := stored == nil || state.SyncStatus != statusOK || drift > collaboratorCountDrift

	now := time.Now().UTC()
	state.LastCheckedAt = &now
	state.MusoProfileID = organisation.ID
	// Only a completed sync may advance the stored count. Advancing it here would
	// erase the very signal the next run compares against.
	if stored == nil {
		state.CollaboratorsCount = organisation.CollaboratorsCount
		state.CreditCount = organisation.CreditCount
	}

	err = s.store.UpsertOrganisationMuso(ctx, state)
	if err != nil {
		log.Error().Err(err).Int("organisation_id", orgID).Msg("muso check failed")
		return false, fmt.Errorf("record muso check: %w", err)
	}
	return warranted, nil
}

// collectRoster pages through every collaborator and returns the pb artists among them,
// keyed by artist id, alongside the roster size muso reports.
func (s *Syncer) collectRoster(ctx context.Context, profileID string) (map[int]string, int, error) {
	artistProfiles := make(map[int]string)
	rosterSize := 0

	for offset := 0; ; {
		page, err := s.client.Collaborators(ctx, profileID, offset)
		if err != nil {
			return nil, 0, fmt.Errorf("get collaborators: %w", err)
		}
		if offset == 0 {
			rosterSize = page.TotalCount
		}
		// muso's totalCount overstates some rosters by a collaborator or two, and a
		// label can have none at all, so an empty page ends the walk rather than
		// failing it. It also stops the loop advancing nothing against a total it
		// would never reach.
		if len(page.Items) == 0 {
			return artistProfiles, rosterSize, nil
		}

		profilesBySpotifyID := make(map[string]string, len(page.Items))
		spotifyIDs := make([]string, 0, len(page.Items))
		for _, collaborator := range page.Items {
			_, seen := profilesBySpotifyID[collaborator.SpotifyID]
			if collaborator.SpotifyID == "" || seen {
				continue
			}
			profilesBySpotifyID[collaborator.SpotifyID] = collaborator.ID
			spotifyIDs = append(spotifyIDs, collaborator.SpotifyID)
		}

		if len(spotifyIDs) > 0 {
			artistIDs, err := s.store.ResolveArtistIDsBySpotifyIDs(ctx, spotifyIDs)
			if err != nil {
				return nil, 0, fmt.Errorf("resolve collaborator spotify ids: %w", err)
			}
			for spotifyID, artistID := range artistIDs {
				artistProfiles[artistID] = profilesBySpotifyID[spotifyID]
			}
		}

		offset += len(page.Items)
		if offset >= rosterSize {
			return artistProfiles, rosterSize, nil
		}
	}
}

// recordFailure stores why the sync stopped. A store error here is logged and dropped:
// the muso error is the one the caller has to act on.
func (s *Syncer) recordFailure(ctx context.Context, state *domain.OrganisationMuso, cause error) {
	// The usual cause is the sync context expiring, which would take this write
	// down with it and leave the row claiming the last attempt still stands.
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), recordFailureTimeout)
	defer cancel()

	now := time.Now().UTC()
	state.LastCheckedAt = &now

	if errors.Is(cause, ErrNotFound) {
		state.SyncStatus = statusNotFound
		state.SyncError = nil
	} else {
		message := cause.Error()
		state.SyncStatus = statusError
		state.SyncError = &message
	}

	err := s.store.UpsertOrganisationMuso(ctx, state)
	if err != nil {
		log.Error().Err(err).Int("organisation_id", state.OrganisationID).Msg("failed to record muso sync status")
	}
}
