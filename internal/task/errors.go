package task

import "errors"

var (
	ErrInvalidInput         = errors.New("invalid task input")
	ErrNotFound             = errors.New("task not found")
	ErrIdempotencyKeyReused = errors.New("idempotency key was used with a different request")
)
