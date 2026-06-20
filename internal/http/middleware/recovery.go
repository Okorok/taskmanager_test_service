package middleware

import (
	"log/slog"
	"net/http"
)

type RecoveryMiddleware struct {
	logger *slog.Logger
}

func NewRecoveryMiddleware(logger *slog.Logger) *RecoveryMiddleware {
	return &RecoveryMiddleware{
		logger: logger,
	}
}

func (rm *RecoveryMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				rm.logger.Error(
					"panic recovered",
					"error", err,
					"method", r.Method,
					"path", r.URL.Path,
				)

				http.Error(w, "internal server error", http.StatusInternalServerError)
			}
		}()
		next.ServeHTTP(w, r)
	})
}
