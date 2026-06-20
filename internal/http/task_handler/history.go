package task_handler

import (
	"context"
	"log/slog"
	"net/http"

	"taskmanager/internal/entity"
	"taskmanager/internal/http/web"
)

type HistoryGetter interface {
	GetHistory(ctx context.Context, actorID, taskID int64) ([]entity.TaskHistory, error)
}

type HistoryHandler struct {
	service HistoryGetter
	logger  *slog.Logger
}

func NewHistoryHandler(service HistoryGetter, logger *slog.Logger) *HistoryHandler {
	return &HistoryHandler{service: service, logger: logger}
}

func (h *HistoryHandler) Handle(w http.ResponseWriter, r *http.Request, rawTaskID string) {
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

	history, err := h.service.GetHistory(r.Context(), actorID, taskID)
	if err != nil {
		h.logger.Error("failed to get task history", "error", err.Error(), "task_id", taskID)

		web.DomainError(w, err)
		return
	}

	web.Success(w, toHistoryResponses(history))
}
