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
		SELECT COUNT(*) FROM organisations
		WHERE ($1 = '' OR type = $1)
		  AND ($2 = '' OR name ILIKE '%' || $2 || '%')
	`, params.Type, params.Search).Scan(&total)
	if err != nil {
		return nil, err
	}

	rows, err := r.db.Query(ctx, `
		SELECT id, name, link, origin, description_uk, description_en, notes, type, recommendation, created_at, updated_at
		FROM organisations
		WHERE ($1 = '' OR type = $1)
		  AND ($2 = '' OR name ILIKE '%' || $2 || '%')
		ORDER BY name ASC
		LIMIT $3 OFFSET $4
	`, params.Type, params.Search, limit, params.Offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.Organisation{}
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
	}
	if err := rows.Err(); err != nil {
		return nil, err
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

func (r *OrganisationRepo) Create(ctx context.Context, input domain.UpsertOrganisationInput) (*domain.Organisation, error) {
	var o domain.Organisation
	var id int
	err := r.db.QueryRow(ctx, `
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
	tag, err := r.db.Exec(ctx, `DELETE FROM organisations WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
