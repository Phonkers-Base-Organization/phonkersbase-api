package repository

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PhonkersBase/base-api2/internal/domain"
)

type OrganisationRepo struct {
	db *pgxpool.Pool
}

func NewOrganisationRepo(db *pgxpool.Pool) *OrganisationRepo {
	return &OrganisationRepo{db: db}
}

func (r *OrganisationRepo) GetAll(ctx context.Context, params domain.ListOrganisationsParams) ([]domain.Organisation, error) {
	query := `
		SELECT id, name, link, origin, info, type, recommendation, created_at, updated_at
		FROM organisations
		WHERE ($1 = '' OR type = $1)
		  AND ($2 = '' OR name ILIKE '%' || $2 || '%')
		ORDER BY name ASC
	`
	rows, err := r.db.Query(ctx, query, params.Type, params.Search)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	orgs := []domain.Organisation{}
	for rows.Next() {
		var (
			id int
			o  domain.Organisation
		)
		if err := rows.Scan(&id, &o.Name, &o.Link, &o.Origin, &o.Info, &o.Type, &o.Recommendation, &o.CreatedAt, &o.UpdatedAt); err != nil {
			return nil, err
		}
		o.ID = strconv.Itoa(id)
		orgs = append(orgs, o)
	}
	return orgs, rows.Err()
}

func (r *OrganisationRepo) Create(ctx context.Context, input domain.UpsertOrganisationInput) (*domain.Organisation, error) {
	var o domain.Organisation
	var id int
	err := r.db.QueryRow(ctx, `
		INSERT INTO organisations (name, link, origin, info, type, recommendation)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, link, origin, info, type, recommendation, created_at, updated_at
	`, input.Name, input.Link, input.Origin, input.Info, input.Type, input.Recommendation).Scan(
		&id, &o.Name, &o.Link, &o.Origin, &o.Info, &o.Type, &o.Recommendation, &o.CreatedAt, &o.UpdatedAt,
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
		UPDATE organisations SET name = $1, link = $2, origin = $3, info = $4, type = $5, recommendation = $6
		WHERE id = $7
		RETURNING id, name, link, origin, info, type, recommendation, created_at, updated_at
	`, input.Name, input.Link, input.Origin, input.Info, input.Type, input.Recommendation, id).Scan(
		&oid, &o.Name, &o.Link, &o.Origin, &o.Info, &o.Type, &o.Recommendation, &o.CreatedAt, &o.UpdatedAt,
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
	tag, err := r.db.Exec(ctx, `DELETE FROM organisations WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
