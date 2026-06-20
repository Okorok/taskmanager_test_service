package service

import (
	"context"
	"errors"
	"testing"

	"taskmanager/internal/entity"
	"taskmanager/internal/infrastructure"
	"taskmanager/internal/repository"
	"taskmanager/internal/service/mocks"
	"taskmanager/internal/utils"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_create_task_with_explicit_priority_and_create_error(t *testing.T) {
	tasks := mocks.NewMockTaskRepository(t)
	tasks.EXPECT().
		Create(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req repository.CreateTaskRequest) {
			assert.Equal(t, "high", req.Priority)
		}).
		Return(nil, errors.New("db error"))

	svc := NewTaskService(passthroughUoW(t), tasks, mocks.NewMockTaskHistoryRepository(t), memberForAll(t), mocks.NewMockTaskCache(t))

	task, err := svc.CreateTask(t.Context(), CreateTaskCommand{TeamID: 10, CreatorID: 1, Title: "Task", Priority: "high"})

	assert.Error(t, err)
	assert.Nil(t, task)
}

func Test_list_tasks_membership_check_unexpected_error(t *testing.T) {
	members := mocks.NewMockMembershipRepository(t)
	members.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil, errors.New("db error"))

	svc := NewTaskService(passthroughUoW(t), mocks.NewMockTaskRepository(t), mocks.NewMockTaskHistoryRepository(t), members, mocks.NewMockTaskCache(t))

	result, err := svc.ListTasks(t.Context(), ListTasksQuery{TeamID: 10, ActorID: 1, Limit: 20})

	assert.Error(t, err)
	assert.NotErrorIs(t, err, ErrForbidden)
	assert.Nil(t, result)
}

func Test_update_task_changes_assignee_priority_description(t *testing.T) {
	stored := &entity.Task{
		ID: 1, TeamID: 10, Title: "t", Description: "d1", Status: entity.TaskStatusTodo, Priority: "low",
	}

	tasks := mocks.NewMockTaskRepository(t)
	tasks.EXPECT().GetByID(mock.Anything, int64(1)).Return(stored, nil).Times(2)

	var updateReq repository.UpdateTaskRequest
	tasks.EXPECT().
		Update(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req repository.UpdateTaskRequest) { updateReq = req }).
		Return(nil)

	changes := make(map[string][2]*string)
	history := mocks.NewMockTaskHistoryRepository(t)
	history.EXPECT().
		Add(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req repository.AddTaskHistoryRequest) {
			changes[req.Field] = [2]*string{req.OldValue, req.NewValue}
		}).
		Return(nil).
		Times(3)

	cache := mocks.NewMockTaskCache(t)
	cache.EXPECT().InvalidateTeam(mock.Anything, mock.Anything).Return(nil)

	svc := NewTaskService(passthroughUoW(t), tasks, history, memberForAll(t), cache)

	newDesc := "d2"
	newPriority := "high"
	newAssignee := int64(2)
	_, err := svc.UpdateTask(t.Context(), UpdateTaskCommand{
		TaskID:      1,
		ActorID:     1,
		Description: &newDesc,
		Priority:    &newPriority,
		AssigneeID:  &newAssignee,
	})

	require.NoError(t, err)
	assert.Equal(t, utils.Ptr(int64(2)), updateReq.AssigneeID)

	assigneeChange := changes[entity.TaskFieldAssignee]
	assert.Nil(t, assigneeChange[0])
	assert.Equal(t, utils.Ptr("2"), assigneeChange[1])

	priorityChange := changes[entity.TaskFieldPriority]
	assert.Equal(t, utils.Ptr("low"), priorityChange[0])
	assert.Equal(t, utils.Ptr("high"), priorityChange[1])
}

func Test_update_task_unassign(t *testing.T) {
	stored := &entity.Task{
		ID: 1, TeamID: 10, Status: entity.TaskStatusTodo,
		AssigneeID: utils.Ptr(int64(2)),
	}

	tasks := mocks.NewMockTaskRepository(t)
	tasks.EXPECT().GetByID(mock.Anything, int64(1)).Return(stored, nil).Times(2)

	var updateReq repository.UpdateTaskRequest
	tasks.EXPECT().
		Update(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req repository.UpdateTaskRequest) { updateReq = req }).
		Return(nil)

	var assigneeChange repository.AddTaskHistoryRequest
	history := mocks.NewMockTaskHistoryRepository(t)
	history.EXPECT().
		Add(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req repository.AddTaskHistoryRequest) {
			if req.Field == entity.TaskFieldAssignee {
				assigneeChange = req
			}
		}).
		Return(nil)

	cache := mocks.NewMockTaskCache(t)
	cache.EXPECT().InvalidateTeam(mock.Anything, mock.Anything).Return(nil)

	svc := NewTaskService(passthroughUoW(t), tasks, history, memberForAll(t), cache)

	unassign := int64(0)
	_, err := svc.UpdateTask(t.Context(), UpdateTaskCommand{TaskID: 1, ActorID: 1, AssigneeID: &unassign})

	require.NoError(t, err)
	assert.Nil(t, updateReq.AssigneeID, "assignee must be cleared")
	assert.Equal(t, utils.Ptr("2"), assigneeChange.OldValue)
	assert.Nil(t, assigneeChange.NewValue, "new assignee value is NULL")
}

func Test_update_task_update_repo_error(t *testing.T) {
	stored := &entity.Task{ID: 1, TeamID: 10, Title: "old", Status: entity.TaskStatusTodo}

	tasks := mocks.NewMockTaskRepository(t)
	tasks.EXPECT().GetByID(mock.Anything, int64(1)).Return(stored, nil)
	tasks.EXPECT().Update(mock.Anything, mock.Anything).Return(errors.New("db error"))

	svc := NewTaskService(passthroughUoW(t), tasks, mocks.NewMockTaskHistoryRepository(t), memberForAll(t), mocks.NewMockTaskCache(t))

	newTitle := "new"
	result, err := svc.UpdateTask(t.Context(), UpdateTaskCommand{TaskID: 1, ActorID: 1, Title: &newTitle})

	assert.Error(t, err)
	assert.Nil(t, result)
}

func Test_get_history_task_not_found(t *testing.T) {
	tasks := mocks.NewMockTaskRepository(t)
	tasks.EXPECT().GetByID(mock.Anything, int64(1)).Return(nil, infrastructure.ErrNotFound)

	svc := NewTaskService(passthroughUoW(t), tasks, mocks.NewMockTaskHistoryRepository(t), memberForAll(t), mocks.NewMockTaskCache(t))

	result, err := svc.GetHistory(t.Context(), 1, 1)

	assert.ErrorIs(t, err, infrastructure.ErrNotFound)
	assert.Nil(t, result)
}
