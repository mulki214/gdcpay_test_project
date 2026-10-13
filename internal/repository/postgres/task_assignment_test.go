package postgres

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/mulki214/gdcpay_test_project/internal/task"
)

func TestAssignRollsBackWhenNotificationWriteFails(t *testing.T) {
	notificationErr := errors.New("notification write failed")
	tx := &assignmentTransactionStub{sameTeam: true, notificationErr: notificationErr}
	repository := repositoryWithAssignmentTx(tx)
	ownerID, taskID, assigneeID := uuid.New(), uuid.New(), uuid.New()

	_, err := repository.Assign(context.Background(), ownerID, taskID, assigneeID)
	if !errors.Is(err, notificationErr) {
		t.Fatalf("assign error = %v, want %v", err, notificationErr)
	}
	if tx.committed || !tx.rolledBack {
		t.Fatalf("transaction committed=%t rolled_back=%t, want rollback", tx.committed, tx.rolledBack)
	}
	want := []string{"lock_task", "check_membership", "update_task", "write_task_log", "write_notification"}
	if !reflect.DeepEqual(tx.events, want) {
		t.Fatalf("transaction steps = %v, want %v", tx.events, want)
	}
}

func TestAssignRejectsAssigneeOutsideSharedTeam(t *testing.T) {
	tx := &assignmentTransactionStub{sameTeam: false}
	repository := repositoryWithAssignmentTx(tx)

	_, err := repository.Assign(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if !errors.Is(err, task.ErrAssigneeNotInTeam) {
		t.Fatalf("assign error = %v, want %v", err, task.ErrAssigneeNotInTeam)
	}
	if tx.committed || !tx.rolledBack {
		t.Fatalf("transaction committed=%t rolled_back=%t, want rollback", tx.committed, tx.rolledBack)
	}
	want := []string{"lock_task", "check_membership"}
	if !reflect.DeepEqual(tx.events, want) {
		t.Fatalf("transaction steps = %v, want %v", tx.events, want)
	}
}

func TestAssignCommitsTaskLogAndNotificationTogether(t *testing.T) {
	tx := &assignmentTransactionStub{sameTeam: true}
	repository := repositoryWithAssignmentTx(tx)
	ownerID, taskID, assigneeID := uuid.New(), uuid.New(), uuid.New()

	item, err := repository.Assign(context.Background(), ownerID, taskID, assigneeID)
	if err != nil {
		t.Fatalf("assign task: %v", err)
	}
	if !tx.committed || tx.rolledBack {
		t.Fatalf("transaction committed=%t rolled_back=%t, want commit", tx.committed, tx.rolledBack)
	}
	if item.ID != taskID || item.OwnerID != ownerID || item.AssigneeID == nil || *item.AssigneeID != assigneeID {
		t.Fatalf("unexpected assigned task: %+v", item)
	}
	want := []string{"lock_task", "check_membership", "update_task", "write_task_log", "write_notification"}
	if !reflect.DeepEqual(tx.events, want) {
		t.Fatalf("transaction steps = %v, want %v", tx.events, want)
	}
}

func repositoryWithAssignmentTx(tx assignmentTransaction) *TaskRepository {
	return &TaskRepository{beginAssignTx: func(context.Context) (assignmentTransaction, error) {
		return tx, nil
	}}
}

type assignmentTransactionStub struct {
	sameTeam        bool
	notificationErr error
	events          []string
	committed       bool
	rolledBack      bool
}

func (tx *assignmentTransactionStub) QueryRow(_ context.Context, query string, args ...any) pgx.Row {
	switch {
	case strings.Contains(query, "SELECT assignee_id FROM tasks"):
		tx.events = append(tx.events, "lock_task")
		return assignmentRow{values: []any{(*uuid.UUID)(nil)}}
	case strings.Contains(query, "SELECT EXISTS"):
		tx.events = append(tx.events, "check_membership")
		return assignmentRow{values: []any{tx.sameTeam}}
	case strings.Contains(query, "UPDATE tasks"):
		tx.events = append(tx.events, "update_task")
		taskID, ownerID, assigneeID := args[0].(uuid.UUID), args[1].(uuid.UUID), args[2].(uuid.UUID)
		assignedAt := uuidPointer(assigneeID)
		createdAt := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
		return assignmentRow{values: []any{taskID, ownerID, assignedAt, "Review design", "", task.StatusPending, createdAt, createdAt}}
	default:
		return assignmentRow{err: errors.New("unexpected query in assignment test")}
	}
}

func (tx *assignmentTransactionStub) Exec(_ context.Context, query string, _ ...any) (pgconn.CommandTag, error) {
	if strings.Contains(query, "INSERT INTO task_logs") {
		tx.events = append(tx.events, "write_task_log")
		return pgconn.NewCommandTag("INSERT 0 1"), nil
	}
	if strings.Contains(query, "INSERT INTO task_notifications") {
		tx.events = append(tx.events, "write_notification")
		if tx.notificationErr != nil {
			return pgconn.CommandTag{}, tx.notificationErr
		}
		return pgconn.NewCommandTag("INSERT 0 1"), nil
	}
	return pgconn.CommandTag{}, errors.New("unexpected exec in assignment test")
}

func (tx *assignmentTransactionStub) Commit(context.Context) error {
	tx.committed = true
	return nil
}

func (tx *assignmentTransactionStub) Rollback(context.Context) error {
	if tx.committed {
		return pgx.ErrTxClosed
	}
	tx.rolledBack = true
	return nil
}

type assignmentRow struct {
	values []any
	err    error
}

func (r assignmentRow) Scan(dest ...any) error {
	if r.err != nil {
		return r.err
	}
	if len(dest) != len(r.values) {
		return errors.New("unexpected scan argument count")
	}
	for i, value := range r.values {
		target := reflect.ValueOf(dest[i])
		if target.Kind() != reflect.Pointer || target.IsNil() {
			return errors.New("scan target must be a non-nil pointer")
		}
		field := target.Elem()
		if value == nil {
			field.SetZero()
			continue
		}
		source := reflect.ValueOf(value)
		if !source.Type().AssignableTo(field.Type()) {
			return errors.New("scan value type does not match target")
		}
		field.Set(source)
	}
	return nil
}

func uuidPointer(value uuid.UUID) *uuid.UUID {
	return &value
}
