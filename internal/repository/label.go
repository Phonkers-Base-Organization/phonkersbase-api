package repository

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PhonkersBase/base-api2/internal/domain"
)

type LabelRepo struct {
	db *pgxpool.Pool
}

func NewLabelRepo(db *pgxpool.Pool) *LabelRepo {
	return &LabelRepo{db: db}
}

func (r *LabelRepo) GetAll(ctx context.Context) ([]domain.Label, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, name, original_name, priority, created_at, updated_at
		FROM labels
		ORDER BY priority DESC, name ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	labels := []domain.Label{}
	for rows.Next() {
		var (
			id int
			l  domain.Label
		)
		if err := rows.Scan(&id, &l.Name, &l.OriginalName, &l.Priority, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		l.ID = strconv.Itoa(id)
		labels = append(labels, l)
	}
	return labels, rows.Err()
}
