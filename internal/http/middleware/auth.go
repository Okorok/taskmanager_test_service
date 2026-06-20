package middleware

import (
	"net/http"
	"strings"

	"taskmanager/internal/infrastructure"
)

type TokenParser interface {
	Parse(token string) (int64, error)
}

type AuthMiddleware struct {
	tokens TokenParser
}

func NewAuthMiddleware(tokens TokenParser) *AuthMiddleware {
	return &AuthMiddleware{
		tokens: tokens,
	}
}

func (am *AuthMiddleware) Handle(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		auth := r.Header.Get("Authorization")
		if auth == "" || !strings.HasPrefix(auth, "Bearer ") {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		token := strings.TrimPrefix(auth, "Bearer ")

		userID, err := am.tokens.Parse(token)
		if err != nil {
			http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
			return
		}

		ctx := infrastructure.WithUserID(r.Context(), userID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
