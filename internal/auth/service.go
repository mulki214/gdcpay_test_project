package auth

import (
	"context"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Repository interface {
	CreateUser(context.Context, User) error
	FindUserByEmail(context.Context, string) (User, error)
}

type TokenIssuer interface {
	Issue(User) (string, time.Time, error)
}

type Service struct {
	repository Repository
	hasher     PasswordHasher
	tokens     TokenIssuer
}

func NewService(repository Repository, hasher PasswordHasher, tokens TokenIssuer) *Service {
	return &Service{repository: repository, hasher: hasher, tokens: tokens}
}

func (s *Service) Register(ctx context.Context, email, password string) (AuthResult, error) {
	email = normalizeEmail(email)
	if !validEmail(email) || len(password) < 8 || len(password) > 72 {
		return AuthResult{}, ErrInvalidInput
	}

	passwordHash, err := s.hasher.Hash(password)
	if err != nil {
		return AuthResult{}, err
	}
	user := User{ID: uuid.New(), Email: email, PasswordHash: passwordHash}
	if err := s.repository.CreateUser(ctx, user); err != nil {
		return AuthResult{}, err
	}
	return s.issue(user)
}

func (s *Service) Login(ctx context.Context, email, password string) (AuthResult, error) {
	email = normalizeEmail(email)
	if !validEmail(email) || password == "" {
		return AuthResult{}, ErrInvalidCredentials
	}
	user, err := s.repository.FindUserByEmail(ctx, email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return AuthResult{}, ErrInvalidCredentials
		}
		return AuthResult{}, err
	}
	if !s.hasher.Compare(user.PasswordHash, password) {
		return AuthResult{}, ErrInvalidCredentials
	}
	return s.issue(user)
}

func (s *Service) issue(user User) (AuthResult, error) {
	token, expiresAt, err := s.tokens.Issue(user)
	if err != nil {
		return AuthResult{}, err
	}
	return AuthResult{User: user, AccessToken: token, ExpiresAt: expiresAt}, nil
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func validEmail(email string) bool {
	if len(email) == 0 || len(email) > 254 {
		return false
	}
	parsed, err := mail.ParseAddress(email)
	return err == nil && parsed.Address == email
}
