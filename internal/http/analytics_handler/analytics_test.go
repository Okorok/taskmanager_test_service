package analytics_handler

import (
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"taskmanager/internal/http/analytics_handler/mocks"
	"taskmanager/internal/repository"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func Test_team_stats_ok(t *testing.T) {
	repo := mocks.NewMockAnalytics(t)
	repo.EXPECT().TeamStats(mock.Anything).Return([]repository.TeamStats{{TeamID: 1, TeamName: "A", MembersCount: 2}}, nil)

	h := NewAnalyticsHandler(repo, slog.Default())

	rr := httptest.NewRecorder()
	h.TeamStats(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "members_count")
}

func Test_top_creators_ok(t *testing.T) {
	repo := mocks.NewMockAnalytics(t)
	repo.EXPECT().TopCreatorsPerTeam(mock.Anything).Return([]repository.TopCreator{{TeamID: 1, UserID: 2, TasksCreated: 3, Rank: 1}}, nil)

	h := NewAnalyticsHandler(repo, slog.Default())

	rr := httptest.NewRecorder()
	h.TopCreators(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "tasks_created")
}

func Test_inconsistent_tasks_ok(t *testing.T) {
	repo := mocks.NewMockAnalytics(t)
	repo.EXPECT().InconsistentTasks(mock.Anything).Return([]repository.InconsistentTask{{TaskID: 1, TeamID: 1, AssigneeID: 9}}, nil)

	h := NewAnalyticsHandler(repo, slog.Default())

	rr := httptest.NewRecorder()
	h.InconsistentTasks(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Body.String(), "task_id")
}

func Test_analytics_repo_error_returns_500(t *testing.T) {
	repo := mocks.NewMockAnalytics(t)
	repo.EXPECT().TeamStats(mock.Anything).Return(nil, assert.AnError)

	h := NewAnalyticsHandler(repo, slog.Default())

	rr := httptest.NewRecorder()
	h.TeamStats(rr, httptest.NewRequest(http.MethodGet, "/", nil))

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}
