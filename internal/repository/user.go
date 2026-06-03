package repository

import (
	"context"
	"errors"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PhonkersBase/base-api2/internal/domain"
)

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) GetAll(ctx context.Context) ([]domain.User, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, username, role, created_at, updated_at
		FROM users
		ORDER BY created_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := []domain.User{}
	for rows.Next() {
		var (
			id int
			u  domain.User
		)
		if err := rows.Scan(&id, &u.Username, &u.Role, &u.CreatedAt, &u.UpdatedAt); err != nil {
			return nil, err
		}
		u.ID = strconv.Itoa(id)
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepo) GetByUsername(ctx context.Context, username string) (*domain.User, string, error) {
	var (
		id           int
		u            domain.User
		passwordHash string
	)
	err := r.db.QueryRow(ctx, `
		SELECT id, username, role, password_hash, created_at, updated_at
		FROM users
		WHERE username = $1
	`, username).Scan(&id, &u.Username, &u.Role, &passwordHash, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, "", ErrNotFound
		}
		return nil, "", err
	}
	u.ID = strconv.Itoa(id)
	return &u, passwordHash, nil
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*domain.User, error) {
	var (
		uid int
		u   domain.User
	)
	err := r.db.QueryRow(ctx, `
		SELECT id, username, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`, id).Scan(&uid, &u.Username, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	u.ID = strconv.Itoa(uid)
	return &u, nil
}

func (r *UserRepo) Create(ctx context.Context, username, passwordHash, role string) (*domain.User, error) {
	var (
		id int
		u  domain.User
	)
	err := r.db.QueryRow(ctx, `
		INSERT INTO users (username, password_hash, role)
		VALUES ($1, $2, $3)
		RETURNING id, username, role, created_at, updated_at
	`, username, passwordHash, role).Scan(&id, &u.Username, &u.Role, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return nil, err
	}
	u.ID = strconv.Itoa(id)
	return &u, nil
}

func (r *UserRepo) Delete(ctx context.Context, id string) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM users WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *UserRepo) UpdateRole(ctx context.Context, id, role string) error {
	tag, err := r.db.Exec(ctx, `UPDATE users SET role = $1 WHERE id = $2`, role, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
