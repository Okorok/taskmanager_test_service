package team_handler

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"taskmanager/internal/entity"
	"taskmanager/internal/http/team_handler/mocks"
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

func Test_create_team_created(t *testing.T) {
	svc := mocks.NewMockTeamCreator(t)
	svc.EXPECT().CreateTeam(mock.Anything, int64(1), "Platform").Return(&entity.Team{ID: 1}, nil)

	h := NewCreateTeamHandler(svc, slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, authedRequest(http.MethodPost, "/", `{"name":"Platform"}`, 1))

	assert.Equal(t, http.StatusCreated, rr.Code)
}

func Test_create_team_unauthorized(t *testing.T) {
	h := NewCreateTeamHandler(mocks.NewMockTeamCreator(t), slog.Default())

	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(`{"name":"Platform"}`))
	h.Handle(rr, req)

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func Test_create_team_empty_name(t *testing.T) {
	h := NewCreateTeamHandler(mocks.NewMockTeamCreator(t), slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, authedRequest(http.MethodPost, "/", `{"name":"  "}`, 1))

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func Test_list_teams_ok(t *testing.T) {
	svc := mocks.NewMockTeamLister(t)
	svc.EXPECT().ListTeams(mock.Anything, int64(1)).Return([]entity.Team{{ID: 1}}, nil)

	h := NewListTeamsHandler(svc, slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, authedRequest(http.MethodGet, "/", "", 1))

	assert.Equal(t, http.StatusOK, rr.Code)
}

func Test_list_teams_unauthorized(t *testing.T) {
	h := NewListTeamsHandler(mocks.NewMockTeamLister(t), slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusUnauthorized, rr.Code)
}

func Test_invite_ok(t *testing.T) {
	svc := mocks.NewMockTeamInviter(t)
	svc.EXPECT().Invite(mock.Anything, mock.Anything).Return(nil)

	h := NewInviteHandler(svc, slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, authedRequest(http.MethodPost, "/", `{"user_id":2,"role":"member"}`, 1), "10")

	assert.Equal(t, http.StatusOK, rr.Code)
}

func Test_invite_invalid_team_id(t *testing.T) {
	h := NewInviteHandler(mocks.NewMockTeamInviter(t), slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, authedRequest(http.MethodPost, "/", `{"user_id":2,"role":"member"}`, 1), "abc")

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func Test_invite_missing_user_id(t *testing.T) {
	h := NewInviteHandler(mocks.NewMockTeamInviter(t), slog.Default())

	rr := httptest.NewRecorder()
	h.Handle(rr, authedRequest(http.MethodPost, "/", `{"role":"member"}`, 1), "10")

	assert.Equal(t, http.StatusBadRequest, rr.Code)
}

func Test_invite_domain_errors(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		wantCode int
	}{
		{"forbidden", service.ErrForbidden, http.StatusForbidden},
		{"invalid_role", service.ErrInvalidRole, http.StatusBadRequest},
		{"user_not_found", service.ErrUserNotFound, http.StatusNotFound},
		{"already_member", service.ErrAlreadyMember, http.StatusConflict},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			svc := mocks.NewMockTeamInviter(t)
			svc.EXPECT().Invite(mock.Anything, mock.Anything).Return(tc.err)

			h := NewInviteHandler(svc, slog.Default())

			rr := httptest.NewRecorder()
			h.Handle(rr, authedRequest(http.MethodPost, "/", `{"user_id":2,"role":"member"}`, 1), "10")

			assert.Equal(t, tc.wantCode, rr.Code)
		})
	}
}
