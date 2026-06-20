//go:build integration

package service_test

import (
	"testing"

	"taskmanager/internal/entity"
	"taskmanager/internal/infrastructure"
	"taskmanager/internal/infrastructure/testdb"
	"taskmanager/internal/repository"
	"taskmanager/internal/service"
	"taskmanager/internal/service/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// noopCache — мок кеша с опциональными вызовами: интеграционно проверяем работу
// с реальной MySQL без поднятия Redis.
func noopCache(t *testing.T) *mocks.MockTaskCache {
	c := mocks.NewMockTaskCache(t)
	c.EXPECT().Get(mock.Anything, mock.Anything).Return(nil, false, nil).Maybe()
	c.EXPECT().Set(mock.Anything, mock.Anything, mock.Anything).Return(nil).Maybe()
	c.EXPECT().InvalidateTeam(mock.Anything, mock.Anything).Return(nil).Maybe()

	return c
}

// Сквозной сценарий через сервисный слой на реальной MySQL:
// создание команды (owner добавляется в той же транзакции) -> создание задачи
// (+ запись в историю) -> обновление статуса (+ запись в историю) -> чтение истории.
func Test_integration_task_lifecycle_through_services(t *testing.T) {
	db := testdb.New(t)
	defer db.Close()

	uow := infrastructure.NewUnitOfWork(db)
	userRepo := repository.NewUserRepository(db)
	teamRepo := repository.NewTeamRepository(db)
	memberRepo := repository.NewTeamMemberRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	historyRepo := repository.NewTaskHistoryRepository(db)

	user, err := userRepo.Create(t.Context(), repository.CreateUserRequest{
		Email:        "lifecycle@test.local",
		PasswordHash: "hash",
		Name:         "User",
	})
	require.NoError(t, err)

	teamService := service.NewTeamService(uow, teamRepo, memberRepo, userRepo)
	team, err := teamService.CreateTeam(t.Context(), user.ID, "Lifecycle")
	require.NoError(t, err)

	// Создатель команды автоматически стал её участником (owner) в той же транзакции.
	owner, err := memberRepo.Get(t.Context(), team.ID, user.ID)
	require.NoError(t, err)
	assert.Equal(t, entity.TeamRoleOwner, owner.Role)

	taskService := service.NewTaskService(uow, taskRepo, historyRepo, memberRepo, noopCache(t))

	task, err := taskService.CreateTask(t.Context(), service.CreateTaskCommand{
		TeamID:    team.ID,
		CreatorID: user.ID,
		Title:     "Important",
	})
	require.NoError(t, err)
	assert.Equal(t, entity.TaskStatusTodo, task.Status)

	newStatus := string(entity.TaskStatusDone)
	updated, err := taskService.UpdateTask(t.Context(), service.UpdateTaskCommand{
		TaskID:  task.ID,
		ActorID: user.ID,
		Status:  &newStatus,
	})
	require.NoError(t, err)
	assert.Equal(t, entity.TaskStatusDone, updated.Status)

	history, err := taskService.GetHistory(t.Context(), user.ID, task.ID)
	require.NoError(t, err)

	// Ожидаем минимум две записи: создание задачи и смена статуса.
	require.GreaterOrEqual(t, len(history), 2)

	fields := make([]string, 0, len(history))
	for _, h := range history {
		fields = append(fields, h.Field)
	}
	assert.Contains(t, fields, entity.TaskFieldCreated)
	assert.Contains(t, fields, entity.TaskFieldStatus)
}

// Проверяем запрет создания задачи пользователем, не состоящим в команде.
func Test_integration_create_task_forbidden_for_non_member(t *testing.T) {
	db := testdb.New(t)
	defer db.Close()

	uow := infrastructure.NewUnitOfWork(db)
	userRepo := repository.NewUserRepository(db)
	teamRepo := repository.NewTeamRepository(db)
	memberRepo := repository.NewTeamMemberRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	historyRepo := repository.NewTaskHistoryRepository(db)

	owner, err := userRepo.Create(t.Context(), repository.CreateUserRequest{
		Email: "owner-forbidden@test.local", PasswordHash: "hash", Name: "Owner",
	})
	require.NoError(t, err)
	outsider, err := userRepo.Create(t.Context(), repository.CreateUserRequest{
		Email: "outsider-forbidden@test.local", PasswordHash: "hash", Name: "Outsider",
	})
	require.NoError(t, err)

	teamService := service.NewTeamService(uow, teamRepo, memberRepo, userRepo)
	team, err := teamService.CreateTeam(t.Context(), owner.ID, "Closed")
	require.NoError(t, err)

	taskService := service.NewTaskService(uow, taskRepo, historyRepo, memberRepo, noopCache(t))

	_, err = taskService.CreateTask(t.Context(), service.CreateTaskCommand{
		TeamID:    team.ID,
		CreatorID: outsider.ID,
		Title:     "Nope",
	})

	assert.ErrorIs(t, err, service.ErrForbidden)
}
