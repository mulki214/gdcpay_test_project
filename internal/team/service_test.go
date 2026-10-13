package team

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

func TestCreateTrimsTeamName(t *testing.T) {
	repository := &teamRepositoryStub{}
	service := NewService(repository)
	ownerID := uuid.New()

	created, err := service.Create(context.Background(), ownerID, CreateInput{Name: "  Product Team  "})
	if err != nil {
		t.Fatalf("create team: %v", err)
	}
	if created.Name != "Product Team" || created.OwnerID != ownerID || repository.createCalls != 1 {
		t.Fatalf("unexpected created team: %+v; create calls = %d", created, repository.createCalls)
	}
}

func TestCreateRejectsInvalidTeamName(t *testing.T) {
	service := NewService(&teamRepositoryStub{})
	for _, name := range []string{"", "   ", string(make([]byte, 121))} {
		if _, err := service.Create(context.Background(), uuid.New(), CreateInput{Name: name}); !errors.Is(err, ErrInvalidInput) {
			t.Errorf("create with %q error = %v, want %v", name, err, ErrInvalidInput)
		}
	}
}

func TestAddMemberNormalizesEmail(t *testing.T) {
	repository := &teamRepositoryStub{}
	service := NewService(repository)
	actorID, teamID := uuid.New(), uuid.New()

	_, err := service.AddMember(context.Background(), actorID, teamID, AddMemberInput{Email: "  User@Example.com "})
	if err != nil {
		t.Fatalf("add member: %v", err)
	}
	if repository.memberEmail != "user@example.com" {
		t.Fatalf("repository email = %q, want normalized email", repository.memberEmail)
	}
}

func TestAddMemberRejectsInvalidEmail(t *testing.T) {
	repository := &teamRepositoryStub{}
	service := NewService(repository)
	if _, err := service.AddMember(context.Background(), uuid.New(), uuid.New(), AddMemberInput{Email: "not-an-email"}); !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("add member error = %v, want %v", err, ErrInvalidInput)
	}
	if repository.addMemberCalls != 0 {
		t.Fatal("repository was called for invalid email")
	}
}

type teamRepositoryStub struct {
	createCalls    int
	addMemberCalls int
	memberEmail    string
}

func (r *teamRepositoryStub) Create(_ context.Context, item Team) (Team, error) {
	r.createCalls++
	return item, nil
}

func (r *teamRepositoryStub) AddMember(_ context.Context, _, teamID uuid.UUID, email string) (Member, error) {
	r.addMemberCalls++
	r.memberEmail = email
	return Member{TeamID: teamID, Email: email}, nil
}
