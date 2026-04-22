package repository

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PhonkersBase/base-api2/internal/domain"
)

type CountryRepo struct {
	db *pgxpool.Pool
}

func NewCountryRepo(db *pgxpool.Pool) *CountryRepo {
	return &CountryRepo{db: db}
}

func (r *CountryRepo) GetAll(ctx context.Context) ([]domain.Country, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, original_name, created_at, updated_at
		FROM countries
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	countries := []domain.Country{}
	for rows.Next() {
		var (
			id                   int
			name, originalName   string
			c                    domain.Country
		)
		if err := rows.Scan(&id, &name, &originalName, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		c.ID = strconv.Itoa(id)
		c.Name = name
		c.OriginalName = originalName
		countries = append(countries, c)
	}
	return countries, rows.Err()
}
