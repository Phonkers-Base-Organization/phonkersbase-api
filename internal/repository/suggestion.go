package repository

import (
	"context"
	"strconv"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PhonkersBase/base-api2/internal/domain"
)

type SuggestionRepo struct {
	db *pgxpool.Pool
}

func NewSuggestionRepo(db *pgxpool.Pool) *SuggestionRepo {
	return &SuggestionRepo{db: db}
}

func (r *SuggestionRepo) GetAll(ctx context.Context) ([]domain.Suggestion, error) {
	rows, err := r.db.Query(ctx, `
		-- name: suggestion.get_all
		SELECT id, name, link, countries, listen_labels, evidence, description, created_at
		FROM suggestions
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	suggestions := []domain.Suggestion{}
	for rows.Next() {
		var (
			id int
			s  domain.Suggestion
		)
		if err := rows.Scan(
			&id, &s.Name, &s.Link, &s.Countries, &s.ListenLabels,
			&s.Evidence, &s.Description,
			&s.CreatedAt,
		); err != nil {
			return nil, err
		}
		s.ID = strconv.Itoa(id)
		if s.Countries == nil {
			s.Countries = []string{}
		}
		if s.ListenLabels == nil {
			s.ListenLabels = []string{}
		}
		suggestions = append(suggestions, s)
	}
	return suggestions, rows.Err()
}

func (r *SuggestionRepo) Create(ctx context.Context, name string, link *string, countries []string, listenLabels []string, evidence *string, description *string) (*domain.Suggestion, error) {
	var (
		id int
		s  domain.Suggestion
	)
	err := r.db.QueryRow(ctx, `
		-- name: suggestion.create
		INSERT INTO suggestions (name, link, countries, listen_labels, evidence, description)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, name, link, countries, listen_labels, evidence, description, created_at
	`, name, link, countries, listenLabels, evidence, description).Scan(
		&id, &s.Name, &s.Link, &s.Countries, &s.ListenLabels,
		&s.Evidence, &s.Description,
		&s.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	s.ID = strconv.Itoa(id)
	if s.Countries == nil {
		s.Countries = []string{}
	}
	if s.ListenLabels == nil {
		s.ListenLabels = []string{}
	}
	return &s, nil
}

func (r *SuggestionRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `
		-- name: suggestion.delete
		DELETE FROM suggestions WHERE id = $1
	`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
