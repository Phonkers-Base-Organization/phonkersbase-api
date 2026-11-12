package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PhonkersBase/base-api2/internal/domain"
)

type SiteSettingsRepo struct {
	db *pgxpool.Pool
}

func NewSiteSettingsRepo(db *pgxpool.Pool) *SiteSettingsRepo {
	return &SiteSettingsRepo{db: db}
}

func (r *SiteSettingsRepo) Get(ctx context.Context) (*domain.SiteSettings, error) {
	var s domain.SiteSettings
	err := r.db.QueryRow(ctx, `
		-- name: site_settings.get
		SELECT show_organisation_recommendations
		FROM site_settings
	`).Scan(&s.ShowOrganisationRecommendations)
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func (r *SiteSettingsRepo) Update(ctx context.Context, input domain.SiteSettings) (*domain.SiteSettings, error) {
	var s domain.SiteSettings
	err := r.db.QueryRow(ctx, `
		-- name: site_settings.update
		UPDATE site_settings
		SET show_organisation_recommendations = $1
		RETURNING show_organisation_recommendations
	`, input.ShowOrganisationRecommendations).Scan(&s.ShowOrganisationRecommendations)
	if err != nil {
		return nil, err
	}
	return &s, nil
}
