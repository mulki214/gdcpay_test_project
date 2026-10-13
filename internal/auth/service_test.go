package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

type memoryRepository struct {
	users map[string]User
}

func newMemoryRepository() *memoryRepository {
	return &memoryRepository{users: make(map[string]User)}
}

func (r *memoryRepository) CreateUser(_ context.Context, user User) error {
	if _, exists := r.users[user.Email]; exists {
		return ErrEmailAlreadyRegistered
	}
	r.users[user.Email] = user
	return nil
}

func (r *memoryRepository) FindUserByEmail(_ context.Context, email string) (User, error) {
	user, exists := r.users[email]
	if !exists {
		return User{}, ErrUserNotFound
	}
	return user, nil
}

type testHasher struct{}

func (testHasher) Hash(password string) (string, error) {
	return "hashed:" + password, nil
}

func (testHasher) Compare(hash, password string) bool {
	return hash == "hashed:"+password
}

type testTokenIssuer struct{}

func (testTokenIssuer) Issue(user User) (string, time.Time, error) {
	return "token-for-" + user.ID.String(), time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC), nil
}

func newTestService(repo *memoryRepository) *Service {
	return NewService(repo, testHasher{}, testTokenIssuer{})
}

func TestRegisterNormalizesEmailAndStoresPasswordHash(t *testing.T) {
	repo := newMemoryRepository()
	service := newTestService(repo)

	result, err := service.Register(context.Background(), " User@Example.com ", "long-password")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	if result.User.Email != "user@example.com" {
		t.Fatalf("registered email = %q, want normalized email", result.User.Email)
	}
	if result.User.PasswordHash != "hashed:long-password" {
		t.Fatalf("password hash = %q, want hashed value", result.User.PasswordHash)
	}
	if result.AccessToken == "" {
		t.Fatal("Register() returned an empty access token")
	}
	if _, exists := repo.users[result.User.Email]; !exists {
		t.Fatal("registered user was not stored")
	}
}

func TestRegisterRejectsInvalidCredentials(t *testing.T) {
	service := newTestService(newMemoryRepository())
	tests := []struct {
		name     string
		email    string
		password string
	}{
		{name: "invalid email", email: "not-an-email", password: "long-password"},
		{name: "short password", email: "user@example.com", password: "short"},
		{name: "password longer than bcrypt limit", email: "user@example.com", password: string(make([]byte, 73))},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := service.Register(context.Background(), tt.email, tt.password)
			if !errors.Is(err, ErrInvalidInput) {
				t.Fatalf("Register() error = %v, want %v", err, ErrInvalidInput)
			}
		})
	}
}

func TestRegisterRejectsDuplicateEmail(t *testing.T) {
	repo := newMemoryRepository()
	service := newTestService(repo)
	if _, err := service.Register(context.Background(), "user@example.com", "long-password"); err != nil {
		t.Fatalf("first Register() error = %v", err)
	}
	_, err := service.Register(context.Background(), "USER@example.com", "another-password")
	if !errors.Is(err, ErrEmailAlreadyRegistered) {
		t.Fatalf("second Register() error = %v, want %v", err, ErrEmailAlreadyRegistered)
	}
}

func TestLoginReturnsGenericInvalidCredentials(t *testing.T) {
	service := newTestService(newMemoryRepository())
	_, err := service.Login(context.Background(), "missing@example.com", "long-password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want generic credentials error", err)
	}
}

func TestLoginSucceedsWithRegisteredPassword(t *testing.T) {
	repo := newMemoryRepository()
	service := newTestService(repo)
	registered, err := service.Register(context.Background(), "user@example.com", "long-password")
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	result, err := service.Login(context.Background(), "USER@example.com", "long-password")
	if err != nil {
		t.Fatalf("Login() error = %v", err)
	}
	if result.User.ID != registered.User.ID || result.AccessToken == "" {
		t.Fatalf("Login() result = %+v, want registered user and token", result)
	}
}

func TestLoginRejectsIncorrectPassword(t *testing.T) {
	repo := newMemoryRepository()
	service := newTestService(repo)
	if _, err := service.Register(context.Background(), "user@example.com", "long-password"); err != nil {
		t.Fatalf("Register() error = %v", err)
	}
	_, err := service.Login(context.Background(), "user@example.com", "incorrect-password")
	if !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("Login() error = %v, want generic credentials error", err)
	}
}
