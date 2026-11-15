package counterparty

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

func GetLoginFromContext(ctx context.Context) string {
	return middleware.GetLoginFromContext(ctx)
}

func LogUserAction(r *http.Request, action, entity string, entityID *int64, details string) {
	middleware.LogUserAction(r, action, entity, entityID, details)
}
