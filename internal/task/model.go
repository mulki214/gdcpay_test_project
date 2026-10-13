package task

import (
	"time"

	"github.com/google/uuid"
)

type Status string

const (
	StatusPending    Status = "pending"
	StatusInProgress Status = "in_progress"
	StatusCompleted  Status = "completed"
)

func (s Status) Valid() bool {
	return s == StatusPending || s == StatusInProgress || s == StatusCompleted
}

type Task struct {
	ID          uuid.UUID  `json:"id"`
	OwnerID     uuid.UUID  `json:"owner_id"`
	AssigneeID  *uuid.UUID `json:"assignee_id,omitempty"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      Status     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type CreateInput struct {
	Title       string
	Description string
}

type UpdateInput struct {
	Title       string
	Description string
	Status      Status
}

type ListFilter struct {
	Status Status
	Search string
	Page   int
	Limit  int
}

type Page struct {
	Tasks []Task `json:"tasks"`
	Page  int    `json:"page"`
	Limit int    `json:"limit"`
	Total int64  `json:"total"`
}
