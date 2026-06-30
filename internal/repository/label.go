package repository

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
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
		-- name: label.get_all
		SELECT id, name, priority, created_at, updated_at
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
		if err := rows.Scan(&id, &l.Name, &l.Priority, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		l.ID = strconv.Itoa(id)
		labels = append(labels, l)
	}
	return labels, rows.Err()
}

func (r *LabelRepo) Create(ctx context.Context, input domain.UpsertLabelInput) (*domain.Label, error) {
	var l domain.Label
	var id int
	err := r.db.QueryRow(ctx, `
		-- name: label.create
		INSERT INTO labels (name, priority)
		VALUES ($1, $2)
		RETURNING id, name, priority, created_at, updated_at
	`, input.Name, input.Priority).Scan(
		&id, &l.Name, &l.Priority, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	l.ID = strconv.Itoa(id)
	return &l, nil
}

func (r *LabelRepo) Update(ctx context.Context, id string, input domain.UpsertLabelInput) (*domain.Label, error) {
	var l domain.Label
	var lid int
	err := r.db.QueryRow(ctx, `
		-- name: label.update
		UPDATE labels SET name = $1, priority = $2
		WHERE id = $3
		RETURNING id, name, priority, created_at, updated_at
	`, input.Name, input.Priority, id).Scan(
		&lid, &l.Name, &l.Priority, &l.CreatedAt, &l.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	l.ID = strconv.Itoa(lid)
	return &l, nil
}

func (r *LabelRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `
		-- name: label.delete
		DELETE FROM labels WHERE id = $1
	`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
