package service

import (
	"context"

	"taskmanager/internal/entity"
	"taskmanager/internal/infrastructure"
	"taskmanager/internal/repository"

	"github.com/pkg/errors"
)

var (
	ErrUserNotFound  = errors.New("user not found")
	ErrAlreadyMember = errors.New("user is already a team member")
	ErrInvalidRole   = errors.New("invalid role")
)

type TeamRepository interface {
	Create(ctx context.Context, request repository.CreateTeamRequest) (*entity.Team, error)
	GetByID(ctx context.Context, id int64) (*entity.Team, error)
	ListByUser(ctx context.Context, userID int64) ([]entity.Team, error)
}

type TeamMemberRepository interface {
	Add(ctx context.Context, request repository.AddTeamMemberRequest) error
	Get(ctx context.Context, teamID, userID int64) (*entity.TeamMember, error)
}

type TeamUserRepository interface {
	GetByID(ctx context.Context, id int64) (*entity.User, error)
}

type TeamService struct {
	uow     UnitOfWork
	teams   TeamRepository
	members TeamMemberRepository
	users   TeamUserRepository
}

func NewTeamService(uow UnitOfWork, teams TeamRepository, members TeamMemberRepository, users TeamUserRepository) *TeamService {
	return &TeamService{
		uow:     uow,
		teams:   teams,
		members: members,
		users:   users,
	}
}

func (s *TeamService) CreateTeam(ctx context.Context, ownerID int64, name string) (*entity.Team, error) {
	var team *entity.Team

	err := s.uow.Do(ctx, func(ctx context.Context) error {
		created, err := s.teams.Create(ctx, repository.CreateTeamRequest{
			Name:      name,
			CreatedBy: ownerID,
		})
		if err != nil {
			return err
		}

		if err := s.members.Add(ctx, repository.AddTeamMemberRequest{
			TeamID: created.ID,
			UserID: ownerID,
			Role:   entity.TeamRoleOwner,
		}); err != nil {
			return err
		}

		team = created
		return nil
	})
	if err != nil {
		return nil, err
	}

	return team, nil
}

func (s *TeamService) ListTeams(ctx context.Context, userID int64) ([]entity.Team, error) {
	return s.teams.ListByUser(ctx, userID)
}

type InviteCommand struct {
	TeamID    int64
	ActorID   int64
	InviteeID int64
	Role      entity.TeamRole
}

func (s *TeamService) Invite(ctx context.Context, cmd InviteCommand) error {
	actor, err := s.members.Get(ctx, cmd.TeamID, cmd.ActorID)
	if err != nil {
		if errors.Is(err, infrastructure.ErrNotFound) {
			return ErrForbidden
		}

		return err
	}

	if !actor.Role.CanInvite() {
		return ErrForbidden
	}

	if !cmd.Role.IsValid() || cmd.Role == entity.TeamRoleOwner {
		return ErrInvalidRole
	}

	if _, err := s.users.GetByID(ctx, cmd.InviteeID); err != nil {
		if errors.Is(err, infrastructure.ErrNotFound) {
			return ErrUserNotFound
		}

		return err
	}

	if err := s.members.Add(ctx, repository.AddTeamMemberRequest{
		TeamID: cmd.TeamID,
		UserID: cmd.InviteeID,
		Role:   cmd.Role,
	}); err != nil {
		if errors.Is(err, infrastructure.ErrAlreadyExists) {
			return ErrAlreadyMember
		}

		return err
	}

	return nil
}
