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

func Test_integration_create_and_get_task(t *testing.T) {
	db := testdb.New(t)
	defer db.Close()

	userID := createUser(t, db, "Creator")
	teamID := createTeam(t, db, userID, "Team")

	repo := repository.NewTaskRepository(db)
	created, err := repo.Create(t.Context(), repository.CreateTaskRequest{
		TeamID:      teamID,
		Title:       "Task",
		Description: "Description",
		Status:      entity.TaskStatusTodo,
		Priority:    "high",
		AssigneeID:  &userID,
		CreatedBy:   userID,
	})
	require.NoError(t, err)
	assert.NotZero(t, created.ID)

	loaded, err := repo.GetByID(t.Context(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, created, loaded)
	assert.Equal(t, entity.TaskStatusTodo, loaded.Status)
	require.NotNil(t, loaded.AssigneeID)
	assert.Equal(t, userID, *loaded.AssigneeID)
}

func Test_integration_get_task_not_found(t *testing.T) {
	db := testdb.New(t)
	defer db.Close()

	repo := repository.NewTaskRepository(db)
	_, err := repo.GetByID(t.Context(), 999999)

	assert.ErrorIs(t, err, infrastructure.ErrNotFound)
}

func Test_integration_list_tasks_with_filters_and_pagination(t *testing.T) {
	db := testdb.New(t)
	defer db.Close()

	userID := createUser(t, db, "Creator")
	assigneeID := createUser(t, db, "Assignee")
	teamID := createTeam(t, db, userID, "Team")

	assignee := &assigneeID
	// 3 todo + 1 done, часть с исполнителем.
	insertTask(t, db, teamID, userID, string(entity.TaskStatusTodo), assignee)
	insertTask(t, db, teamID, userID, string(entity.TaskStatusTodo), assignee)
	insertTask(t, db, teamID, userID, string(entity.TaskStatusTodo), nil)
	insertTask(t, db, teamID, userID, string(entity.TaskStatusDone), assignee)

	repo := repository.NewTaskRepository(db)

	// Фильтр по статусу.
	todo, err := repo.List(t.Context(), repository.ListTasksFilter{
		TeamID: teamID, Status: string(entity.TaskStatusTodo), Limit: 10,
	})
	require.NoError(t, err)
	assert.Len(t, todo, 3)

	// Фильтр по статусу + исполнителю.
	todoAssigned, err := repo.List(t.Context(), repository.ListTasksFilter{
		TeamID: teamID, Status: string(entity.TaskStatusTodo), AssigneeID: assigneeID, Limit: 10,
	})
	require.NoError(t, err)
	assert.Len(t, todoAssigned, 2)

	// Пагинация: всего 4 задачи, по 2 на страницу.
	page1, err := repo.List(t.Context(), repository.ListTasksFilter{TeamID: teamID, Limit: 2, Offset: 0})
	require.NoError(t, err)
	assert.Len(t, page1, 2)

	page2, err := repo.List(t.Context(), repository.ListTasksFilter{TeamID: teamID, Limit: 2, Offset: 2})
	require.NoError(t, err)
	assert.Len(t, page2, 2)

	// Страницы не пересекаются.
	assert.NotEqual(t, page1[0].ID, page2[0].ID)
}

func Test_integration_update_task(t *testing.T) {
	db := testdb.New(t)
	defer db.Close()

	userID := createUser(t, db, "Creator")
	teamID := createTeam(t, db, userID, "Team")
	taskID := insertTask(t, db, teamID, userID, string(entity.TaskStatusTodo), nil)

	repo := repository.NewTaskRepository(db)
	err := repo.Update(t.Context(), repository.UpdateTaskRequest{
		ID:          taskID,
		Title:       "Updated",
		Description: "Updated description",
		Status:      entity.TaskStatusDone,
		Priority:    "low",
		AssigneeID:  &userID,
	})
	require.NoError(t, err)

	loaded, err := repo.GetByID(t.Context(), taskID)
	require.NoError(t, err)
	assert.Equal(t, "Updated", loaded.Title)
	assert.Equal(t, entity.TaskStatusDone, loaded.Status)
	assert.Equal(t, "low", loaded.Priority)
	require.NotNil(t, loaded.AssigneeID)
	assert.Equal(t, userID, *loaded.AssigneeID)
}

func Test_integration_update_task_not_found(t *testing.T) {
	db := testdb.New(t)
	defer db.Close()

	repo := repository.NewTaskRepository(db)
	err := repo.Update(t.Context(), repository.UpdateTaskRequest{
		ID:     999999,
		Title:  "X",
		Status: entity.TaskStatusTodo,
	})

	assert.ErrorIs(t, err, infrastructure.ErrNotFound)
}
