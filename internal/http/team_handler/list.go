package team_handler

import (
	"context"
	"log/slog"
	"net/http"

	"taskmanager/internal/entity"
	"taskmanager/internal/http/web"
)

type TeamLister interface {
	ListTeams(ctx context.Context, userID int64) ([]entity.Team, error)
}

type ListTeamsHandler struct {
	service TeamLister
	logger  *slog.Logger
}

func NewListTeamsHandler(service TeamLister, logger *slog.Logger) *ListTeamsHandler {
	return &ListTeamsHandler{service: service, logger: logger}
}

func (h *ListTeamsHandler) Handle(w http.ResponseWriter, r *http.Request) {
	userID, ok := web.AuthUserID(r)
	if !ok {
		web.Unauthorized(w)
		return
	}

	teams, err := h.service.ListTeams(r.Context(), userID)
	if err != nil {
		h.logger.Error("failed to list teams", "error", err.Error(), "user_id", userID)

		web.DomainError(w, err)
		return
	}

	web.Success(w, toTeamResponses(teams))
}
