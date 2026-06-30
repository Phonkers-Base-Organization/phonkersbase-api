package repository

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PhonkersBase/base-api2/internal/domain"
)

type FeedbackRepo struct {
	db *pgxpool.Pool
}

func NewFeedbackRepo(db *pgxpool.Pool) *FeedbackRepo {
	return &FeedbackRepo{db: db}
}

func (r *FeedbackRepo) GetAll(ctx context.Context) ([]domain.Feedback, error) {
	rows, err := r.db.Query(ctx, `
		-- name: feedback.get_all
		SELECT id, type, text, email, created_at
		FROM feedbacks
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	feedbacks := []domain.Feedback{}
	for rows.Next() {
		var (
			id int
			f  domain.Feedback
		)
		if err := rows.Scan(&id, &f.Type, &f.Text, &f.Email, &f.CreatedAt); err != nil {
			return nil, err
		}
		f.ID = strconv.Itoa(id)
		feedbacks = append(feedbacks, f)
	}
	return feedbacks, rows.Err()
}

func (r *FeedbackRepo) Create(ctx context.Context, typ, text string, email *string) (*domain.Feedback, error) {
	var (
		id int
		f  domain.Feedback
	)
	err := r.db.QueryRow(ctx, `
		-- name: feedback.create
		INSERT INTO feedbacks (type, text, email)
		VALUES ($1, $2, $3)
		RETURNING id, type, text, email, created_at
	`, typ, text, email).Scan(&id, &f.Type, &f.Text, &f.Email, &f.CreatedAt)
	if err != nil {
		return nil, err
	}
	f.ID = strconv.Itoa(id)
	return &f, nil
}

func (r *FeedbackRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `
		-- name: feedback.delete
		DELETE FROM feedbacks WHERE id = $1
	`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
