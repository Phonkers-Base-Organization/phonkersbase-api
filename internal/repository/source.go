package repository

import (
	"context"
	"strconv"
	"time"

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
	rows, err := r.db.Query(ctx, `SELECT id, name, created_at FROM evidence_sources ORDER BY name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	sources := []domain.EvidenceSource{}
	for rows.Next() {
		var (
			id        int
			name      string
			createdAt time.Time
		)
		if err := rows.Scan(&id, &name, &createdAt); err != nil {
			return nil, err
		}
		sources = append(sources, domain.EvidenceSource{
			ID:        strconv.Itoa(id),
			Name:      name,
			CreatedAt: createdAt,
		})
	}
	return sources, rows.Err()
}

func (r *EvidenceSourceRepo) Create(ctx context.Context, name string) (*domain.EvidenceSource, error) {
	var (
		id        int
		createdAt time.Time
	)
	err := r.db.QueryRow(ctx,
		`INSERT INTO evidence_sources (name) VALUES ($1) RETURNING id, name, created_at`,
		name,
	).Scan(&id, &name, &createdAt)
	if err != nil {
		return nil, err
	}
	return &domain.EvidenceSource{
		ID:        strconv.Itoa(id),
		Name:      name,
		CreatedAt: createdAt,
	}, nil
}
