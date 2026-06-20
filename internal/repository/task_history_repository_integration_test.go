//go:build integration

package repository_test

import (
	"testing"

	"taskmanager/internal/entity"
	"taskmanager/internal/infrastructure/testdb"
	"taskmanager/internal/repository"
	"taskmanager/internal/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func Test_integration_task_history_add_and_list(t *testing.T) {
	db := testdb.New(t)
	defer db.Close()

	userID := createUser(t, db, "User")
	teamID := createTeam(t, db, userID, "Team")
	taskID := insertTask(t, db, teamID, userID, string(entity.TaskStatusTodo), nil)

	repo := repository.NewTaskHistoryRepository(db)

	require.NoError(t, repo.Add(t.Context(), repository.AddTaskHistoryRequest{
		TaskID:    taskID,
		ChangedBy: userID,
		Field:     entity.TaskFieldStatus,
		OldValue:  utils.Ptr("todo"),
		NewValue:  utils.Ptr("done"),
	}))
	require.NoError(t, repo.Add(t.Context(), repository.AddTaskHistoryRequest{
		TaskID:    taskID,
		ChangedBy: userID,
		Field:     entity.TaskFieldTitle,
		OldValue:  nil,
		NewValue:  utils.Ptr("new title"),
	}))

	history, err := repo.ListByTask(t.Context(), taskID)
	require.NoError(t, err)
	require.Len(t, history, 2)

	assert.Equal(t, entity.TaskFieldStatus, history[0].Field)
	require.NotNil(t, history[0].NewValue)
	assert.Equal(t, "done", *history[0].NewValue)
	assert.Equal(t, entity.TaskFieldTitle, history[1].Field)
	assert.Nil(t, history[1].OldValue)
}
