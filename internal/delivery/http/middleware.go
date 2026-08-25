package http

import (
	"context"
	"net/http"

	"CurrencyControl/internal/errs"
)

type contextKey string

const (
	LoginContextKey contextKey = "login"
)

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		login := r.Header.Get("Login")
		if login == "" {
			writeJSON(w, http.StatusUnauthorized, CommonError{Error: errs.ErrUnauthorized.Error()})
			return
		}
		ctx := context.WithValue(r.Context(), LoginContextKey, login)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetLoginFromContext(ctx context.Context) string {
	if login, ok := ctx.Value(LoginContextKey).(string); ok {
		return login
	}
	return ""
}
