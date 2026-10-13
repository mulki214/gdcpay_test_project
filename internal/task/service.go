package task

import (
	"context"
	"strings"

	"github.com/google/uuid"
)

type Repository interface {
	Create(context.Context, Task) (Task, error)
	List(context.Context, uuid.UUID, ListFilter) ([]Task, int64, error)
	GetByID(context.Context, uuid.UUID, uuid.UUID) (Task, error)
	Update(context.Context, uuid.UUID, uuid.UUID, UpdateInput) (Task, error)
	Delete(context.Context, uuid.UUID, uuid.UUID) error
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, ownerID uuid.UUID, input CreateInput) (Task, error) {
	input.Title = strings.TrimSpace(input.Title)
	if input.Title == "" || len(input.Title) > 200 || len(input.Description) > 5000 {
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
