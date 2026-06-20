package analytics_handler

import (
	"context"
	"log/slog"
	"net/http"

	"taskmanager/internal/http/web"
	"taskmanager/internal/repository"
)

type Analytics interface {
	TeamStats(ctx context.Context) ([]repository.TeamStats, error)
	TopCreatorsPerTeam(ctx context.Context) ([]repository.TopCreator, error)
	InconsistentTasks(ctx context.Context) ([]repository.InconsistentTask, error)
}

type AnalyticsHandler struct {
	repo   Analytics
	logger *slog.Logger
}

func NewAnalyticsHandler(repo Analytics, logger *slog.Logger) *AnalyticsHandler {
	return &AnalyticsHandler{repo: repo, logger: logger}
}

type teamStatsResponse struct {
	TeamID             int64  `json:"team_id"`
	TeamName           string `json:"team_name"`
	MembersCount       int64  `json:"members_count"`
	DoneTasksLast7Days int64  `json:"done_tasks_last_7_days"`
}

func (h *AnalyticsHandler) TeamStats(w http.ResponseWriter, r *http.Request) {
	stats, err := h.repo.TeamStats(r.Context())
	if err != nil {
		h.logger.Error("failed to get team stats", "error", err.Error())

		web.DomainError(w, err)
		return
	}

	result := make([]teamStatsResponse, 0, len(stats))
	for _, s := range stats {
		result = append(result, teamStatsResponse{
			TeamID:             s.TeamID,
			TeamName:           s.TeamName,
			MembersCount:       s.MembersCount,
			DoneTasksLast7Days: s.DoneTasksLast7Days,
		})
	}

	web.Success(w, result)
}

type topCreatorResponse struct {
	TeamID       int64 `json:"team_id"`
	UserID       int64 `json:"user_id"`
	TasksCreated int64 `json:"tasks_created"`
	Rank         int64 `json:"rank"`
}

func (h *AnalyticsHandler) TopCreators(w http.ResponseWriter, r *http.Request) {
	creators, err := h.repo.TopCreatorsPerTeam(r.Context())
	if err != nil {
		h.logger.Error("failed to get top creators", "error", err.Error())

		web.DomainError(w, err)
		return
	}

	result := make([]topCreatorResponse, 0, len(creators))
	for _, c := range creators {
		result = append(result, topCreatorResponse{
			TeamID:       c.TeamID,
			UserID:       c.UserID,
			TasksCreated: c.TasksCreated,
			Rank:         c.Rank,
		})
	}

	web.Success(w, result)
}

type inconsistentTaskResponse struct {
	TaskID     int64 `json:"task_id"`
	TeamID     int64 `json:"team_id"`
	AssigneeID int64 `json:"assignee_id"`
}

func (h *AnalyticsHandler) InconsistentTasks(w http.ResponseWriter, r *http.Request) {
	tasks, err := h.repo.InconsistentTasks(r.Context())
	if err != nil {
		h.logger.Error("failed to get inconsistent tasks", "error", err.Error())

		web.DomainError(w, err)
		return
	}

	result := make([]inconsistentTaskResponse, 0, len(tasks))
	for _, t := range tasks {
		result = append(result, inconsistentTaskResponse{
			TaskID:     t.TaskID,
			TeamID:     t.TeamID,
			AssigneeID: t.AssigneeID,
		})
	}

	web.Success(w, result)
}
