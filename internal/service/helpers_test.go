package service

import (
	"context"
	"testing"

	"taskmanager/internal/entity"
	"taskmanager/internal/service/mocks"

	"github.com/stretchr/testify/mock"
)

func member(teamID, userID int64, role entity.TeamRole) *entity.TeamMember {
	return &entity.TeamMember{TeamID: teamID, UserID: userID, Role: role}
}

func memberForAll(t *testing.T) *mocks.MockMembershipRepository {
	t.Helper()

	m := mocks.NewMockMembershipRepository(t)
	m.EXPECT().
		Get(mock.Anything, mock.Anything, mock.Anything).
		Return(member(0, 0, entity.TeamRoleMember), nil).
		Maybe()

	return m
}

func passthroughUoW(t *testing.T) *mocks.MockUnitOfWork {
	t.Helper()

	m := mocks.NewMockUnitOfWork(t)
	m.EXPECT().
		Do(mock.Anything, mock.Anything).
		RunAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).
		Maybe()

	return m
}
