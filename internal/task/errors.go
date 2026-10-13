package task

import "errors"

var (
	ErrInvalidInput = errors.New("invalid task input")
	ErrNotFound     = errors.New("task not found")
)
