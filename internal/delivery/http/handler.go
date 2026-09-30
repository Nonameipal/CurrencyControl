package http

import (
	"errors"
	"net/http"
	"strings"

	"CurrencyControl/internal/errs"
)

func handleError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	errMsg := err.Error()
	switch {
	case errors.Is(err, errs.ErrNotFound) ||
		errors.Is(err, errs.ErrContractNotFound) ||
		strings.Contains(errMsg, "не найден") ||
		strings.Contains(errMsg, "не найдена") ||
		strings.Contains(errMsg, "не найдено") ||
		strings.Contains(errMsg, "не существует") ||
		strings.Contains(errMsg, "not found"):
		writeJSON(w, http.StatusNotFound, CommonError{Error: errMsg})

	case errors.Is(err, errs.ErrInvalidRequestBody) ||
		errors.Is(err, errs.ErrInvalidFieldValue) ||
		strings.Contains(errMsg, "не находится в архиве"):
		writeJSON(w, http.StatusBadRequest, CommonError{Error: errMsg})

	case errors.Is(err, errs.ErrUnauthorized):
		writeJSON(w, http.StatusUnauthorized, CommonError{Error: errMsg})

	case errors.Is(err, errs.ErrAccessDenied):
		writeJSON(w, http.StatusForbidden, CommonError{Error: errMsg})

	case errors.Is(err, errs.ErrContractAlreadyExists):
		writeJSON(w, http.StatusUnprocessableEntity, CommonError{Error: errMsg})

	default:
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: errMsg})
	}
}
