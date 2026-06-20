package team_handler

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

type TeamInviter interface {
	Invite(ctx context.Context, cmd service.InviteCommand) error
}

type InviteHandler struct {
	service TeamInviter
	logger  *slog.Logger
}

func NewInviteHandler(service TeamInviter, logger *slog.Logger) *InviteHandler {
	return &InviteHandler{service: service, logger: logger}
}

type inviteRequest struct {
	UserID int64  `json:"user_id"`
	Role   string `json:"role"`
}

func (h *InviteHandler) Handle(w http.ResponseWriter, r *http.Request, rawTeamID string) {
	actorID, ok := web.AuthUserID(r)
	if !ok {
		web.Unauthorized(w)
		return
	}

	teamID, err := parseID(rawTeamID)
	if err != nil {
		h.logger.Error("failed to parse team id", "error", err.Error(), "raw_id", rawTeamID)

		web.BadRequest(w, err)
		return
	}

	req, err := parseInviteRequest(r)
	if err != nil {
		h.logger.Error("failed to parse invite request", "error", err.Error())

		web.BadRequest(w, err)
		return
	}

	err = h.service.Invite(r.Context(), service.InviteCommand{
		TeamID:    teamID,
		ActorID:   actorID,
		InviteeID: req.UserID,
		Role:      entity.TeamRole(req.Role),
	})
	if err != nil {
		h.logger.Error("failed to invite user", "error", err.Error(), "team_id", teamID, "invitee_id", req.UserID)

		web.DomainError(w, err)
		return
	}

	web.Success(w, map[string]string{"status": "ok"})
}

func parseInviteRequest(r *http.Request) (*inviteRequest, error) {
	var req inviteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.New("invalid json")
	}

	if req.UserID <= 0 {
		return nil, errors.New("user_id is required")
	}

	if req.Role == "" {
		return nil, errors.New("role is required")
	}

	return &req, nil
}

func parseID(raw string) (int64, error) {
	id, err := strconv.ParseInt(raw, 10, 64)
	if err != nil || id <= 0 {
		return 0, errors.New("invalid id")
	}

	return id, nil
}
