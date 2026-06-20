package task_handler

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"taskmanager/internal/entity"
	"taskmanager/internal/http/web"
	"taskmanager/internal/service"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

type TaskLister interface {
	ListTasks(ctx context.Context, query service.ListTasksQuery) ([]entity.Task, error)
}

type ListTasksHandler struct {
	service TaskLister
	logger  *slog.Logger
}

func NewListTasksHandler(service TaskLister, logger *slog.Logger) *ListTasksHandler {
	return &ListTasksHandler{service: service, logger: logger}
}

func (h *ListTasksHandler) Handle(w http.ResponseWriter, r *http.Request) {
	userID, ok := web.AuthUserID(r)
	if !ok {
		web.Unauthorized(w)
		return
	}

	query, err := parseListTasksQuery(r, userID)
	if err != nil {
		h.logger.Error("failed to parse list tasks query", "error", err.Error())

		web.BadRequest(w, err)
		return
	}

	tasks, err := h.service.ListTasks(r.Context(), query)
	if err != nil {
		h.logger.Error("failed to list tasks", "error", err.Error(), "team_id", query.TeamID)

		web.DomainError(w, err)
		return
	}

	web.Success(w, toTaskResponses(tasks))
}

func parseListTasksQuery(r *http.Request, actorID int64) (service.ListTasksQuery, error) {
	q := r.URL.Query()

	teamID, err := strconv.ParseInt(q.Get("team_id"), 10, 64)
	if err != nil || teamID <= 0 {
		return service.ListTasksQuery{}, errors.New("team_id is required")
	}

	var assigneeID int64
	if raw := q.Get("assignee_id"); raw != "" {
		assigneeID, err = strconv.ParseInt(raw, 10, 64)
		if err != nil || assigneeID < 0 {
			return service.ListTasksQuery{}, errors.New("invalid assignee_id")
		}
	}

	limit := defaultLimit
	if raw := q.Get("limit"); raw != "" {
		limit, err = strconv.Atoi(raw)
		if err != nil || limit <= 0 {
			return service.ListTasksQuery{}, errors.New("invalid limit")
		}
		if limit > maxLimit {
			limit = maxLimit
		}
	}

	offset := 0
	if raw := q.Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil || offset < 0 {
			return service.ListTasksQuery{}, errors.New("invalid offset")
		}
	}

	return service.ListTasksQuery{
		TeamID:     teamID,
		ActorID:    actorID,
		Status:     q.Get("status"),
		AssigneeID: assigneeID,
		Limit:      limit,
		Offset:     offset,
	}, nil
}
