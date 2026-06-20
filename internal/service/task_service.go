package service

import (
	"context"
	"strconv"

	"taskmanager/internal/entity"
	"taskmanager/internal/infrastructure/cache"
	"taskmanager/internal/repository"
	"taskmanager/internal/utils"

	"github.com/pkg/errors"
)

var (
	ErrAssigneeNotMember = errors.New("assignee is not a team member")
	ErrInvalidStatus     = errors.New("invalid task status")
)

type TaskRepository interface {
	Create(ctx context.Context, request repository.CreateTaskRequest) (*entity.Task, error)
	GetByID(ctx context.Context, id int64) (*entity.Task, error)
	List(ctx context.Context, filter repository.ListTasksFilter) ([]entity.Task, error)
	Update(ctx context.Context, request repository.UpdateTaskRequest) error
}

type TaskHistoryRepository interface {
	Add(ctx context.Context, request repository.AddTaskHistoryRequest) error
	ListByTask(ctx context.Context, taskID int64) ([]entity.TaskHistory, error)
}

type TaskCache interface {
	Get(ctx context.Context, key string) ([]entity.Task, bool, error)
	Set(ctx context.Context, key string, tasks []entity.Task) error
	InvalidateTeam(ctx context.Context, teamID int64) error
}

type TaskService struct {
	uow     UnitOfWork
	tasks   TaskRepository
	history TaskHistoryRepository
	members MembershipRepository
	cache   TaskCache
}

func NewTaskService(
	uow UnitOfWork,
	tasks TaskRepository,
	history TaskHistoryRepository,
	members MembershipRepository,
	taskCache TaskCache,
) *TaskService {
	return &TaskService{
		uow:     uow,
		tasks:   tasks,
		history: history,
		members: members,
		cache:   taskCache,
	}
}

type CreateTaskCommand struct {
	TeamID      int64
	CreatorID   int64
	Title       string
	Description string
	Priority    string
	AssigneeID  int64
}

func (s *TaskService) CreateTask(ctx context.Context, cmd CreateTaskCommand) (*entity.Task, error) {
	if err := s.ensureMember(ctx, cmd.TeamID, cmd.CreatorID); err != nil {
		return nil, err
	}

	assignee, err := s.resolveAssignee(ctx, cmd.TeamID, cmd.AssigneeID)
	if err != nil {
		return nil, err
	}

	priority := cmd.Priority
	if priority == "" {
		priority = "medium"
	}

	var created *entity.Task
	err = s.uow.Do(ctx, func(ctx context.Context) error {
		task, err := s.tasks.Create(ctx, repository.CreateTaskRequest{
			TeamID:      cmd.TeamID,
			Title:       cmd.Title,
			Description: cmd.Description,
			Status:      entity.TaskStatusTodo,
			Priority:    priority,
			AssigneeID:  assignee,
			CreatedBy:   cmd.CreatorID,
		})
		if err != nil {
			return err
		}

		if err := s.history.Add(ctx, repository.AddTaskHistoryRequest{
			TaskID:    task.ID,
			ChangedBy: cmd.CreatorID,
			Field:     entity.TaskFieldCreated,
			OldValue:  nil,
			NewValue:  utils.Ptr(string(task.Status)),
		}); err != nil {
			return err
		}

		created = task
		return nil
	})
	if err != nil {
		return nil, err
	}

	_ = s.cache.InvalidateTeam(ctx, cmd.TeamID)

	return created, nil
}

type ListTasksQuery struct {
	TeamID     int64
	ActorID    int64
	Status     string
	AssigneeID int64
	Limit      int
	Offset     int
}

func (s *TaskService) ListTasks(ctx context.Context, query ListTasksQuery) ([]entity.Task, error) {
	member, err := s.isMember(ctx, query.TeamID, query.ActorID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, ErrForbidden
	}

	key := cache.TeamTasksKey(query.TeamID, query.Status, query.AssigneeID, query.Limit, query.Offset)

	if cached, ok, err := s.cache.Get(ctx, key); err == nil && ok {
		return cached, nil
	}

	tasks, err := s.tasks.List(ctx, repository.ListTasksFilter{
		TeamID:     query.TeamID,
		Status:     query.Status,
		AssigneeID: query.AssigneeID,
		Limit:      query.Limit,
		Offset:     query.Offset,
	})
	if err != nil {
		return nil, err
	}

	_ = s.cache.Set(ctx, key, tasks)

	return tasks, nil
}

type UpdateTaskCommand struct {
	TaskID      int64
	ActorID     int64
	Title       *string
	Description *string
	Status      *string
	Priority    *string
	AssigneeID  *int64
}

func (s *TaskService) UpdateTask(ctx context.Context, cmd UpdateTaskCommand) (*entity.Task, error) {
	var updated *entity.Task
	var teamID int64

	err := s.uow.Do(ctx, func(ctx context.Context) error {
		task, err := s.tasks.GetByID(ctx, cmd.TaskID)
		if err != nil {
			return err
		}
		teamID = task.TeamID

		if err := s.ensureMember(ctx, task.TeamID, cmd.ActorID); err != nil {
			return err
		}

		patched, changes, err := s.buildPatch(ctx, task, cmd)
		if err != nil {
			return err
		}

		if len(changes) == 0 {
			updated = task
			return nil
		}

		if err := s.tasks.Update(ctx, repository.UpdateTaskRequest{
			ID:          patched.ID,
			Title:       patched.Title,
			Description: patched.Description,
			Status:      patched.Status,
			Priority:    patched.Priority,
			AssigneeID:  patched.AssigneeID,
		}); err != nil {
			return err
		}

		for _, change := range changes {
			if err := s.history.Add(ctx, change); err != nil {
				return err
			}
		}

		updated, err = s.tasks.GetByID(ctx, patched.ID)
		return err
	})
	if err != nil {
		return nil, err
	}

	_ = s.cache.InvalidateTeam(ctx, teamID)

	return updated, nil
}

func (s *TaskService) buildPatch(
	ctx context.Context,
	task *entity.Task,
	cmd UpdateTaskCommand,
) (entity.Task, []repository.AddTaskHistoryRequest, error) {
	patch := taskPatch{taskID: task.ID, actorID: cmd.ActorID, state: *task}

	patch.setString(entity.TaskFieldTitle, &patch.state.Title, cmd.Title)
	patch.setString(entity.TaskFieldDescription, &patch.state.Description, cmd.Description)
	patch.setString(entity.TaskFieldPriority, &patch.state.Priority, cmd.Priority)

	if cmd.Status != nil {
		status := entity.TaskStatus(*cmd.Status)
		if !status.IsValid() {
			return entity.Task{}, nil, ErrInvalidStatus
		}
		patch.setStatus(status)
	}

	if cmd.AssigneeID != nil {
		assignee, err := s.resolveAssignee(ctx, task.TeamID, *cmd.AssigneeID)
		if err != nil {
			return entity.Task{}, nil, err
		}
		patch.setAssignee(assignee)
	}

	return patch.state, patch.entries, nil
}

type taskPatch struct {
	taskID, actorID int64
	state           entity.Task
	entries         []repository.AddTaskHistoryRequest
}

func (p *taskPatch) record(field string, old, new *string) {
	p.entries = append(p.entries, historyEntry(p.taskID, p.actorID, field, old, new))
}

func (p *taskPatch) setString(field string, current, incoming *string) {
	if incoming == nil || *incoming == *current {
		return
	}
	p.record(field, utils.Ptr(*current), utils.Ptr(*incoming))
	*current = *incoming
}

func (p *taskPatch) setStatus(status entity.TaskStatus) {
	if status == p.state.Status {
		return
	}
	p.record(entity.TaskFieldStatus, utils.Ptr(string(p.state.Status)), utils.Ptr(string(status)))
	p.state.Status = status
}

func (p *taskPatch) setAssignee(assignee *int64) {
	if utils.EqualPtr(assignee, p.state.AssigneeID) {
		return
	}
	toStr := func(v int64) string { return strconv.FormatInt(v, 10) }
	p.record(entity.TaskFieldAssignee, utils.MapPtr(p.state.AssigneeID, toStr), utils.MapPtr(assignee, toStr))
	p.state.AssigneeID = assignee
}

func (s *TaskService) GetHistory(ctx context.Context, actorID, taskID int64) ([]entity.TaskHistory, error) {
	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return nil, err
	}

	member, err := s.isMember(ctx, task.TeamID, actorID)
	if err != nil {
		return nil, err
	}
	if !member {
		return nil, ErrForbidden
	}

	return s.history.ListByTask(ctx, taskID)
}

func (s *TaskService) isMember(ctx context.Context, teamID, userID int64) (bool, error) {
	return isTeamMember(ctx, s.members, teamID, userID)
}

func (s *TaskService) ensureMember(ctx context.Context, teamID, userID int64) error {
	member, err := s.isMember(ctx, teamID, userID)
	if err != nil {
		return err
	}

	if !member {
		return ErrForbidden
	}

	return nil
}

func (s *TaskService) resolveAssignee(ctx context.Context, teamID, assigneeID int64) (*int64, error) {
	if assigneeID <= 0 {
		return nil, nil
	}

	member, err := s.isMember(ctx, teamID, assigneeID)
	if err != nil {
		return nil, err
	}

	if !member {
		return nil, ErrAssigneeNotMember
	}

	return &assigneeID, nil
}

func historyEntry(taskID, changedBy int64, field string, oldValue, newValue *string) repository.AddTaskHistoryRequest {
	return repository.AddTaskHistoryRequest{
		TaskID:    taskID,
		ChangedBy: changedBy,
		Field:     field,
		OldValue:  oldValue,
		NewValue:  newValue,
	}
}
