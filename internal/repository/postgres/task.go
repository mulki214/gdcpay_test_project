package postgres

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mulki214/gdcpay_test_project/internal/task"
)

type TaskRepository struct {
	db            *pgxpool.Pool
	beginAssignTx func(context.Context) (assignmentTransaction, error)
}

type assignmentTransaction interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Commit(context.Context) error
	Rollback(context.Context) error
}

func NewTaskRepository(db *pgxpool.Pool) *TaskRepository {
	return &TaskRepository{
		db: db,
		beginAssignTx: func(ctx context.Context) (assignmentTransaction, error) {
			return db.Begin(ctx)
		},
	}
}

func (r *TaskRepository) Create(ctx context.Context, item task.Task) (task.Task, error) {
	err := insertTask(ctx, r.db, &item)
	return item, err
}

func (r *TaskRepository) CreateIdempotent(ctx context.Context, item task.Task, key uuid.UUID, requestHash [sha256.Size]byte, expiresAt time.Time) (task.Task, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return task.Task{}, err
	}
	defer tx.Rollback(ctx)

	var claimedKey uuid.UUID
	err = tx.QueryRow(ctx, `
		INSERT INTO task_idempotency_keys (owner_id, idempotency_key, request_hash, expires_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (owner_id, idempotency_key) DO UPDATE
		SET request_hash = EXCLUDED.request_hash,
		    response_body = NULL,
		    expires_at = EXCLUDED.expires_at
		WHERE task_idempotency_keys.expires_at <= $4 - INTERVAL '24 hours'
		RETURNING idempotency_key`, item.OwnerID, key, requestHash[:], expiresAt,
	).Scan(&claimedKey)
	if errors.Is(err, pgx.ErrNoRows) {
		var storedHash, responseBody []byte
		if err := tx.QueryRow(ctx, `
			SELECT request_hash, response_body
			FROM task_idempotency_keys
			WHERE owner_id = $1 AND idempotency_key = $2 AND expires_at > $3 - INTERVAL '24 hours'`, item.OwnerID, key, expiresAt,
		).Scan(&storedHash, &responseBody); err != nil {
			return task.Task{}, err
		}
		if !bytes.Equal(storedHash, requestHash[:]) {
			return task.Task{}, task.ErrIdempotencyKeyReused
		}
		if len(responseBody) == 0 {
			return task.Task{}, errors.New("idempotency record has no response")
		}
		if err := json.Unmarshal(responseBody, &item); err != nil {
			return task.Task{}, err
		}
		if err := tx.Commit(ctx); err != nil {
			return task.Task{}, err
		}
		return item, nil
	}
	if err != nil {
		return task.Task{}, err
	}

	if err := insertTask(ctx, tx, &item); err != nil {
		return task.Task{}, err
	}
	responseBody, err := json.Marshal(item)
	if err != nil {
		return task.Task{}, err
	}
	if _, err := tx.Exec(ctx, `
		UPDATE task_idempotency_keys
		SET response_body = $3
		WHERE owner_id = $1 AND idempotency_key = $2`, item.OwnerID, key, responseBody,
	); err != nil {
		return task.Task{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return task.Task{}, err
	}
	return item, nil
}

type taskQueryRower interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}

func insertTask(ctx context.Context, db taskQueryRower, item *task.Task) error {
	return db.QueryRow(ctx, `
		INSERT INTO tasks (id, owner_id, title, description, status)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, owner_id, assignee_id, title, description, status, created_at, updated_at`,
		item.ID, item.OwnerID, item.Title, item.Description, item.Status,
	).Scan(&item.ID, &item.OwnerID, &item.AssigneeID, &item.Title, &item.Description, &item.Status, &item.CreatedAt, &item.UpdatedAt)
}

func (r *TaskRepository) Assign(ctx context.Context, ownerID, taskID, assigneeID uuid.UUID) (task.Task, error) {
	tx, err := r.beginAssignTx(ctx)
	if err != nil {
		return task.Task{}, err
	}
	defer tx.Rollback(ctx)

	var previousAssigneeID *uuid.UUID
	err = tx.QueryRow(ctx, `
		SELECT assignee_id FROM tasks
		WHERE id = $1 AND owner_id = $2
		FOR UPDATE`, taskID, ownerID,
	).Scan(&previousAssigneeID)
	if errors.Is(err, pgx.ErrNoRows) {
		return task.Task{}, task.ErrNotFound
	}
	if err != nil {
		return task.Task{}, err
	}

	var sameTeam bool
	err = tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1
			FROM team_members owner_members
			JOIN team_members assignee_members USING (team_id)
			WHERE owner_members.user_id = $1
			  AND assignee_members.user_id = $2
		)`, ownerID, assigneeID,
	).Scan(&sameTeam)
	if err != nil {
		return task.Task{}, err
	}
	if !sameTeam {
		return task.Task{}, task.ErrAssigneeNotInTeam
	}

	var item task.Task
	err = tx.QueryRow(ctx, `
		UPDATE tasks
		SET assignee_id = $3, updated_at = NOW()
		WHERE id = $1 AND owner_id = $2
		RETURNING id, owner_id, assignee_id, title, description, status, created_at, updated_at`,
		taskID, ownerID, assigneeID,
	).Scan(&item.ID, &item.OwnerID, &item.AssigneeID, &item.Title, &item.Description, &item.Status, &item.CreatedAt, &item.UpdatedAt)
	if err != nil {
		return task.Task{}, err
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO task_logs (id, task_id, actor_id, event_type, previous_assignee_id, new_assignee_id)
		VALUES ($1, $2, $3, 'assigned', $4, $5)`, uuid.New(), taskID, ownerID, previousAssigneeID, assigneeID,
	); err != nil {
		return task.Task{}, err
	}
	payload, err := json.Marshal(struct {
		TaskID     uuid.UUID `json:"task_id"`
		TaskTitle  string    `json:"task_title"`
		AssignedBy uuid.UUID `json:"assigned_by"`
	}{TaskID: item.ID, TaskTitle: item.Title, AssignedBy: ownerID})
	if err != nil {
		return task.Task{}, fmt.Errorf("encode assignment notification: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO task_notifications (id, task_id, recipient_user_id, event_type, payload)
		VALUES ($1, $2, $3, 'task.assigned', $4)`, uuid.New(), taskID, assigneeID, payload,
	); err != nil {
		return task.Task{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return task.Task{}, err
	}
	return item, nil
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
