//go:build integration

package repository_test

import (
	"context"
	"errors"
	"testing"

	"taskmanager/internal/entity"
	"taskmanager/internal/infrastructure"
	"taskmanager/internal/infrastructure/testdb"
	"taskmanager/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_integration_unit_of_work_commits_atomically(t *testing.T) {
	db := testdb.New(t)
	defer db.Close()

	userID := createUser(t, db, "Owner")

	uow := infrastructure.NewUnitOfWork(db)
	teamRepo := repository.NewTeamRepository(db)
	memberRepo := repository.NewTeamMemberRepository(db)

	var teamID int64
	err := uow.Do(t.Context(), func(ctx context.Context) error {
		team, err := teamRepo.Create(ctx, repository.CreateTeamRequest{Name: "Tx", CreatedBy: userID})
		if err != nil {
			return err
		}
		teamID = team.ID

		return memberRepo.Add(ctx, repository.AddTeamMemberRequest{
			TeamID: team.ID,
			UserID: userID,
			Role:   entity.TeamRoleOwner,
		})
	})
	require.NoError(t, err)

	member, err := memberRepo.Get(t.Context(), teamID, userID)
	require.NoError(t, err)
	assert.Equal(t, entity.TeamRoleOwner, member.Role)
}

func Test_integration_unit_of_work_rolls_back_on_error(t *testing.T) {
	db := testdb.New(t)
	defer db.Close()

	userID := createUser(t, db, "Owner")

	uow := infrastructure.NewUnitOfWork(db)
	teamRepo := repository.NewTeamRepository(db)

	var teamID int64
	err := uow.Do(t.Context(), func(ctx context.Context) error {
		team, err := teamRepo.Create(ctx, repository.CreateTeamRequest{Name: "Rollback", CreatedBy: userID})
		if err != nil {
			return err
		}
		teamID = team.ID

		return errors.New("forced failure")
	})
	require.Error(t, err)

	_, err = teamRepo.GetByID(t.Context(), teamID)
	assert.ErrorIs(t, err, infrastructure.ErrNotFound)
}
