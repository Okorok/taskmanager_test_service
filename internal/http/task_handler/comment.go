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
)

type CommentAdder interface {
	AddComment(ctx context.Context, actorID, taskID int64, body string) (*entity.TaskComment, error)
}

type CommentLister interface {
	ListComments(ctx context.Context, actorID, taskID int64) ([]entity.TaskComment, error)
}

type CommentHandler struct {
	adder  CommentAdder
	lister CommentLister
	logger *slog.Logger
}

func NewCommentHandler(adder CommentAdder, lister CommentLister, logger *slog.Logger) *CommentHandler {
	return &CommentHandler{adder: adder, lister: lister, logger: logger}
}

type addCommentRequest struct {
	Body string `json:"body"`
}

func (h *CommentHandler) Add(w http.ResponseWriter, r *http.Request, rawTaskID string) {
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

	var req addCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		web.BadRequest(w, errors.New("invalid json"))
		return
	}

	req.Body = strings.TrimSpace(req.Body)
	if req.Body == "" {
		web.BadRequest(w, errors.New("body is required"))
		return
	}

	comment, err := h.adder.AddComment(r.Context(), actorID, taskID, req.Body)
	if err != nil {
		h.logger.Error("failed to add comment", "error", err.Error(), "task_id", taskID)

		web.DomainError(w, err)
		return
	}

	web.Created(w, toCommentResponse(comment))
}

func (h *CommentHandler) List(w http.ResponseWriter, r *http.Request, rawTaskID string) {
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

	comments, err := h.lister.ListComments(r.Context(), actorID, taskID)
	if err != nil {
		h.logger.Error("failed to list comments", "error", err.Error(), "task_id", taskID)

		web.DomainError(w, err)
		return
	}

	web.Success(w, toCommentResponses(comments))
}
