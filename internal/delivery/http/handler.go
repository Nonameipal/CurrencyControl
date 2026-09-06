package http

import (
	"errors"
	"net/http"

	"CurrencyControl/internal/errs"
)


func handleError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, errs.ErrNotFound) ||
		errors.Is(err, errs.ErrContractNotFound):
		writeJSON(w, http.StatusNotFound, CommonError{Error: err.Error()})

	case errors.Is(err, errs.ErrInvalidRequestBody) ||
		errors.Is(err, errs.ErrInvalidFieldValue):
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})

	case errors.Is(err, errs.ErrUnauthorized):
		writeJSON(w, http.StatusUnauthorized, CommonError{Error: err.Error()})

	case errors.Is(err, errs.ErrAccessDenied):
		writeJSON(w, http.StatusForbidden, CommonError{Error: err.Error()})

	case errors.Is(err, errs.ErrContractAlreadyExists) ||
		errors.Is(err, errs.ErrPaymentExceedsBalance):
		writeJSON(w, http.StatusUnprocessableEntity, CommonError{Error: err.Error()})

	default:
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: err.Error()})
	}
}
