package repository

import (
	"context"

	"taskmanager/internal/entity"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type TaskHistoryRepository struct {
	db *sqlx.DB
}

func NewTaskHistoryRepository(db *sqlx.DB) *TaskHistoryRepository {
	return &TaskHistoryRepository{db: db}
}

type AddTaskHistoryRequest struct {
	TaskID    int64
	ChangedBy int64
	Field     string
	OldValue  *string
	NewValue  *string
}

const queryInsertTaskHistory = `
	INSERT INTO task_history (task_id, changed_by, field, old_value, new_value)
	VALUES (?, ?, ?, ?, ?)
`

func (r *TaskHistoryRepository) Add(ctx context.Context, request AddTaskHistoryRequest) error {
	_, err := queryExecutor(ctx, r.db).ExecContext(ctx, queryInsertTaskHistory,
		request.TaskID,
		request.ChangedBy,
		request.Field,
		request.OldValue,
		request.NewValue,
	)
	if err != nil {
		return errors.WithStack(errors.Wrap(err, "failed to insert task history"))
	}

	return nil
}

const queryListTaskHistory = `
	SELECT id, task_id, changed_by, field, old_value, new_value, changed_at
	FROM task_history
	WHERE task_id = ?
	ORDER BY changed_at, id
`

func (r *TaskHistoryRepository) ListByTask(ctx context.Context, taskID int64) ([]entity.TaskHistory, error) {
	var history []entity.TaskHistory
	if err := sqlx.SelectContext(ctx, queryExecutor(ctx, r.db), &history, queryListTaskHistory, taskID); err != nil {
		return nil, errors.WithStack(errors.Wrap(err, "failed to list task history"))
	}

	return history, nil
}
