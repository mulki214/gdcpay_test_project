package team

import (
	"time"

	"github.com/google/uuid"
)

const (
	RoleOwner  = "owner"
	RoleMember = "member"
)

type Team struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	OwnerID   uuid.UUID `json:"owner_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Member struct {
	TeamID    uuid.UUID `json:"team_id"`
	UserID    uuid.UUID `json:"user_id"`
	Email     string    `json:"email"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateInput struct {
	Name string
}

type AddMemberInput struct {
	Email string
}
