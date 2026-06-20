//go:build integration

package repository_test

import (
	"testing"

	"taskmanager/internal/entity"
	"taskmanager/internal/infrastructure/testdb"
	"taskmanager/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Запрос (а): JOIN 3+ таблиц + агрегация.
func Test_integration_analytics_team_stats(t *testing.T) {
	db := testdb.New(t)
	defer db.Close()

	owner := createUser(t, db, "Owner")
	member := createUser(t, db, "Member")
	teamID := createTeam(t, db, owner, "StatsTeam")
	addMember(t, db, teamID, owner, string(entity.TeamRoleOwner))
	addMember(t, db, teamID, member, string(entity.TeamRoleMember))

	// 2 задачи done (updated_at = now) + 1 todo.
	insertTask(t, db, teamID, owner, string(entity.TaskStatusDone), nil)
	insertTask(t, db, teamID, owner, string(entity.TaskStatusDone), nil)
	insertTask(t, db, teamID, owner, string(entity.TaskStatusTodo), nil)

	repo := repository.NewAnalyticsRepository(db)
	stats, err := repo.TeamStats(t.Context())
	require.NoError(t, err)

	var found *repository.TeamStats
	for i := range stats {
		if stats[i].TeamID == teamID {
			found = &stats[i]
			break
		}
	}

	require.NotNil(t, found, "team must be present in stats")
	assert.Equal(t, "StatsTeam", found.TeamName)
	assert.Equal(t, int64(2), found.MembersCount)
	assert.Equal(t, int64(2), found.DoneTasksLast7Days)
}

// Запрос (б): оконная функция ROW_NUMBER(), топ-3 автора задач в команде за месяц.
func Test_integration_analytics_top_creators(t *testing.T) {
	db := testdb.New(t)
	defer db.Close()

	u1 := createUser(t, db, "U1")
	u2 := createUser(t, db, "U2")
	u3 := createUser(t, db, "U3")
	teamID := createTeam(t, db, u1, "TopTeam")

	// u1 — 3 задачи, u2 — 2, u3 — 1.
	for i := 0; i < 3; i++ {
		insertTask(t, db, teamID, u1, string(entity.TaskStatusTodo), nil)
	}
	for i := 0; i < 2; i++ {
		insertTask(t, db, teamID, u2, string(entity.TaskStatusTodo), nil)
	}
	insertTask(t, db, teamID, u3, string(entity.TaskStatusTodo), nil)

	repo := repository.NewAnalyticsRepository(db)
	creators, err := repo.TopCreatorsPerTeam(t.Context())
	require.NoError(t, err)

	var teamCreators []repository.TopCreator
	for _, c := range creators {
		if c.TeamID == teamID {
			teamCreators = append(teamCreators, c)
		}
	}

	require.Len(t, teamCreators, 3)
	assert.Equal(t, u1, teamCreators[0].UserID)
	assert.Equal(t, int64(3), teamCreators[0].TasksCreated)
	assert.Equal(t, int64(1), teamCreators[0].Rank)
	assert.Equal(t, u2, teamCreators[1].UserID)
	assert.Equal(t, int64(2), teamCreators[1].TasksCreated)
	assert.Equal(t, u3, teamCreators[2].UserID)
	assert.Equal(t, int64(3), teamCreators[2].Rank)
}

// Запрос (в): условие по связанным таблицам — исполнитель вне команды задачи.
func Test_integration_analytics_inconsistent_tasks(t *testing.T) {
	db := testdb.New(t)
	defer db.Close()

	owner := createUser(t, db, "Owner")
	outsider := createUser(t, db, "Outsider")
	teamID := createTeam(t, db, owner, "IntegrityTeam")
	addMember(t, db, teamID, owner, string(entity.TeamRoleOwner))

	// Корректная задача: исполнитель — участник команды.
	insertTask(t, db, teamID, owner, string(entity.TaskStatusTodo), &owner)
	// Некорректная задача: исполнитель не состоит в команде.
	badTaskID := insertTask(t, db, teamID, owner, string(entity.TaskStatusTodo), &outsider)

	repo := repository.NewAnalyticsRepository(db)
	tasks, err := repo.InconsistentTasks(t.Context())
	require.NoError(t, err)

	var found *repository.InconsistentTask
	for i := range tasks {
		if tasks[i].TaskID == badTaskID {
			found = &tasks[i]
			break
		}
	}

	require.NotNil(t, found, "inconsistent task must be detected")
	assert.Equal(t, teamID, found.TeamID)
	assert.Equal(t, outsider, found.AssigneeID)
}
