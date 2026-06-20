package web

import (
	"encoding/json"
	"errors"
	"net/http"

	"taskmanager/internal/infrastructure"
	"taskmanager/internal/service"
)

type errorResponse struct {
	Error string `json:"error"`
}

func JSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func Error(w http.ResponseWriter, status int, msg string) {
	JSON(w, status, errorResponse{Error: msg})
}

func Success(w http.ResponseWriter, value any) {
	JSON(w, http.StatusOK, value)
}

func Created(w http.ResponseWriter, value any) {
	JSON(w, http.StatusCreated, value)
}

func BadRequest(w http.ResponseWriter, err error) {
	Error(w, http.StatusBadRequest, err.Error())
}

func DomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidCredentials):
		Error(w, http.StatusUnauthorized, "invalid credentials")
	case errors.Is(err, service.ErrEmailAlreadyTaken):
		Error(w, http.StatusConflict, "email already taken")
	case errors.Is(err, service.ErrAlreadyMember):
		Error(w, http.StatusConflict, "user is already a team member")
	case errors.Is(err, service.ErrForbidden):
		Error(w, http.StatusForbidden, "forbidden")
	case errors.Is(err, service.ErrInvalidRole):
		Error(w, http.StatusBadRequest, "invalid role")
	case errors.Is(err, service.ErrInvalidStatus):
		Error(w, http.StatusBadRequest, "invalid task status")
	case errors.Is(err, service.ErrAssigneeNotMember):
		Error(w, http.StatusUnprocessableEntity, "assignee is not a team member")
	case errors.Is(err, service.ErrUserNotFound):
		Error(w, http.StatusNotFound, "user not found")
	case errors.Is(err, infrastructure.ErrNotFound):
		Error(w, http.StatusNotFound, "not found")
	default:
		Error(w, http.StatusInternalServerError, "internal error")
	}
}
