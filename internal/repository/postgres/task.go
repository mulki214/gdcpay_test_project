package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mulki214/gdcpay_test_project/internal/task"
)

type TaskRepository struct {
	db *pgxpool.Pool
}

func NewTaskRepository(db *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{db: db}
}

func (r *TaskRepository) Create(ctx context.Context, item task.Task) (task.Task, error) {
	err := r.db.QueryRow(ctx, `
		INSERT INTO tasks (id, owner_id, title, description, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, owner_id, assignee_id, title, description, status, created_at, updated_at`,
		item.ID, item.OwnerID, item.Title, item.Description, item.Status,
	).Scan(&item.ID, &item.OwnerID, &item.AssigneeID, &item.Title, &item.Description, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	return item, err
}

func (r *TaskRepository) List(ctx context.Context, ownerID uuid.UUID, filter task.ListFilter) ([]task.Task, int64, error) {
	const where = `owner_id = $1 AND ($2 = '' OR status = $2) AND ($3 = '' OR title ILIKE '%' || $3 || '%')`
	var total int64
	if err := r.db.QueryRow(ctx, `SELECT COUNT(*) FROM tasks WHERE `+where, ownerID, filter.Status, filter.Search).Scan(&total); err != nil {
		return nil, 0, err
	}
	rows, err := r.db.Query(ctx, `
		SELECT id, owner_id, assignee_id, title, description, status, created_at, updated_at
		FROM tasks
		WHERE `+where+`
		ORDER BY created_at DESC, id DESC
		LIMIT $4 OFFSET $5`, ownerID, filter.Status, filter.Search, filter.Limit, (filter.Page-1)*filter.Limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	items := make([]task.Task, 0)
	for rows.Next() {
		var item task.Task
		if err := rows.Scan(&item.ID, &item.OwnerID, &item.AssigneeID, &item.Title, &item.Description, &item.Status, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}

func (r *TaskRepository) GetByID(ctx context.Context, ownerID, taskID uuid.UUID) (task.Task, error) {
	var item task.Task
	err := r.db.QueryRow(ctx, `
		SELECT id, owner_id, assignee_id, title, description, status, created_at, updated_at
		FROM tasks WHERE id = $1 AND owner_id = $2`, taskID, ownerID,
	).Scan(&item.ID, &item.OwnerID, &item.AssigneeID, &item.Title, &item.Description, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return task.Task{}, task.ErrNotFound
	}
	return item, err
}

func (r *TaskRepository) Update(ctx context.Context, ownerID, taskID uuid.UUID, input task.UpdateInput) (task.Task, error) {
	var item task.Task
	err := r.db.QueryRow(ctx, `
		UPDATE tasks SET title = $3, description = $4, status = $5, updated_at = NOW()
		WHERE id = $1 AND owner_id = $2
		RETURNING id, owner_id, assignee_id, title, description, status, created_at, updated_at`,
		taskID, ownerID, input.Title, input.Description, input.Status,
	).Scan(&item.ID, &item.OwnerID, &item.AssigneeID, &item.Title, &item.Description, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return task.Task{}, task.ErrNotFound
	}
	return item, err
}

func (r *TaskRepository) Delete(ctx context.Context, ownerID, taskID uuid.UUID) error {
	tag, err := r.db.Exec(ctx, `DELETE FROM tasks WHERE id = $1 AND owner_id = $2`, taskID, ownerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return task.ErrNotFound
	}
	return nil
}
