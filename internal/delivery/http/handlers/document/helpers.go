package document

import (
	"context"
	"net/http"

	"CurrencyControl/internal/delivery/http/middleware"
	"CurrencyControl/internal/delivery/http/response"
)

type CommonError = response.CommonError
type FormValidator = response.FormValidator
type DiffTracker = response.DiffTracker

var (
	writeJSON        = response.WriteJSON
	decodeJSON       = response.DecodeJSON
	handleError      = response.HandleError
	requireID        = response.RequireID
	parseDate        = response.ParseDate
	saveUploadedFile = response.SaveUploadedFile
	NewFormValidator = response.NewFormValidator
	NewDiffTracker   = response.NewDiffTracker

	contractRequiredFields            = response.ContractRequiredFields
	invoiceRequiredFields             = response.InvoiceRequiredFields
	gtdRequiredFields                 = response.GTDRequiredFields
	actRequiredFields                 = response.ActRequiredFields
	additionalAgreementRequiredFields = response.AdditionalAgreementRequiredFields
	paymentOrderRequiredFields        = response.PaymentOrderRequiredFields
)

func GetLoginFromContext(ctx context.Context) string {
	return middleware.GetLoginFromContext(ctx)
}

func GetRoleFromContext(ctx context.Context) string {
	return middleware.GetRoleFromContext(ctx)
}

func GetBranchIDFromContext(ctx context.Context) int64 {
	return middleware.GetBranchIDFromContext(ctx)
}

func LogUserAction(r *http.Request, action, entity string, entityID *int64, details string) {
	middleware.LogUserAction(r, action, entity, entityID, details)
}
