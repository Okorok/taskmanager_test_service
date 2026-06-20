package auth_handler

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"taskmanager/internal/http/web"
	"taskmanager/internal/service"
)

type Authenticator interface {
	Login(ctx context.Context, cmd service.LoginCommand) (string, error)
}

type LoginHandler struct {
	service Authenticator
	logger  *slog.Logger
}

func NewLoginHandler(service Authenticator, logger *slog.Logger) *LoginHandler {
	return &LoginHandler{service: service, logger: logger}
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type loginResponse struct {
	Token string `json:"token"`
}

func (h *LoginHandler) Handle(w http.ResponseWriter, r *http.Request) {
	req, err := parseLoginRequest(r)
	if err != nil {
		h.logger.Error("failed to parse login request", "error", err.Error())

		web.BadRequest(w, err)
		return
	}

	token, err := h.service.Login(r.Context(), service.LoginCommand{
		Email:    req.Email,
		Password: req.Password,
	})
	if err != nil {
		h.logger.Error("failed to login user", "error", err.Error(), "email", req.Email)

		web.DomainError(w, err)
		return
	}

	web.Success(w, loginResponse{Token: token})
}

func parseLoginRequest(r *http.Request) (*loginRequest, error) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		return nil, errors.New("invalid json")
	}

	req.Email = strings.TrimSpace(req.Email)

	if req.Email == "" || req.Password == "" {
		return nil, errors.New("email and password are required")
	}

	return &req, nil
}
