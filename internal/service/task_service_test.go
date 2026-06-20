package service

import (
	"context"
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

func Test_create_task_successfully(t *testing.T) {
	created := &entity.Task{ID: 1, TeamID: 10, Title: "Task", Status: entity.TaskStatusTodo}

	tasks := mocks.NewMockTaskRepository(t)
	tasks.EXPECT().
		Create(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req repository.CreateTaskRequest) {
			assert.Equal(t, entity.TaskStatusTodo, req.Status)
			assert.Equal(t, "medium", req.Priority)
		}).
		Return(created, nil)

	history := mocks.NewMockTaskHistoryRepository(t)
	history.EXPECT().Add(mock.Anything, mock.Anything).Return(nil).Once()

	cache := mocks.NewMockTaskCache(t)
	cache.EXPECT().InvalidateTeam(mock.Anything, int64(10)).Return(nil)

	svc := NewTaskService(passthroughUoW(t), tasks, history, memberForAll(t), cache)

	task, err := svc.CreateTask(t.Context(), CreateTaskCommand{TeamID: 10, CreatorID: 1, Title: "Task"})

	require.NoError(t, err)
	assert.Equal(t, created, task)
}

func Test_create_task_with_assignee_member(t *testing.T) {
	tasks := mocks.NewMockTaskRepository(t)
	tasks.EXPECT().
		Create(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req repository.CreateTaskRequest) {
			assert.Equal(t, utils.Ptr(int64(2)), req.AssigneeID)
		}).
		Return(&entity.Task{ID: 1, TeamID: 10}, nil)

	history := mocks.NewMockTaskHistoryRepository(t)
	history.EXPECT().Add(mock.Anything, mock.Anything).Return(nil)

	cache := mocks.NewMockTaskCache(t)
	cache.EXPECT().InvalidateTeam(mock.Anything, mock.Anything).Return(nil)

	svc := NewTaskService(passthroughUoW(t), tasks, history, memberForAll(t), cache)

	_, err := svc.CreateTask(t.Context(), CreateTaskCommand{TeamID: 10, CreatorID: 1, Title: "Task", AssigneeID: 2})

	require.NoError(t, err)
}

func Test_create_task_forbidden_for_non_member(t *testing.T) {
	members := mocks.NewMockMembershipRepository(t)
	members.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil, infrastructure.ErrNotFound)

	svc := NewTaskService(passthroughUoW(t), mocks.NewMockTaskRepository(t), mocks.NewMockTaskHistoryRepository(t), members, mocks.NewMockTaskCache(t))

	task, err := svc.CreateTask(t.Context(), CreateTaskCommand{TeamID: 10, CreatorID: 1, Title: "Task"})

	assert.ErrorIs(t, err, ErrForbidden)
	assert.Nil(t, task)
}

func Test_create_task_assignee_not_member(t *testing.T) {
	members := mocks.NewMockMembershipRepository(t)
	members.EXPECT().Get(mock.Anything, int64(10), int64(1)).Return(member(10, 1, entity.TeamRoleMember), nil)
	members.EXPECT().Get(mock.Anything, int64(10), int64(99)).Return(nil, infrastructure.ErrNotFound)

	svc := NewTaskService(passthroughUoW(t), mocks.NewMockTaskRepository(t), mocks.NewMockTaskHistoryRepository(t), members, mocks.NewMockTaskCache(t))

	task, err := svc.CreateTask(t.Context(), CreateTaskCommand{TeamID: 10, CreatorID: 1, Title: "Task", AssigneeID: 99})

	assert.ErrorIs(t, err, ErrAssigneeNotMember)
	assert.Nil(t, task)
}

func Test_list_tasks_cache_hit(t *testing.T) {
	cached := []entity.Task{{ID: 1}, {ID: 2}}

	cache := mocks.NewMockTaskCache(t)
	cache.EXPECT().Get(mock.Anything, mock.Anything).Return(cached, true, nil)

	// tasks.List не должен вызываться при попадании в кеш — мок без ожидания List это гарантирует.
	svc := NewTaskService(passthroughUoW(t), mocks.NewMockTaskRepository(t), mocks.NewMockTaskHistoryRepository(t), memberForAll(t), cache)

	result, err := svc.ListTasks(t.Context(), ListTasksQuery{TeamID: 10, ActorID: 1, Limit: 20})

	require.NoError(t, err)
	assert.Equal(t, cached, result)
}

func Test_list_tasks_cache_miss_populates_cache(t *testing.T) {
	fromDB := []entity.Task{{ID: 3}}

	tasks := mocks.NewMockTaskRepository(t)
	tasks.EXPECT().
		List(mock.Anything, mock.Anything).
		Run(func(_ context.Context, filter repository.ListTasksFilter) {
			assert.Equal(t, int64(10), filter.TeamID)
			assert.Equal(t, "todo", filter.Status)
		}).
		Return(fromDB, nil)

	cache := mocks.NewMockTaskCache(t)
	cache.EXPECT().Get(mock.Anything, mock.Anything).Return(nil, false, nil)
	cache.EXPECT().Set(mock.Anything, mock.Anything, fromDB).Return(nil)

	svc := NewTaskService(passthroughUoW(t), tasks, mocks.NewMockTaskHistoryRepository(t), memberForAll(t), cache)

	result, err := svc.ListTasks(t.Context(), ListTasksQuery{TeamID: 10, ActorID: 1, Status: "todo", Limit: 20})

	require.NoError(t, err)
	assert.Equal(t, fromDB, result)
}

func Test_list_tasks_forbidden_for_non_member(t *testing.T) {
	members := mocks.NewMockMembershipRepository(t)
	members.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil, infrastructure.ErrNotFound)

	svc := NewTaskService(passthroughUoW(t), mocks.NewMockTaskRepository(t), mocks.NewMockTaskHistoryRepository(t), members, mocks.NewMockTaskCache(t))

	result, err := svc.ListTasks(t.Context(), ListTasksQuery{TeamID: 10, ActorID: 1, Limit: 20})

	assert.ErrorIs(t, err, ErrForbidden)
	assert.Nil(t, result)
}

func Test_update_task_records_history_for_changed_fields(t *testing.T) {
	stored := &entity.Task{ID: 1, TeamID: 10, Title: "old", Description: "desc", Status: entity.TaskStatusTodo, Priority: "medium"}
	updated := &entity.Task{ID: 1, TeamID: 10, Title: "new", Description: "desc", Status: entity.TaskStatusDone, Priority: "medium"}

	tasks := mocks.NewMockTaskRepository(t)
	tasks.EXPECT().GetByID(mock.Anything, int64(1)).Return(stored, nil).Once()
	tasks.EXPECT().GetByID(mock.Anything, int64(1)).Return(updated, nil).Once()

	var updateReq repository.UpdateTaskRequest
	tasks.EXPECT().
		Update(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req repository.UpdateTaskRequest) { updateReq = req }).
		Return(nil)

	var changedFields []string
	history := mocks.NewMockTaskHistoryRepository(t)
	history.EXPECT().
		Add(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req repository.AddTaskHistoryRequest) {
			changedFields = append(changedFields, req.Field)
		}).
		Return(nil).
		Times(2)

	cache := mocks.NewMockTaskCache(t)
	cache.EXPECT().InvalidateTeam(mock.Anything, mock.Anything).Return(nil)

	svc := NewTaskService(passthroughUoW(t), tasks, history, memberForAll(t), cache)

	newTitle := "new"
	newStatus := "done"
	result, err := svc.UpdateTask(t.Context(), UpdateTaskCommand{TaskID: 1, ActorID: 1, Title: &newTitle, Status: &newStatus})

	require.NoError(t, err)
	assert.Equal(t, updated, result)
	assert.Equal(t, "new", updateReq.Title)
	assert.Equal(t, entity.TaskStatusDone, updateReq.Status)
	assert.ElementsMatch(t, []string{entity.TaskFieldTitle, entity.TaskFieldStatus}, changedFields)
}

func Test_update_task_no_changes_skips_update(t *testing.T) {
	stored := &entity.Task{ID: 1, TeamID: 10, Title: "same", Status: entity.TaskStatusTodo}

	tasks := mocks.NewMockTaskRepository(t)
	tasks.EXPECT().GetByID(mock.Anything, int64(1)).Return(stored, nil)

	cache := mocks.NewMockTaskCache(t)
	cache.EXPECT().InvalidateTeam(mock.Anything, mock.Anything).Return(nil)

	// tasks.Update без ожидания: если он будет вызван — тест упадёт.
	svc := NewTaskService(passthroughUoW(t), tasks, mocks.NewMockTaskHistoryRepository(t), memberForAll(t), cache)

	sameTitle := "same"
	result, err := svc.UpdateTask(t.Context(), UpdateTaskCommand{TaskID: 1, ActorID: 1, Title: &sameTitle})

	require.NoError(t, err)
	assert.Equal(t, stored, result)
}

func Test_update_task_forbidden_for_non_member(t *testing.T) {
	stored := &entity.Task{ID: 1, TeamID: 10}

	tasks := mocks.NewMockTaskRepository(t)
	tasks.EXPECT().GetByID(mock.Anything, int64(1)).Return(stored, nil)

	members := mocks.NewMockMembershipRepository(t)
	members.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil, infrastructure.ErrNotFound)

	svc := NewTaskService(passthroughUoW(t), tasks, mocks.NewMockTaskHistoryRepository(t), members, mocks.NewMockTaskCache(t))

	newTitle := "new"
	result, err := svc.UpdateTask(t.Context(), UpdateTaskCommand{TaskID: 1, ActorID: 2, Title: &newTitle})

	assert.ErrorIs(t, err, ErrForbidden)
	assert.Nil(t, result)
}

func Test_update_task_invalid_status(t *testing.T) {
	stored := &entity.Task{ID: 1, TeamID: 10, Status: entity.TaskStatusTodo}

	tasks := mocks.NewMockTaskRepository(t)
	tasks.EXPECT().GetByID(mock.Anything, int64(1)).Return(stored, nil)

	svc := NewTaskService(passthroughUoW(t), tasks, mocks.NewMockTaskHistoryRepository(t), memberForAll(t), mocks.NewMockTaskCache(t))

	badStatus := "flying"
	result, err := svc.UpdateTask(t.Context(), UpdateTaskCommand{TaskID: 1, ActorID: 1, Status: &badStatus})

	assert.ErrorIs(t, err, ErrInvalidStatus)
	assert.Nil(t, result)
}

func Test_update_task_assignee_not_member(t *testing.T) {
	stored := &entity.Task{ID: 1, TeamID: 10}

	tasks := mocks.NewMockTaskRepository(t)
	tasks.EXPECT().GetByID(mock.Anything, int64(1)).Return(stored, nil)

	members := mocks.NewMockMembershipRepository(t)
	members.EXPECT().Get(mock.Anything, int64(10), int64(1)).Return(member(10, 1, entity.TeamRoleMember), nil)
	members.EXPECT().Get(mock.Anything, int64(10), int64(99)).Return(nil, infrastructure.ErrNotFound)

	svc := NewTaskService(passthroughUoW(t), tasks, mocks.NewMockTaskHistoryRepository(t), members, mocks.NewMockTaskCache(t))

	assignee := int64(99)
	result, err := svc.UpdateTask(t.Context(), UpdateTaskCommand{TaskID: 1, ActorID: 1, AssigneeID: &assignee})

	assert.ErrorIs(t, err, ErrAssigneeNotMember)
	assert.Nil(t, result)
}

func Test_update_task_not_found(t *testing.T) {
	tasks := mocks.NewMockTaskRepository(t)
	tasks.EXPECT().GetByID(mock.Anything, int64(1)).Return(nil, infrastructure.ErrNotFound)

	svc := NewTaskService(passthroughUoW(t), tasks, mocks.NewMockTaskHistoryRepository(t), memberForAll(t), mocks.NewMockTaskCache(t))

	newTitle := "new"
	result, err := svc.UpdateTask(t.Context(), UpdateTaskCommand{TaskID: 1, ActorID: 1, Title: &newTitle})

	assert.ErrorIs(t, err, infrastructure.ErrNotFound)
	assert.Nil(t, result)
}

func Test_get_history_successfully(t *testing.T) {
	stored := &entity.Task{ID: 1, TeamID: 10}
	expected := []entity.TaskHistory{{ID: 1, Field: entity.TaskFieldStatus}}

	tasks := mocks.NewMockTaskRepository(t)
	tasks.EXPECT().GetByID(mock.Anything, int64(1)).Return(stored, nil)

	history := mocks.NewMockTaskHistoryRepository(t)
	history.EXPECT().ListByTask(mock.Anything, int64(1)).Return(expected, nil)

	svc := NewTaskService(passthroughUoW(t), tasks, history, memberForAll(t), mocks.NewMockTaskCache(t))

	result, err := svc.GetHistory(t.Context(), 1, 1)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func Test_get_history_forbidden_for_non_member(t *testing.T) {
	stored := &entity.Task{ID: 1, TeamID: 10}

	tasks := mocks.NewMockTaskRepository(t)
	tasks.EXPECT().GetByID(mock.Anything, int64(1)).Return(stored, nil)

	members := mocks.NewMockMembershipRepository(t)
	members.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil, infrastructure.ErrNotFound)

	svc := NewTaskService(passthroughUoW(t), tasks, mocks.NewMockTaskHistoryRepository(t), members, mocks.NewMockTaskCache(t))

	result, err := svc.GetHistory(t.Context(), 2, 1)

	assert.ErrorIs(t, err, ErrForbidden)
	assert.Nil(t, result)
}
