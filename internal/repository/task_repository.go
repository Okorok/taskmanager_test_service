package repository

import (
	"context"
	"database/sql"
	"strings"

	"taskmanager/internal/entity"
	"taskmanager/internal/infrastructure"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type TaskRepository struct {
	db *sqlx.DB
}

func NewTaskRepository(db *sqlx.DB) *TaskRepository {
	return &TaskRepository{db: db}
}

const taskColumns = `id, team_id, title, description, status, priority, assignee_id, created_by, created_at, updated_at`

type CreateTaskRequest struct {
	TeamID      int64
	Title       string
	Description string
	Status      entity.TaskStatus
	Priority    string
	AssigneeID  *int64
	CreatedBy   int64
}

const queryInsertTask = `
	INSERT INTO tasks (team_id, title, description, status, priority, assignee_id, created_by)
	VALUES (?, ?, ?, ?, ?, ?, ?)
`

func (r *TaskRepository) Create(ctx context.Context, request CreateTaskRequest) (*entity.Task, error) {
	res, err := queryExecutor(ctx, r.db).ExecContext(ctx, queryInsertTask,
		request.TeamID,
		request.Title,
		request.Description,
		request.Status,
		request.Priority,
		request.AssigneeID,
		request.CreatedBy,
	)
	if err != nil {
		return nil, errors.WithStack(errors.Wrap(err, "failed to insert task"))
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, errors.WithStack(errors.Wrap(err, "failed to get inserted task id"))
	}

	return r.GetByID(ctx, id)
}

const queryGetTaskByID = `
	SELECT ` + taskColumns + `
	FROM tasks
	WHERE id = ?
`

func (r *TaskRepository) GetByID(ctx context.Context, id int64) (*entity.Task, error) {
	var task entity.Task
	if err := sqlx.GetContext(ctx, queryExecutor(ctx, r.db), &task, queryGetTaskByID, id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, infrastructure.ErrNotFound
		}

		return nil, errors.WithStack(errors.Wrap(err, "failed to get task by id"))
	}

	return &task, nil
}

type ListTasksFilter struct {
	TeamID     int64
	Status     string
	AssigneeID int64
	Limit      int
	Offset     int
}

func (r *TaskRepository) List(ctx context.Context, filter ListTasksFilter) ([]entity.Task, error) {
	conditions := []string{"team_id = ?"}
	args := []any{filter.TeamID}

	if filter.Status != "" {
		conditions = append(conditions, "status = ?")
		args = append(args, filter.Status)
	}

	if filter.AssigneeID > 0 {
		conditions = append(conditions, "assignee_id = ?")
		args = append(args, filter.AssigneeID)
	}

	query := `SELECT ` + taskColumns + ` FROM tasks WHERE ` +
		strings.Join(conditions, " AND ") +
		` ORDER BY id DESC LIMIT ? OFFSET ?`
	args = append(args, filter.Limit, filter.Offset)

	var tasks []entity.Task
	if err := sqlx.SelectContext(ctx, queryExecutor(ctx, r.db), &tasks, query, args...); err != nil {
		return nil, errors.WithStack(errors.Wrap(err, "failed to list tasks"))
	}

	return tasks, nil
}

type UpdateTaskRequest struct {
	ID          int64
	Title       string
	Description string
	Status      entity.TaskStatus
	Priority    string
	AssigneeID  *int64
}

const queryUpdateTask = `
	UPDATE tasks
	SET title = ?, description = ?, status = ?, priority = ?, assignee_id = ?
	WHERE id = ?
`

func (r *TaskRepository) Update(ctx context.Context, request UpdateTaskRequest) error {
	res, err := queryExecutor(ctx, r.db).ExecContext(ctx, queryUpdateTask,
		request.Title,
		request.Description,
		request.Status,
		request.Priority,
		request.AssigneeID,
		request.ID,
	)
	if err != nil {
		return errors.WithStack(errors.Wrap(err, "failed to update task"))
	}

	rows, err := res.RowsAffected()
	if err != nil {
		return errors.WithStack(errors.Wrap(err, "failed to read affected rows"))
	}

	if rows == 0 {
		return infrastructure.ErrNotFound
	}

	return nil
}
