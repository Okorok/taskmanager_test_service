package repository

import (
	"context"

	"taskmanager/internal/entity"

	"github.com/jmoiron/sqlx"
	"github.com/pkg/errors"
)

type TaskCommentRepository struct {
	db *sqlx.DB
}

func NewTaskCommentRepository(db *sqlx.DB) *TaskCommentRepository {
	return &TaskCommentRepository{db: db}
}

type CreateTaskCommentRequest struct {
	TaskID int64
	UserID int64
	Body   string
}

const queryInsertTaskComment = `
	INSERT INTO task_comments (task_id, user_id, body)
	VALUES (?, ?, ?)
`

func (r *TaskCommentRepository) Create(ctx context.Context, request CreateTaskCommentRequest) (*entity.TaskComment, error) {
	res, err := queryExecutor(ctx, r.db).ExecContext(ctx, queryInsertTaskComment,
		request.TaskID,
		request.UserID,
		request.Body,
	)
	if err != nil {
		return nil, errors.WithStack(errors.Wrap(err, "failed to insert task comment"))
	}

	id, err := res.LastInsertId()
	if err != nil {
		return nil, errors.WithStack(errors.Wrap(err, "failed to get inserted comment id"))
	}

	return r.GetByID(ctx, id)
}

const queryGetTaskCommentByID = `
	SELECT id, task_id, user_id, body, created_at
	FROM task_comments
	WHERE id = ?
`

func (r *TaskCommentRepository) GetByID(ctx context.Context, id int64) (*entity.TaskComment, error) {
	var comment entity.TaskComment
	if err := sqlx.GetContext(ctx, queryExecutor(ctx, r.db), &comment, queryGetTaskCommentByID, id); err != nil {
		return nil, errors.WithStack(errors.Wrap(err, "failed to get task comment by id"))
	}

	return &comment, nil
}

const queryListTaskComments = `
	SELECT id, task_id, user_id, body, created_at
	FROM task_comments
	WHERE task_id = ?
	ORDER BY created_at, id
`

func (r *TaskCommentRepository) ListByTask(ctx context.Context, taskID int64) ([]entity.TaskComment, error) {
	var comments []entity.TaskComment
	if err := sqlx.SelectContext(ctx, queryExecutor(ctx, r.db), &comments, queryListTaskComments, taskID); err != nil {
		return nil, errors.WithStack(errors.Wrap(err, "failed to list task comments"))
	}

	return comments, nil
}
