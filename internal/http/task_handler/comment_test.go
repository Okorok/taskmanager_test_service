package task_handler

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"taskmanager/internal/entity"
	"taskmanager/internal/http/task_handler/mocks"
	"taskmanager/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_add_comment_created(t *testing.T) {
	adder := mocks.NewMockCommentAdder(t)
	adder.EXPECT().AddComment(mock.Anything, int64(1), int64(1), "hello").Return(&entity.TaskComment{ID: 1}, nil)

	h := NewCommentHandler(adder, mocks.NewMockCommentLister(t), slog.Default())

	rr := httptest.NewRecorder()
	h.Add(rr, authedRequest(http.MethodPost, "/", `{"body":"hello"}`, 1), "1")

	assert.Equal(t, http.StatusCreated, rr.Code)
}

func Test_add_comment_empty_body(t *testing.T) {
	h := NewCommentHandler(mocks.NewMockCommentAdder(t), mocks.NewMockCommentLister(t), slog.Default())

	rr := httptest.NewRecorder()
	h.Add(rr, authedRequest(http.MethodPost, "/", `{"body":"  "}`, 1), "1")

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func Test_add_comment_invalid_task_id(t *testing.T) {
	h := NewCommentHandler(mocks.NewMockCommentAdder(t), mocks.NewMockCommentLister(t), slog.Default())

	rr := httptest.NewRecorder()
	h.Add(rr, authedRequest(http.MethodPost, "/", `{"body":"hello"}`, 1), "abc")

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func Test_add_comment_unauthorized(t *testing.T) {
	h := NewCommentHandler(mocks.NewMockCommentAdder(t), mocks.NewMockCommentLister(t), slog.Default())

	rr := httptest.NewRecorder()
	h.Add(rr, httptest.NewRequest(http.MethodPost, "/", nil), "1")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func Test_list_comments_ok(t *testing.T) {
	lister := mocks.NewMockCommentLister(t)
	lister.EXPECT().ListComments(mock.Anything, int64(1), int64(1)).Return([]entity.TaskComment{{ID: 1}}, nil)

	h := NewCommentHandler(mocks.NewMockCommentAdder(t), lister, slog.Default())

	rr := httptest.NewRecorder()
	h.List(rr, authedRequest(http.MethodGet, "/", "", 1), "1")

	assert.Equal(t, http.StatusOK, rr.Code)
}

func Test_list_comments_forbidden(t *testing.T) {
	lister := mocks.NewMockCommentLister(t)
	lister.EXPECT().ListComments(mock.Anything, mock.Anything, mock.Anything).Return(nil, service.ErrForbidden)

	h := NewCommentHandler(mocks.NewMockCommentAdder(t), lister, slog.Default())

	rr := httptest.NewRecorder()
	h.List(rr, authedRequest(http.MethodGet, "/", "", 1), "1")

	assert.Equal(t, http.StatusForbidden, rr.Code)
}
