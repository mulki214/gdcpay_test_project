package team

import (
	"context"
	"net/mail"
	"strings"

	"github.com/google/uuid"
)

type Repository interface {
	Create(context.Context, Team) (Team, error)
	AddMember(context.Context, uuid.UUID, uuid.UUID, string) (Member, error)
}

type Service struct {
	repository Repository
}

func NewService(repository Repository) *Service {
	return &Service{repository: repository}
}

func (s *Service) Create(ctx context.Context, ownerID uuid.UUID, input CreateInput) (Team, error) {
	name := strings.TrimSpace(input.Name)
	if name == "" || len(name) > 120 {
		return Team{}, ErrInvalidInput
	}
	return s.repository.Create(ctx, Team{ID: uuid.New(), Name: name, OwnerID: ownerID})
}

func (s *Service) AddMember(ctx context.Context, actorID, teamID uuid.UUID, input AddMemberInput) (Member, error) {
	email := strings.ToLower(strings.TrimSpace(input.Email))
	if !validEmail(email) {
		return Member{}, ErrInvalidInput
	}
	return s.repository.AddMember(ctx, actorID, teamID, email)
}

func validEmail(value string) bool {
	if value == "" || len(value) > 254 {
		return false
	}
	parsed, err := mail.ParseAddress(value)
	return err == nil && parsed.Address == value
}
