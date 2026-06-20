package service

import (
	"errors"
	"testing"

	"taskmanager/internal/entity"
	"taskmanager/internal/service/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_invite_membership_lookup_unexpected_error(t *testing.T) {
	members := mocks.NewMockTeamMemberRepository(t)
	members.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	svc := NewTeamService(passthroughUoW(t), mocks.NewMockTeamRepository(t), members, mocks.NewMockTeamUserRepository(t))

	err := svc.Invite(t.Context(), InviteCommand{TeamID: 10, ActorID: 1, InviteeID: 42, Role: entity.TeamRoleMember})

	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrForbidden)
}

func Test_invite_invitee_lookup_unexpected_error(t *testing.T) {
	members := mocks.NewMockTeamMemberRepository(t)
	members.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(member(10, 1, entity.TeamRoleAdmin), nil)

	users := mocks.NewMockTeamUserRepository(t)
	users.EXPECT().GetByID(mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	svc := NewTeamService(passthroughUoW(t), mocks.NewMockTeamRepository(t), members, users)

	err := svc.Invite(t.Context(), InviteCommand{TeamID: 10, ActorID: 1, InviteeID: 42, Role: entity.TeamRoleMember})

	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrUserNotFound)
}

func Test_invite_add_member_unexpected_error(t *testing.T) {
	members := mocks.NewMockTeamMemberRepository(t)
	members.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(member(10, 1, entity.TeamRoleAdmin), nil)
	members.EXPECT().Add(mock.Anything, mock.Anything).Return(errors.New("db error"))

	users := mocks.NewMockTeamUserRepository(t)
	users.EXPECT().GetByID(mock.Anything, mock.Anything).Return(&entity.User{ID: 42}, nil)

	svc := NewTeamService(passthroughUoW(t), mocks.NewMockTeamRepository(t), members, users)

	err := svc.Invite(t.Context(), InviteCommand{TeamID: 10, ActorID: 1, InviteeID: 42, Role: entity.TeamRoleMember})

	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrAlreadyMember)
}
