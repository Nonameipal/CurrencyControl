package auth

import (
	"context"
	"net/http"

	"CurrencyControl/internal/delivery/http/middleware"
	"CurrencyControl/internal/delivery/http/response"
)

type CommonError = response.CommonError
type DiffTracker = response.DiffTracker

var (
	writeJSON      = response.WriteJSON
	decodeJSON     = response.DecodeJSON
	handleError    = response.HandleError
	requireID      = response.RequireID
	NewDiffTracker = response.NewDiffTracker
)

func getClientIP(r *http.Request) string {
	return middleware.GetClientIP(r)
}

func extractToken(r *http.Request) string {
	return middleware.ExtractToken(r)
}

func GetLoginFromContext(ctx context.Context) string {
	return middleware.GetLoginFromContext(ctx)
}

func GetRoleFromContext(ctx context.Context) string {
	return middleware.GetRoleFromContext(ctx)
}

func GetLastNameFromContext(ctx context.Context) string {
	return middleware.GetLastNameFromContext(ctx)
}

func GetFirstNameFromContext(ctx context.Context) string {
	return middleware.GetFirstNameFromContext(ctx)
}

func GetEmailFromContext(ctx context.Context) string {
	return middleware.GetEmailFromContext(ctx)
}

func LogUserAction(r *http.Request, action, entity string, entityID *int64, details string) {
	middleware.LogUserAction(r, action, entity, entityID, details)
}
