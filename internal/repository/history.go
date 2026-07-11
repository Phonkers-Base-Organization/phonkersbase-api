package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/PhonkersBase/base-api2/internal/domain"
)

type ChangeHistoryRepo struct {
	db *pgxpool.Pool
}

func NewChangeHistoryRepo(db *pgxpool.Pool) *ChangeHistoryRepo {
	return &ChangeHistoryRepo{db: db}
}

// ChangeHistoryEntry is the input to Insert; old/new are marshaled to JSONB.
// Either may be nil (create has no old, delete has no new).
type ChangeHistoryEntry struct {
	EntityType     string
	EntityID       string
	EntityName     string
	Action         string
	EditorID       string
	EditorUsername string
	Old            any
	New            any
}

func (r *ChangeHistoryRepo) Insert(ctx context.Context, entry ChangeHistoryEntry) error {
	oldData, err := marshalOrNil(entry.Old)
	if err != nil {
		return err
	}
	newData, err := marshalOrNil(entry.New)
	if err != nil {
		return err
	}

	_, err = r.db.Exec(ctx, `
		-- name: change_history.insert
		INSERT INTO change_history (entity_type, entity_id, entity_name, action, editor_id, editor_username, old_data, new_data)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, entry.EntityType, entry.EntityID, entry.EntityName, entry.Action, entry.EditorID, entry.EditorUsername, oldData, newData)
	return err
}

func marshalOrNil(v any) ([]byte, error) {
	if v == nil {
		return nil, nil
	}
	return json.Marshal(v)
}

// ListEditors returns the distinct editor usernames present in the change history.
func (r *ChangeHistoryRepo) ListEditors(ctx context.Context) ([]string, error) {
	rows, err := r.db.Query(ctx, `
		-- name: change_history.list_editors
		SELECT DISTINCT editor_username FROM change_history ORDER BY editor_username
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	editors := []string{}
	for rows.Next() {
		var e string
		if err := rows.Scan(&e); err != nil {
			return nil, err
		}
		editors = append(editors, e)
	}
	return editors, rows.Err()
}

func (r *ChangeHistoryRepo) List(ctx context.Context, params domain.ListChangesParams) (*domain.PaginatedChanges, error) {
	limit := params.Limit
	if limit <= 0 {
		limit = 50
	}
	if limit > 500 {
		limit = 500
	}

	conditions := []string{}
	args := []any{}
	n := 1

	if params.EntityType != "" {
		conditions = append(conditions, fmt.Sprintf("entity_type = $%d", n))
		args = append(args, params.EntityType)
		n++
	}
	if params.EntityID != "" {
		conditions = append(conditions, fmt.Sprintf("entity_id = $%d", n))
		args = append(args, params.EntityID)
		n++
	}
	if params.Action != "" {
		conditions = append(conditions, fmt.Sprintf("action = $%d", n))
		args = append(args, params.Action)
		n++
	}
	if params.Editor != "" {
		conditions = append(conditions, fmt.Sprintf("editor_username = $%d", n))
		args = append(args, params.Editor)
		n++
	}
	if params.Search != "" {
		conditions = append(conditions, fmt.Sprintf("entity_name ILIKE '%%' || $%d || '%%'", n))
		args = append(args, escapeLikePattern(params.Search))
		n++
	}

	where := ""
	if len(conditions) > 0 {
		where = "WHERE "
		for i, c := range conditions {
			if i > 0 {
				where += " AND "
			}
			where += c
		}
	}

	var total int
	countQuery := "-- name: change_history.list.count\nSELECT COUNT(*) FROM change_history " + where
	if err := r.db.QueryRow(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, err
	}

	itemsArgs := append(append([]any{}, args...), limit, params.Offset)
	itemsQuery := fmt.Sprintf(`
		-- name: change_history.list.items
		SELECT id, entity_type, entity_id, entity_name, action, editor_id, editor_username, old_data, new_data, created_at
		FROM change_history
		%s
		ORDER BY created_at DESC, id DESC
		LIMIT $%d OFFSET $%d
	`, where, n, n+1)

	rows, err := r.db.Query(ctx, itemsQuery, itemsArgs...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []domain.ChangeEntry{}
	for rows.Next() {
		var (
			id int64
			e  domain.ChangeEntry
		)
		if err := rows.Scan(&id, &e.EntityType, &e.EntityID, &e.EntityName, &e.Action, &e.EditorID, &e.EditorUsername, &e.OldData, &e.NewData, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.ID = fmt.Sprintf("%d", id)
		items = append(items, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	totalPages := total / limit
	if total%limit != 0 {
		totalPages++
	}
	currentPage := params.Offset/limit + 1

	return &domain.PaginatedChanges{
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
