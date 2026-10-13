package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mulki214/gdcpay_test_project/internal/auth"
)

type AuthRepository struct {
	db *pgxpool.Pool
}

func NewAuthRepository(db *pgxpool.Pool) *AuthRepository {
	return &AuthRepository{db: db}
}

func (r *AuthRepository) CreateUser(ctx context.Context, user auth.User) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO users (id, email, password_hash) VALUES ($1, $2, $3)`,
		user.ID, user.Email, user.PasswordHash,
	)
	if err == nil {
		return nil
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return auth.ErrEmailAlreadyRegistered
	}
	return err
}

func (r *AuthRepository) FindUserByEmail(ctx context.Context, email string) (auth.User, error) {
	var user auth.User
	err := r.db.QueryRow(ctx,
		`SELECT id, email, password_hash, created_at, updated_at FROM users WHERE LOWER(email) = $1`, email,
	).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.CreatedAt, &user.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.User{}, auth.ErrUserNotFound
	}
	if err != nil {
		return auth.User{}, err
	}
	return user, nil
}
