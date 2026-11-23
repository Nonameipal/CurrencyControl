package response

import (
	"errors"
	"net/http"
	"strings"

	"CurrencyControl/internal/errs"
)

func HandleError(w http.ResponseWriter, err error) {
	if err == nil {
		return
	}
	errMsg := err.Error()
	switch {
	case errors.Is(err, errs.ErrADUnavailable) ||
		strings.Contains(errMsg, "недоступен"):
		WriteJSON(w, http.StatusServiceUnavailable, CommonError{Error: errMsg})
	case errors.Is(err, errs.ErrInvalidRequestBody) ||
		errors.Is(err, errs.ErrInvalidFieldValue) ||
		strings.Contains(errMsg, "не находится в архиве") ||
		strings.Contains(errMsg, "недопустимое решение") ||
		strings.Contains(errMsg, "недопустимая роль") ||
		strings.Contains(errMsg, "нельзя назначить") ||
		strings.Contains(errMsg, "обязательна") ||
		strings.Contains(errMsg, "обязателен") ||
		strings.Contains(errMsg, "находится в статусе") ||
		strings.Contains(errMsg, "неизвестный тип") ||
		strings.Contains(errMsg, "превышает") ||
		strings.Contains(errMsg, "не может быть меньше") ||
		strings.Contains(errMsg, "уже существует"):
		WriteJSON(w, http.StatusBadRequest, CommonError{Error: errMsg})
	case errors.Is(err, errs.ErrNotFound) ||
		errors.Is(err, errs.ErrContractNotFound) ||
		strings.Contains(errMsg, "не найден") ||
		strings.Contains(errMsg, "не найдена") ||
		strings.Contains(errMsg, "не найдено") ||
		strings.Contains(errMsg, "не существует") ||
		strings.Contains(errMsg, "отсутствует") ||
		strings.Contains(errMsg, "not found"):
		WriteJSON(w, http.StatusNotFound, CommonError{Error: errMsg})
	case errors.Is(err, errs.ErrUnauthorized) ||
		errors.Is(err, errs.ErrInvalidCredentials) ||
		errors.Is(err, errs.ErrSessionExpired) ||
		strings.Contains(errMsg, "недействителен") ||
		strings.Contains(errMsg, "отозван") ||
		strings.Contains(errMsg, "неверный логин"):
		WriteJSON(w, http.StatusUnauthorized, CommonError{Error: errMsg})
	case errors.Is(err, errs.ErrAccessDenied) ||
		strings.Contains(errMsg, "запрещено") ||
		strings.Contains(errMsg, "нельзя удалить") ||
		strings.Contains(errMsg, "уже в корзине") ||
		strings.Contains(errMsg, "только его создатель") ||
		strings.Contains(errMsg, "только сотрудники") ||
		strings.Contains(errMsg, "нет прав"):
		WriteJSON(w, http.StatusForbidden, CommonError{Error: errMsg})
	case errors.Is(err, errs.ErrContractAlreadyExists):
		WriteJSON(w, http.StatusUnprocessableEntity, CommonError{Error: errMsg})
	default:
		WriteJSON(w, http.StatusInternalServerError, CommonError{Error: errMsg})
	}
}

