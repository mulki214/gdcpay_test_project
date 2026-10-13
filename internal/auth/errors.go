package auth

import "errors"

var (
	ErrInvalidInput           = errors.New("invalid authentication input")
	ErrEmailAlreadyRegistered = errors.New("email is already registered")
	ErrInvalidCredentials     = errors.New("invalid email or password")
	ErrUserNotFound           = errors.New("user not found")
)
