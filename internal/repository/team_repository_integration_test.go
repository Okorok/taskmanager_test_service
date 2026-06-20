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

func Test_integration_create_and_get_team(t *testing.T) {
	db := testdb.New(t)
	defer db.Close()

	ownerID := createUser(t, db, "Owner")
	repo := repository.NewTeamRepository(db)

	created, err := repo.Create(t.Context(), repository.CreateTeamRequest{Name: "Platform", CreatedBy: ownerID})
	require.NoError(t, err)
	assert.NotZero(t, created.ID)

	loaded, err := repo.GetByID(t.Context(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, created, loaded)
}

func Test_integration_list_teams_by_user(t *testing.T) {
	db := testdb.New(t)
	defer db.Close()

	userID := createUser(t, db, "Member")
	otherUserID := createUser(t, db, "Other")

	teamA := createTeam(t, db, userID, "A")
	teamB := createTeam(t, db, userID, "B")
	teamC := createTeam(t, db, otherUserID, "C")

	addMember(t, db, teamA, userID, string(entity.TeamRoleOwner))
	addMember(t, db, teamB, userID, string(entity.TeamRoleMember))
	addMember(t, db, teamC, otherUserID, string(entity.TeamRoleOwner))

	repo := repository.NewTeamRepository(db)
	teams, err := repo.ListByUser(t.Context(), userID)
	require.NoError(t, err)

	ids := make([]int64, 0, len(teams))
	for _, team := range teams {
		ids = append(ids, team.ID)
	}

	assert.ElementsMatch(t, []int64{teamA, teamB}, ids)
}

func Test_integration_get_team_not_found(t *testing.T) {
	db := testdb.New(t)
	defer db.Close()

	repo := repository.NewTeamRepository(db)
	_, err := repo.GetByID(t.Context(), 999999)

	assert.ErrorIs(t, err, infrastructure.ErrNotFound)
}
