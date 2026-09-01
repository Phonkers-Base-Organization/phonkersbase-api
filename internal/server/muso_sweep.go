package server

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/PhonkersBase/base-api2/internal/muso"
	"github.com/PhonkersBase/base-api2/internal/repository"
)

const (
	musoSweepHour     = 3
	musoSweepTimezone = "Europe/Kyiv"
)

func runMusoSweep(ctx context.Context, syncer *muso.Syncer, repo *repository.OrganisationMusoRepo) {
	location, err := time.LoadLocation(musoSweepTimezone)
	if err != nil {
		log.Error().Err(err).Str("timezone", musoSweepTimezone).Msg("failed to load muso sweep timezone; using UTC")
		location = time.UTC
	}

	for {
		now := time.Now().In(location)
		next := time.Date(now.Year(), now.Month(), now.Day(), musoSweepHour, 0, 0, 0, location)
		if !next.After(now) {
			next = next.AddDate(0, 0, 1)
		}

		// Target the next wall-clock occurrence so restarts and daylight-saving changes do not shift the schedule.
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			sweepOrganisations(ctx, syncer, repo)
		}
	}
}

func sweepOrganisations(ctx context.Context, syncer *muso.Syncer, repo *repository.OrganisationMusoRepo) {
	startedAt := time.Now()
	checked := 0
	synced := 0

	log.Info().Msg("muso organisation sweep started")
	defer func() {
		log.Info().
			Int("checked", checked).
			Int("synced", synced).
			Dur("duration", time.Since(startedAt)).
			Msg("muso organisation sweep finished")
	}()

	targets, err := repo.GetOrganisationsForSweep(ctx)
	if err != nil {
		if ctx.Err() == nil {
			log.Error().Err(err).Msg("failed to load organisations for muso sweep")
		}
		return
	}

	flagged := make([]repository.OrganisationMusoSweepTarget, 0)
	// Finish the cheap checks for every organisation before any expensive roster walk begins.
	for _, target := range targets {
		select {
		case <-ctx.Done():
			return
		default:
		}

		checked++
		needsSync, err := syncer.CheckOrganisation(ctx, target.ID, target.Name)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Error().Err(err).
				Int("organisation_id", target.ID).
				Str("organisation_name", target.Name).
				Str("phase", "check").
				Msg("muso organisation sweep failed")
			continue
		}
		if needsSync {
			flagged = append(flagged, target)
		}
	}

	for _, target := range flagged {
		select {
		case <-ctx.Done():
			return
		default:
		}

		if err := syncer.SyncOrganisation(ctx, target.ID, target.Name); err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Error().Err(err).
				Int("organisation_id", target.ID).
				Str("organisation_name", target.Name).
				Str("phase", "sync").
				Msg("muso organisation sweep failed")
			continue
		}
		synced++
	}
}
