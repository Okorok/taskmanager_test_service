package service

import (
	"context"
	"testing"

	"taskmanager/internal/entity"
	"taskmanager/internal/infrastructure"
	"taskmanager/internal/repository"
	"taskmanager/internal/service/mocks"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func Test_add_comment_successfully(t *testing.T) {
	stored := &entity.Task{ID: 1, TeamID: 10}
	created := &entity.TaskComment{ID: 5, TaskID: 1, UserID: 1, Body: "hello"}

	tasks := mocks.NewMockCommentTaskRepository(t)
	tasks.EXPECT().GetByID(mock.Anything, int64(1)).Return(stored, nil)

	comments := mocks.NewMockCommentRepository(t)
	comments.EXPECT().
		Create(mock.Anything, mock.Anything).
		Run(func(_ context.Context, req repository.CreateTaskCommentRequest) {
			assert.Equal(t, int64(1), req.TaskID)
			assert.Equal(t, "hello", req.Body)
		}).
		Return(created, nil)

	svc := NewCommentService(tasks, comments, memberForAll(t))

	comment, err := svc.AddComment(t.Context(), 1, 1, "hello")

	require.NoError(t, err)
	assert.Equal(t, created, comment)
}

func Test_add_comment_forbidden_for_non_member(t *testing.T) {
	stored := &entity.Task{ID: 1, TeamID: 10}

	tasks := mocks.NewMockCommentTaskRepository(t)
	tasks.EXPECT().GetByID(mock.Anything, int64(1)).Return(stored, nil)

	members := mocks.NewMockMembershipRepository(t)
	members.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil, infrastructure.ErrNotFound)

	svc := NewCommentService(tasks, mocks.NewMockCommentRepository(t), members)

	comment, err := svc.AddComment(t.Context(), 2, 1, "hello")

	assert.ErrorIs(t, err, ErrForbidden)
	assert.Nil(t, comment)
}

func Test_add_comment_task_not_found(t *testing.T) {
	tasks := mocks.NewMockCommentTaskRepository(t)
	tasks.EXPECT().GetByID(mock.Anything, int64(1)).Return(nil, infrastructure.ErrNotFound)

	svc := NewCommentService(tasks, mocks.NewMockCommentRepository(t), memberForAll(t))

	comment, err := svc.AddComment(t.Context(), 1, 1, "hello")

	assert.ErrorIs(t, err, infrastructure.ErrNotFound)
	assert.Nil(t, comment)
}

func Test_list_comments_successfully(t *testing.T) {
	stored := &entity.Task{ID: 1, TeamID: 10}
	expected := []entity.TaskComment{{ID: 1}, {ID: 2}}

	tasks := mocks.NewMockCommentTaskRepository(t)
	tasks.EXPECT().GetByID(mock.Anything, int64(1)).Return(stored, nil)

	comments := mocks.NewMockCommentRepository(t)
	comments.EXPECT().ListByTask(mock.Anything, int64(1)).Return(expected, nil)

	svc := NewCommentService(tasks, comments, memberForAll(t))

	result, err := svc.ListComments(t.Context(), 1, 1)

	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func Test_list_comments_forbidden_for_non_member(t *testing.T) {
	stored := &entity.Task{ID: 1, TeamID: 10}

	tasks := mocks.NewMockCommentTaskRepository(t)
	tasks.EXPECT().GetByID(mock.Anything, int64(1)).Return(stored, nil)

	members := mocks.NewMockMembershipRepository(t)
	members.EXPECT().Get(mock.Anything, mock.Anything, mock.Anything).Return(nil, infrastructure.ErrNotFound)

	svc := NewCommentService(tasks, mocks.NewMockCommentRepository(t), members)

	result, err := svc.ListComments(t.Context(), 2, 1)

	assert.ErrorIs(t, err, ErrForbidden)
	assert.Nil(t, result)
}
