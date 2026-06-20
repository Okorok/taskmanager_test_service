package task_handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"taskmanager/internal/entity"
	"taskmanager/internal/http/web"
	"taskmanager/internal/service"
)

type TaskUpdater interface {
	UpdateTask(ctx context.Context, cmd service.UpdateTaskCommand) (*entity.Task, error)
}

type UpdateTaskHandler struct {
	service TaskUpdater
	logger  *slog.Logger
}

func NewUpdateTaskHandler(service TaskUpdater, logger *slog.Logger) *UpdateTaskHandler {
	return &UpdateTaskHandler{service: service, logger: logger}
}

type updateTaskRequest struct {
	Title       *string `json:"title"`
	Description *string `json:"description"`
	Status      *string `json:"status"`
	Priority    *string `json:"priority"`
	AssigneeID  *int64  `json:"assignee_id"`
}

func (h *UpdateTaskHandler) Handle(w http.ResponseWriter, r *http.Request, rawTaskID string) {
	actorID, ok := web.AuthUserID(r)
	if !ok {
		web.Unauthorized(w)
		return
	}

	taskID, err := parseTaskID(rawTaskID)
	if err != nil {
		h.logger.Error("failed to parse task id", "error", err.Error(), "raw_id", rawTaskID)

		web.BadRequest(w, err)
		return
	}

	var req updateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.logger.Error("failed to parse update task request", "error", err.Error())

		web.BadRequest(w, errors.New("invalid json"))
		return
	}

	task, err := h.service.UpdateTask(r.Context(), service.UpdateTaskCommand{
		TaskID:      taskID,
		ActorID:     actorID,
		Title:       req.Title,
		Description: req.Description,
		Status:      req.Status,
		Priority:    req.Priority,
		AssigneeID:  req.AssigneeID,
	})
	if err != nil {
		h.logger.Error("failed to update task", "error", err.Error(), "task_id", taskID)

		web.DomainError(w, err)
		return
	}

	web.Success(w, toTaskResponse(task))
}

func parseTaskID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid task id")
	}

	return id, nil
}
