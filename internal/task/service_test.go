package task

import (
	"context"
	"crypto/sha256"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestCreateIdempotentReturnsSameTaskForSequentialDuplicate(t *testing.T) {
	repository := newMemoryTaskRepository()
	service := NewService(repository)
	ownerID := uuid.New()
	key := uuid.New()
	input := CreateInput{Title: "  Prepare report  ", Description: "Quarterly summary"}

	first, err := service.CreateIdempotent(context.Background(), ownerID, key, input)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	second, err := service.CreateIdempotent(context.Background(), ownerID, key, CreateInput{Title: "Prepare report", Description: "Quarterly summary"})
	if err != nil {
		t.Fatalf("duplicate create: %v", err)
	}
	if first != second {
		t.Fatalf("duplicate returned a different task:\nfirst:  %+v\nsecond: %+v", first, second)
	}
	if got := repository.createdCount(); got != 1 {
		t.Fatalf("created task count = %d, want 1", got)
	}
}

func TestCreateIdempotentConcurrentDuplicateCreatesOneTask(t *testing.T) {
	const requestCount = 64
	repository := newMemoryTaskRepository()
	service := NewService(repository)
	ownerID := uuid.New()
	key := uuid.New()
	start := make(chan struct{})
	results := make([]Task, requestCount)
	errs := make([]error, requestCount)
	var wait sync.WaitGroup
	wait.Add(requestCount)

	for i := range requestCount {
		go func(index int) {
			defer wait.Done()
			<-start
			results[index], errs[index] = service.CreateIdempotent(context.Background(), ownerID, key, CreateInput{
				Title: "Concurrent task", Description: "Only one should be created",
			})
		}(i)
	}
	close(start)
	wait.Wait()

	for i, err := range errs {
		if err != nil {
			t.Fatalf("request %d failed: %v", i, err)
		}
		if results[i] != results[0] {
			t.Fatalf("request %d returned a different task", i)
		}
	}
	if got := repository.createdCount(); got != 1 {
		t.Fatalf("created task count = %d, want 1", got)
	}
}

func TestCreateIdempotentRejectsKeyReusedWithDifferentInput(t *testing.T) {
	repository := newMemoryTaskRepository()
	service := NewService(repository)
	ownerID := uuid.New()
	key := uuid.New()

	if _, err := service.CreateIdempotent(context.Background(), ownerID, key, CreateInput{Title: "First task"}); err != nil {
		t.Fatalf("first create: %v", err)
	}
	if _, err := service.CreateIdempotent(context.Background(), ownerID, key, CreateInput{Title: "Different task"}); !errors.Is(err, ErrIdempotencyKeyReused) {
		t.Fatalf("second create error = %v, want %v", err, ErrIdempotencyKeyReused)
	}
	if got := repository.createdCount(); got != 1 {
		t.Fatalf("created task count = %d, want 1", got)
	}
}

func TestCreateIdempotentExpiresWithoutWaitingForTTL(t *testing.T) {
	repository := newMemoryTaskRepository()
	now := time.Date(2026, time.January, 1, 0, 0, 0, 0, time.UTC)
	service := NewServiceWithClock(repository, func() time.Time { return now })
	ownerID := uuid.New()
	key := uuid.New()
	input := CreateInput{Title: "Task after expiry"}

	first, err := service.CreateIdempotent(context.Background(), ownerID, key, input)
	if err != nil {
		t.Fatalf("first create: %v", err)
	}
	now = now.Add(IdempotencyTTL + time.Nanosecond)
	second, err := service.CreateIdempotent(context.Background(), ownerID, key, input)
	if err != nil {
		t.Fatalf("create after expiry: %v", err)
	}
	if first.ID == second.ID {
		t.Fatal("expired key returned the task from the previous window")
	}
	if got := repository.createdCount(); got != 2 {
		t.Fatalf("created task count = %d, want 2", got)
	}
}

type memoryTaskRepository struct {
	mu      sync.Mutex
	items   map[memoryIdempotencyKey]memoryIdempotencyRecord
	created int
}

type memoryIdempotencyKey struct {
	owner uuid.UUID
	key   uuid.UUID
}

type memoryIdempotencyRecord struct {
	hash      [sha256.Size]byte
	item      Task
	expiresAt time.Time
}

func newMemoryTaskRepository() *memoryTaskRepository {
	return &memoryTaskRepository{items: make(map[memoryIdempotencyKey]memoryIdempotencyRecord)}
}

func (r *memoryTaskRepository) Create(_ context.Context, item Task) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.created++
	return item, nil
}

func (r *memoryTaskRepository) CreateIdempotent(_ context.Context, item Task, key uuid.UUID, hash [sha256.Size]byte, expiresAt time.Time) (Task, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	mapKey := memoryIdempotencyKey{owner: item.OwnerID, key: key}
	now := expiresAt.Add(-IdempotencyTTL)
	if existing, ok := r.items[mapKey]; ok && existing.expiresAt.After(now) {
		if existing.hash != hash {
			return Task{}, ErrIdempotencyKeyReused
		}
		return existing.item, nil
	}
	item.CreatedAt = time.Unix(1, 0).UTC()
	item.UpdatedAt = item.CreatedAt
	r.items[mapKey] = memoryIdempotencyRecord{hash: hash, item: item, expiresAt: expiresAt}
	r.created++
	return item, nil
}

func (r *memoryTaskRepository) List(context.Context, uuid.UUID, ListFilter) ([]Task, int64, error) {
	return nil, 0, nil
}

func (r *memoryTaskRepository) GetByID(context.Context, uuid.UUID, uuid.UUID) (Task, error) {
	return Task{}, ErrNotFound
}

func (r *memoryTaskRepository) Update(context.Context, uuid.UUID, uuid.UUID, UpdateInput) (Task, error) {
	return Task{}, ErrNotFound
}

func (r *memoryTaskRepository) Delete(context.Context, uuid.UUID, uuid.UUID) error {
	return ErrNotFound
}

func (r *memoryTaskRepository) createdCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.created
}
