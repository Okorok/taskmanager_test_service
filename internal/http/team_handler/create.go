package team_handler

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

type TeamCreator interface {
	CreateTeam(ctx context.Context, ownerID int64, name string) (*entity.Team, error)
}

type CreateTeamHandler struct {
	service TeamCreator
	logger  *slog.Logger
}

func NewCreateTeamHandler(service TeamCreator, logger *slog.Logger) *CreateTeamHandler {
	return &CreateTeamHandler{service: service, logger: logger}
}

type createTeamRequest struct {
	Name string `json:"name"`
}

func (h *CreateTeamHandler) Handle(w http.ResponseWriter, r *http.Request) {
	userID, ok := web.AuthUserID(r)
	if !ok {
		web.Unauthorized(w)
		return
	}

	name, err := parseCreateTeamRequest(r)
	if err != nil {
		h.logger.Error("failed to parse create team request", "error", err.Error())

		web.BadRequest(w, err)
		return
	}

	team, err := h.service.CreateTeam(r.Context(), userID, name)
	if err != nil {
		h.logger.Error("failed to create team", "error", err.Error(), "owner_id", userID)

		web.DomainError(w, err)
		return
	}

	web.Created(w, toTeamResponse(team))
}

func parseCreateTeamRequest(r *http.Request) (string, error) {
	var req createTeamRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return "", errors.New("invalid json")
	}

	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" {
		return "", errors.New("name is required")
	}

	return req.Name, nil
}
