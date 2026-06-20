package web

import (
	"net/http"

	"taskmanager/internal/infrastructure"
)

func AuthUserID(r *http.Request) (int64, bool) {
	return infrastructure.UserIDFromCtx(r.Context())
}

func Unauthorized(w http.ResponseWriter) {
	Error(w, http.StatusUnauthorized, "unauthorized")
}
