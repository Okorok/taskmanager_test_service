package task_handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"taskmanager/internal/entity"
	"taskmanager/internal/http/web"
	"taskmanager/internal/service"
)

type TaskCreator interface {
	CreateTask(ctx context.Context, cmd service.CreateTaskCommand) (*entity.Task, error)
}

type CreateTaskHandler struct {
	service TaskCreator
	logger  *slog.Logger
}

func NewCreateTaskHandler(service TaskCreator, logger *slog.Logger) *CreateTaskHandler {
	return &CreateTaskHandler{service: service, logger: logger}
}

type createTaskRequest struct {
	TeamID      int64  `json:"team_id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Priority    string `json:"priority"`
	AssigneeID  int64  `json:"assignee_id"`
}

func (h *CreateTaskHandler) Handle(w http.ResponseWriter, r *http.Request) {
	userID, ok := web.AuthUserID(r)
	if !ok {
		web.Unauthorized(w)
		return
	}

	req, err := parseCreateTaskRequest(r)
	if err != nil {
		h.logger.Error("failed to parse create task request", "error", err.Error())

		web.BadRequest(w, err)
		return
	}

	task, err := h.service.CreateTask(r.Context(), service.CreateTaskCommand{
		TeamID:      req.TeamID,
		CreatorID:   userID,
		Title:       req.Title,
		Description: req.Description,
		Priority:    req.Priority,
		AssigneeID:  req.AssigneeID,
	})
	if err != nil {
		h.logger.Error("failed to create task", "error", err.Error(), "team_id", req.TeamID)

		web.DomainError(w, err)
		return
	}

	web.Created(w, toTaskResponse(task))
}

func parseCreateTaskRequest(r *http.Request) (*createTaskRequest, error) {
	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.New("invalid json")
	}

	req.Title = strings.TrimSpace(req.Title)

	if req.TeamID <= 0 {
		return nil, errors.New("team_id is required")
	}

	if req.Title == "" {
		return nil, errors.New("title is required")
	}

	return &req, nil
}
