package team

import "errors"

var (
	ErrInvalidInput = errors.New("invalid team input")
	ErrNotFound     = errors.New("team not found")
	ErrForbidden    = errors.New("team owner permission required")
	ErrUserNotFound = errors.New("user not found")
)
