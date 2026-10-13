package task

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	Create(context.Context, Task) (Task, error)
	CreateIdempotent(context.Context, Task, uuid.UUID, [sha256.Size]byte, time.Time) (Task, error)
	Assign(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (Task, error)
	List(context.Context, uuid.UUID, ListFilter) ([]Task, int64, error)
	GetByID(context.Context, uuid.UUID, uuid.UUID) (Task, error)
	Update(context.Context, uuid.UUID, uuid.UUID, UpdateInput) (Task, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
}

type Service struct {
	repository Repository
	now        func() time.Time
}

const IdempotencyTTL = 24 * time.Hour

func NewService(repository Repository) *Service {
	return NewServiceWithClock(repository, time.Now)
}

func NewServiceWithClock(repository Repository, now func() time.Time) *Service {
	if now == nil {
		now = time.Now
	}
	return &Service{repository: repository, now: now}
}

func (s *Service) Create(ctx context.Context, ownerID uuid.UUID, input CreateInput) (Task, error) {
	input, err := normalizeCreateInput(input)
	if err != nil {
		return Task{}, ErrInvalidInput
	}
	return s.repository.Create(ctx, Task{
		ID:          uuid.New(),
		OwnerID:     ownerID,
		Title:       input.Title,
		Description: input.Description,
		Status:      StatusPending,
	})
}

func (s *Service) CreateIdempotent(ctx context.Context, ownerID, key uuid.UUID, input CreateInput) (Task, error) {
	input, err := normalizeCreateInput(input)
	if err != nil {
		return Task{}, ErrInvalidInput
	}
	requestBody, err := json.Marshal(input)
	if err != nil {
		return Task{}, err
	}
	requestHash := sha256.Sum256(requestBody)
	return s.repository.CreateIdempotent(ctx, Task{
		ID:          uuid.New(),
		OwnerID:     ownerID,
		Title:       input.Title,
		Description: input.Description,
		Status:      StatusPending,
	}, key, requestHash, s.now().Add(IdempotencyTTL))
}

func normalizeCreateInput(input CreateInput) (CreateInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" || len(input.Title) > 200 || len(input.Description) > 5000 {
		return CreateInput{}, ErrInvalidInput
	}
	return input, nil
}

func (s *Service) Assign(ctx context.Context, ownerID, taskID, assigneeID uuid.UUID) (Task, error) {
	if assigneeID == uuid.Nil || assigneeID == ownerID {
		return Task{}, ErrInvalidAssignee
	}
	return s.repository.Assign(ctx, ownerID, taskID, assigneeID)
}

func (s *Service) List(ctx context.Context, ownerID uuid.UUID, filter ListFilter) (Page, error) {
	if filter.Page < 1 || filter.Limit < 1 || filter.Limit > 100 || (filter.Status != "" && !filter.Status.Valid()) || len(filter.Search) > 200 {
		return Page{}, ErrInvalidInput
	}
	filter.Search = strings.TrimSpace(filter.Search)
	tasks, total, err := s.repository.List(ctx, ownerID, filter)
	if err != nil {
		return Page{}, err
	}
	if tasks == nil {
		tasks = []Task{}
	}
	return Page{Tasks: tasks, Page: filter.Page, Limit: filter.Limit, Total: total}, nil
}

func (s *Service) Get(ctx context.Context, ownerID, taskID uuid.UUID) (Task, error) {
	return s.repository.GetByID(ctx, ownerID, taskID)
}

func (s *Service) Update(ctx context.Context, ownerID, taskID uuid.UUID, input UpdateInput) (Task, error) {
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" || len(input.Title) > 200 || len(input.Description) > 5000 || !input.Status.Valid() {
		return Task{}, ErrInvalidInput
	}
	return s.repository.Update(ctx, ownerID, taskID, input)
}

func (s *Service) Delete(ctx context.Context, ownerID, taskID uuid.UUID) error {
	return s.repository.Delete(ctx, ownerID, taskID)
}
