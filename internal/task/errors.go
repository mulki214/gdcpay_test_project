package task

import "errors"

var (
	ErrInvalidInput         = errors.New("invalid task input")
	ErrNotFound             = errors.New("task not found")
	ErrIdempotencyKeyReused = errors.New("idempotency key was used with a different request")
	ErrInvalidAssignee      = errors.New("assignee must be a different user")
	ErrAssigneeNotInTeam    = errors.New("assignee is not in the same team")
)
