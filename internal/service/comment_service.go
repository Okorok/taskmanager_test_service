package service

import (
	"context"

	"taskmanager/internal/entity"
	"taskmanager/internal/repository"
)

type CommentTaskRepository interface {
	GetByID(ctx context.Context, id int64) (*entity.Task, error)
}

type CommentRepository interface {
	Create(ctx context.Context, request repository.CreateTaskCommentRequest) (*entity.TaskComment, error)
	ListByTask(ctx context.Context, taskID int64) ([]entity.TaskComment, error)
}

type CommentService struct {
	tasks    CommentTaskRepository
	comments CommentRepository
	members  MembershipRepository
}

func NewCommentService(tasks CommentTaskRepository, comments CommentRepository, members MembershipRepository) *CommentService {
	return &CommentService{
		tasks:    tasks,
		comments: comments,
		members:  members,
	}
}

func (s *CommentService) AddComment(ctx context.Context, actorID, taskID int64, body string) (*entity.TaskComment, error) {
	if err := s.ensureMember(ctx, actorID, taskID); err != nil {
		return nil, err
	}

	return s.comments.Create(ctx, repository.CreateTaskCommentRequest{
		TaskID: taskID,
		UserID: actorID,
		Body:   body,
	})
}

func (s *CommentService) ListComments(ctx context.Context, actorID, taskID int64) ([]entity.TaskComment, error) {
	if err := s.ensureMember(ctx, actorID, taskID); err != nil {
		return nil, err
	}

	return s.comments.ListByTask(ctx, taskID)
}

func (s *CommentService) ensureMember(ctx context.Context, actorID, taskID int64) error {
	task, err := s.tasks.GetByID(ctx, taskID)
	if err != nil {
		return err
	}

	member, err := isTeamMember(ctx, s.members, task.TeamID, actorID)
	if err != nil {
		return err
	}
	if !member {
		return ErrForbidden
	}

	return nil
}
