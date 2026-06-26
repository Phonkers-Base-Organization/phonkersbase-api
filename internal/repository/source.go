package repository

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PhonkersBase/base-api2/internal/domain"
)

type EvidenceSourceRepo struct {
	db *pgxpool.Pool
}

func NewEvidenceSourceRepo(db *pgxpool.Pool) *EvidenceSourceRepo {
	return &EvidenceSourceRepo{db: db}
}

func (r *EvidenceSourceRepo) GetAll(ctx context.Context) ([]domain.EvidenceSource, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, name_en, created_at
		FROM evidence_sources
		ORDER BY name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sources := []domain.EvidenceSource{}
	for rows.Next() {
		var (
			id        int
			name      string
			nameEn    string
			createdAt time.Time
		)
		if err := rows.Scan(&id, &name, &nameEn, &createdAt); err != nil {
			return nil, err
		}
		sources = append(sources, domain.EvidenceSource{
			ID:        strconv.Itoa(id),
			Name:      name,
			NameEn:    nameEn,
			CreatedAt: createdAt,
		})
	}
	return sources, rows.Err()
}

func (r *EvidenceSourceRepo) Create(ctx context.Context, input domain.UpsertSourceInput) (*domain.EvidenceSource, error) {
	var (
		id        int
		name      string
		nameEn    string
		createdAt time.Time
	)
	err := r.db.QueryRow(ctx,
		`INSERT INTO evidence_sources (name, name_en) VALUES ($1, $2)
		 RETURNING id, name, name_en, created_at`,
		input.Name, input.NameEn,
	).Scan(&id, &name, &nameEn, &createdAt)
	if err != nil {
		return nil, err
	}
	return &domain.EvidenceSource{
		ID:        strconv.Itoa(id),
		Name:      name,
		NameEn:    nameEn,
		CreatedAt: createdAt,
	}, nil
}

func (r *EvidenceSourceRepo) Update(ctx context.Context, id string, input domain.UpsertSourceInput) (*domain.EvidenceSource, error) {
	var (
		sid       int
		name      string
		nameEn    string
		createdAt time.Time
	)
	err := r.db.QueryRow(ctx,
		`UPDATE evidence_sources SET name = $1, name_en = $2
		 WHERE id = $3
		 RETURNING id, name, name_en, created_at`,
		input.Name, input.NameEn, id,
	).Scan(&sid, &name, &nameEn, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &domain.EvidenceSource{
		ID:        strconv.Itoa(sid),
		Name:      name,
		NameEn:    nameEn,
		CreatedAt: createdAt,
	}, nil
}

func (r *EvidenceSourceRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM evidence_sources WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
