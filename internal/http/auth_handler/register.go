package auth_handler

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

const minPasswordLength = 6

type Registrar interface {
	Register(ctx context.Context, cmd service.RegisterCommand) (*entity.User, error)
}

type RegisterHandler struct {
	service Registrar
	logger  *slog.Logger
}

func NewRegisterHandler(service Registrar, logger *slog.Logger) *RegisterHandler {
	return &RegisterHandler{service: service, logger: logger}
}

type registerRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

func (h *RegisterHandler) Handle(w http.ResponseWriter, r *http.Request) {
	req, err := parseRegisterRequest(r)
	if err != nil {
		h.logger.Error("failed to parse register request", "error", err.Error())

		web.BadRequest(w, err)
		return
	}

	user, err := h.service.Register(r.Context(), service.RegisterCommand{
		Email:    req.Email,
		Password: req.Password,
		Name:     req.Name,
	})
	if err != nil {
		h.logger.Error("failed to register user", "error", err.Error(), "email", req.Email)

		web.DomainError(w, err)
		return
	}

	web.Created(w, toUserResponse(user))
}

func parseRegisterRequest(r *http.Request) (*registerRequest, error) {
	var req registerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.New("invalid json")
	}

	req.Email = strings.TrimSpace(req.Email)
	req.Name = strings.TrimSpace(req.Name)

	if !strings.Contains(req.Email, "@") || req.Email == "" {
		return nil, errors.New("invalid email")
	}

	if len(req.Password) < minPasswordLength {
		return nil, errors.New("password must be at least 6 characters")
	}

	if req.Name == "" {
		return nil, errors.New("name is required")
	}

	return &req, nil
}
