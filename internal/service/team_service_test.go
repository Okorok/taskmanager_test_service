package service

import (
	"context"
	"errors"
	"testing"

	"taskmanager/internal/entity"
	"taskmanager/internal/infrastructure"
	"taskmanager/internal/repository"
	"taskmanager/internal/service/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_create_team_adds_owner_member(t *testing.T) {
	team := &entity.Team{ID: 10, Name: "Platform", CreatedBy: 1}

	teams := mocks.NewMockTeamRepository(t)
	teams.EXPECT().
		Create(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req repository.CreateTeamRequest) {
			assert.Equal(t, "Platform", req.Name)
			assert.Equal(t, int64(1), req.CreatedBy)
		}).
		Return(team, nil)

	members := mocks.NewMockTeamMemberRepository(t)
	members.EXPECT().
		Add(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req repository.AddTeamMemberRequest) {
			assert.Equal(t, int64(10), req.TeamID)
			assert.Equal(t, int64(1), req.UserID)
			assert.Equal(t, entity.TeamRoleOwner, req.Role)
		}).
		Return(nil)

	svc := NewTeamService(passthroughUoW(t), teams, members, mocks.NewMockTeamUserRepository(t))

	result, err := svc.CreateTeam(t.Context(), 1, "Platform")

	require.NoError(t, err)
	assert.Equal(t, team, result)
}

func Test_create_team_rolls_back_on_member_error(t *testing.T) {
	teams := mocks.NewMockTeamRepository(t)
	teams.EXPECT().Create(mock.Anything, mock.Anything).Return(&entity.Team{ID: 10}, nil)

	members := mocks.NewMockTeamMemberRepository(t)
	members.EXPECT().Add(mock.Anything, mock.Anything).Return(errors.New("db error"))

	svc := NewTeamService(passthroughUoW(t), teams, members, mocks.NewMockTeamUserRepository(t))

	result, err := svc.CreateTeam(t.Context(), 1, "Platform")

	assert.Error(t, err)
	assert.Nil(t, result)
}

func Test_list_teams(t *testing.T) {
	expected := []entity.Team{{ID: 1}, {ID: 2}}

	teams := mocks.NewMockTeamRepository(t)
	teams.EXPECT().ListByUser(mock.Anything, int64(5)).Return(expected, nil)

	svc := NewTeamService(passthroughUoW(t), teams, mocks.NewMockTeamMemberRepository(t), mocks.NewMockTeamUserRepository(t))

	result, err := svc.ListTeams(t.Context(), 5)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func Test_invite_successfully(t *testing.T) {
	members := mocks.NewMockTeamMemberRepository(t)
	members.EXPECT().Get(mock.Anything, int64(10), int64(1)).Return(member(10, 1, entity.TeamRoleAdmin), nil)
	members.EXPECT().
		Add(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req repository.AddTeamMemberRequest) {
			assert.Equal(t, int64(42), req.UserID)
			assert.Equal(t, entity.TeamRoleMember, req.Role)
		}).
		Return(nil)

	users := mocks.NewMockTeamUserRepository(t)
	users.EXPECT().GetByID(mock.Anything, int64(42)).Return(&entity.User{ID: 42}, nil)

	svc := NewTeamService(passthroughUoW(t), mocks.NewMockTeamRepository(t), members, users)

	err := svc.Invite(t.Context(), InviteCommand{TeamID: 10, ActorID: 1, InviteeID: 42, Role: entity.TeamRoleMember})

	require.NoError(t, err)
}

func Test_invite_actor_not_in_team(t *testing.T) {
	members := mocks.NewMockTeamMemberRepository(t)
	members.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil, infrastructure.ErrNotFound)

	svc := NewTeamService(passthroughUoW(t), mocks.NewMockTeamRepository(t), members, mocks.NewMockTeamUserRepository(t))

	err := svc.Invite(t.Context(), InviteCommand{TeamID: 10, ActorID: 1, InviteeID: 42, Role: entity.TeamRoleMember})

	assert.ErrorIs(t, err, ErrForbidden)
}

func Test_invite_actor_is_plain_member(t *testing.T) {
	members := mocks.NewMockTeamMemberRepository(t)
	members.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(member(10, 1, entity.TeamRoleMember), nil)

	svc := NewTeamService(passthroughUoW(t), mocks.NewMockTeamRepository(t), members, mocks.NewMockTeamUserRepository(t))

	err := svc.Invite(t.Context(), InviteCommand{TeamID: 10, ActorID: 1, InviteeID: 42, Role: entity.TeamRoleMember})

	assert.ErrorIs(t, err, ErrForbidden)
}

func Test_invite_with_owner_role_is_rejected(t *testing.T) {
	members := mocks.NewMockTeamMemberRepository(t)
	members.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(member(10, 1, entity.TeamRoleOwner), nil)

	svc := NewTeamService(passthroughUoW(t), mocks.NewMockTeamRepository(t), members, mocks.NewMockTeamUserRepository(t))

	err := svc.Invite(t.Context(), InviteCommand{TeamID: 10, ActorID: 1, InviteeID: 42, Role: entity.TeamRoleOwner})

	assert.ErrorIs(t, err, ErrInvalidRole)
}

func Test_invite_invalid_role(t *testing.T) {
	members := mocks.NewMockTeamMemberRepository(t)
	members.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(member(10, 1, entity.TeamRoleOwner), nil)

	svc := NewTeamService(passthroughUoW(t), mocks.NewMockTeamRepository(t), members, mocks.NewMockTeamUserRepository(t))

	err := svc.Invite(t.Context(), InviteCommand{TeamID: 10, ActorID: 1, InviteeID: 42, Role: entity.TeamRole("ghost")})

	assert.ErrorIs(t, err, ErrInvalidRole)
}

func Test_invite_invitee_not_found(t *testing.T) {
	members := mocks.NewMockTeamMemberRepository(t)
	members.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(member(10, 1, entity.TeamRoleAdmin), nil)

	users := mocks.NewMockTeamUserRepository(t)
	users.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, infrastructure.ErrNotFound)

	svc := NewTeamService(passthroughUoW(t), mocks.NewMockTeamRepository(t), members, users)

	err := svc.Invite(t.Context(), InviteCommand{TeamID: 10, ActorID: 1, InviteeID: 42, Role: entity.TeamRoleMember})

	assert.ErrorIs(t, err, ErrUserNotFound)
}

func Test_invite_already_member(t *testing.T) {
	members := mocks.NewMockTeamMemberRepository(t)
	members.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(member(10, 1, entity.TeamRoleAdmin), nil)
	members.EXPECT().Add(mock.Anything, mock.Anything).Return(infrastructure.ErrAlreadyExists)

	users := mocks.NewMockTeamUserRepository(t)
	users.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&entity.User{ID: 42}, nil)

	svc := NewTeamService(passthroughUoW(t), mocks.NewMockTeamRepository(t), members, users)

	err := svc.Invite(t.Context(), InviteCommand{TeamID: 10, ActorID: 1, InviteeID: 42, Role: entity.TeamRoleMember})

	assert.ErrorIs(t, err, ErrAlreadyMember)
}
