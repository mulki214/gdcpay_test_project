package postgres

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mulki214/gdcpay_test_project/internal/team"
)

type TeamRepository struct {
	db *pgxpool.Pool
}

func NewTeamRepository(db *pgxpool.Pool) *TeamRepository {
	return &TeamRepository{db: db}
}

func (r *TeamRepository) Create(ctx context.Context, item team.Team) (team.Team, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return team.Team{}, err
	}
	defer tx.Rollback(ctx)

	err = tx.QueryRow(ctx, `
		INSERT INTO teams (id, name, owner_id)
		VALUES ($1, $2, $3)
		RETURNING id, name, owner_id, created_at`, item.ID, item.Name, item.OwnerID,
	).Scan(&item.ID, &item.Name, &item.OwnerID, &item.CreatedAt)
	if err != nil {
		return team.Team{}, err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO team_members (team_id, user_id, role)
		VALUES ($1, $2, $3)`, item.ID, item.OwnerID, team.RoleOwner,
	); err != nil {
		return team.Team{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return team.Team{}, err
	}
	return item, nil
}

func (r *TeamRepository) AddMember(ctx context.Context, actorID, teamID uuid.UUID, email string) (team.Member, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return team.Member{}, err
	}
	defer tx.Rollback(ctx)

	var ownerID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT owner_id FROM teams WHERE id = $1 FOR UPDATE`, teamID).Scan(&ownerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return team.Member{}, team.ErrNotFound
	}
	if err != nil {
		return team.Member{}, err
	}
	if ownerID != actorID {
		return team.Member{}, team.ErrForbidden
	}

	member := team.Member{TeamID: teamID, Email: email, Role: team.RoleMember}
	err = tx.QueryRow(ctx, `SELECT id FROM users WHERE LOWER(email) = $1`, email).Scan(&member.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		return team.Member{}, team.ErrUserNotFound
	}
	if err != nil {
		return team.Member{}, err
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO team_members (team_id, user_id, role)
		VALUES ($1, $2, $3)
		ON CONFLICT (team_id, user_id) DO NOTHING
		RETURNING created_at`, teamID, member.UserID, member.Role,
	).Scan(&member.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tx.QueryRow(ctx, `
			SELECT created_at FROM team_members WHERE team_id = $1 AND user_id = $2`, teamID, member.UserID,
		).Scan(&member.CreatedAt)
	}
	if err != nil {
		return team.Member{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return team.Member{}, err
	}
	return member, nil
}
