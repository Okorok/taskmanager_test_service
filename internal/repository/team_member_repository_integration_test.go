//go:build integration

package repository_test

import (
	"testing"

	"taskmanager/internal/entity"
	"taskmanager/internal/infrastructure"
	"taskmanager/internal/infrastructure/testdb"
	"taskmanager/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_integration_add_and_get_team_member(t *testing.T) {
	db := testdb.New(t)
	defer db.Close()

	ownerID := createUser(t, db, "Owner")
	teamID := createTeam(t, db, ownerID, "Team")

	repo := repository.NewTeamMemberRepository(db)
	err := repo.Add(t.Context(), repository.AddTeamMemberRequest{
		TeamID: teamID,
		UserID: ownerID,
		Role:   entity.TeamRoleOwner,
	})
	require.NoError(t, err)

	member, err := repo.Get(t.Context(), teamID, ownerID)
	require.NoError(t, err)
	assert.Equal(t, entity.TeamRoleOwner, member.Role)
}

func Test_integration_add_duplicate_team_member(t *testing.T) {
	db := testdb.New(t)
	defer db.Close()

	userID := createUser(t, db, "User")
	teamID := createTeam(t, db, userID, "Team")

	repo := repository.NewTeamMemberRepository(db)
	req := repository.AddTeamMemberRequest{TeamID: teamID, UserID: userID, Role: entity.TeamRoleMember}

	require.NoError(t, repo.Add(t.Context(), req))

	err := repo.Add(t.Context(), req)
	assert.ErrorIs(t, err, infrastructure.ErrAlreadyExists)
}

func Test_integration_get_team_member_not_found(t *testing.T) {
	db := testdb.New(t)
	defer db.Close()

	repo := repository.NewTeamMemberRepository(db)
	_, err := repo.Get(t.Context(), 999999, 999999)

	assert.ErrorIs(t, err, infrastructure.ErrNotFound)
}
