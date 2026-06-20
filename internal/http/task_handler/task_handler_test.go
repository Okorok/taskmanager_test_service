package task_handler

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"taskmanager/internal/entity"
	"taskmanager/internal/http/task_handler/mocks"
	"taskmanager/internal/infrastructure"
	"taskmanager/internal/service"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func authedRequest(method, target, body string, userID int64) *http.Request {
	var r *http.Request
	if body == "" {
		r = httptest.NewRequest(method, target, nil)
	} else {
		r = httptest.NewRequest(method, target, bytes.NewBufferString(body))
	}

	return r.WithContext(infrastructure.WithUserID(r.Context(), userID))
}

func Test_create_task_created(t *testing.T) {
	svc := mocks.NewMockTaskCreator(t)
	svc.EXPECT().CreateTask(mock.Anything, mock.Anything).Return(&entity.Task{ID: 1}, nil)

	h := NewCreateTaskHandler(svc, slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, authedRequest(http.MethodPost, "/", `{"team_id":10,"title":"Task"}`, 1))

	assert.Equal(t, http.StatusCreated, rr.Code)
}

func Test_create_task_unauthorized(t *testing.T) {
	h := NewCreateTaskHandler(mocks.NewMockTaskCreator(t), slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"team_id":10,"title":"Task"}`)))

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func Test_create_task_validation(t *testing.T) {
	tests := []struct {
		name string
		body string
	}{
		{"invalid_json", `nope`},
		{"missing_team", `{"title":"Task"}`},
		{"missing_title", `{"team_id":10}`},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := NewCreateTaskHandler(mocks.NewMockTaskCreator(t), slog.Default())

			rr := httptest.NewRecorder()
			h.Handle(rr, authedRequest(http.MethodPost, "/", tc.body, 1))

			assert.Equal(t, http.StatusBadRequest, rr.Code)
		})
	}
}

func Test_create_task_domain_errors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{"forbidden", service.ErrForbidden, http.StatusForbidden},
		{"assignee_not_member", service.ErrAssigneeNotMember, http.StatusUnprocessableEntity},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockTaskCreator(t)
			svc.EXPECT().CreateTask(mock.Anything, mock.Anything).Return(nil, tc.err)

			h := NewCreateTaskHandler(svc, slog.Default())

			rr := httptest.NewRecorder()
			h.Handle(rr, authedRequest(http.MethodPost, "/", `{"team_id":10,"title":"Task"}`, 1))

			assert.Equal(t, tc.wantCode, rr.Code)
		})
	}
}

func Test_list_tasks_ok(t *testing.T) {
	svc := mocks.NewMockTaskLister(t)
	svc.EXPECT().ListTasks(mock.Anything, mock.Anything).Return([]entity.Task{{ID: 1}}, nil)

	h := NewListTasksHandler(svc, slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, authedRequest(http.MethodGet, "/?team_id=10&status=todo&limit=5", "", 1))

	assert.Equal(t, http.StatusOK, rr.Code)
}

func Test_list_tasks_missing_team_id(t *testing.T) {
	h := NewListTasksHandler(mocks.NewMockTaskLister(t), slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, authedRequest(http.MethodGet, "/", "", 1))

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func Test_list_tasks_forbidden(t *testing.T) {
	svc := mocks.NewMockTaskLister(t)
	svc.EXPECT().ListTasks(mock.Anything, mock.Anything).Return(nil, service.ErrForbidden)

	h := NewListTasksHandler(svc, slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, authedRequest(http.MethodGet, "/?team_id=10", "", 1))

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func Test_update_task_ok(t *testing.T) {
	svc := mocks.NewMockTaskUpdater(t)
	svc.EXPECT().UpdateTask(mock.Anything, mock.Anything).Return(&entity.Task{ID: 1}, nil)

	h := NewUpdateTaskHandler(svc, slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, authedRequest(http.MethodPut, "/", `{"title":"new"}`, 1), "1")

	assert.Equal(t, http.StatusOK, rr.Code)
}

func Test_update_task_invalid_id(t *testing.T) {
	h := NewUpdateTaskHandler(mocks.NewMockTaskUpdater(t), slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, authedRequest(http.MethodPut, "/", `{"title":"new"}`, 1), "abc")

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func Test_update_task_invalid_json(t *testing.T) {
	h := NewUpdateTaskHandler(mocks.NewMockTaskUpdater(t), slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, authedRequest(http.MethodPut, "/", `nope`, 1), "1")

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func Test_update_task_domain_errors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{"not_found", infrastructure.ErrNotFound, http.StatusNotFound},
		{"invalid_status", service.ErrInvalidStatus, http.StatusBadRequest},
		{"forbidden", service.ErrForbidden, http.StatusForbidden},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockTaskUpdater(t)
			svc.EXPECT().UpdateTask(mock.Anything, mock.Anything).Return(nil, tc.err)

			h := NewUpdateTaskHandler(svc, slog.Default())

			rr := httptest.NewRecorder()
			h.Handle(rr, authedRequest(http.MethodPut, "/", `{"status":"x"}`, 1), "1")

			assert.Equal(t, tc.wantCode, rr.Code)
		})
	}
}

func Test_history_ok(t *testing.T) {
	svc := mocks.NewMockHistoryGetter(t)
	svc.EXPECT().GetHistory(mock.Anything, int64(1), int64(1)).Return([]entity.TaskHistory{{ID: 1}}, nil)

	h := NewHistoryHandler(svc, slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, authedRequest(http.MethodGet, "/", "", 1), "1")

	assert.Equal(t, http.StatusOK, rr.Code)
}

func Test_history_not_found(t *testing.T) {
	svc := mocks.NewMockHistoryGetter(t)
	svc.EXPECT().GetHistory(mock.Anything, mock.Anything, mock.Anything).Return(nil, infrastructure.ErrNotFound)

	h := NewHistoryHandler(svc, slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, authedRequest(http.MethodGet, "/", "", 1), "1")

	assert.Equal(t, http.StatusNotFound, rr.Code)
}

func Test_history_unauthorized(t *testing.T) {
	h := NewHistoryHandler(mocks.NewMockHistoryGetter(t), slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, httptest.NewRequest(http.MethodGet, "/", nil), "1")

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}
