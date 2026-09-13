package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/errs"
	"CurrencyControl/internal/service/ports"

	"github.com/gorilla/mux"
)

type ComplianceHandler struct {
	permSvc ports.PermissionService
}

func NewComplianceHandler(permSvc ports.PermissionService) *ComplianceHandler {
	return &ComplianceHandler{permSvc: permSvc}
}

type GrantPermissionRequest struct {
	Login     string `json:"login"` 
	CanEdit   bool   `json:"can_edit"`
	CanDelete bool   `json:"can_delete"`
}

// @Summary Просмотр прав валютного контроля
// @Description Доступно только сотрудникам Комплаенс и Администраторам. Возвращает список выданных разрешений на редактирование и удаление файлов.
// @Tags Compliance
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {array} domain.CurrencyControlPermission
// @Failure 401 {object} CommonError
// @Failure 403 {object} CommonError
// @Router /api/compliance/permissions/currency-control [get]
func (h *ComplianceHandler) GetPermissions(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}

	perms, err := h.permSvc.GetCurrencyControlPermissions(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, perms)
}

// @Summary Предоставление доступа на редактирование/удаление файлов валютному контролю
// @Description Сотрудник Комплаенса предоставляет доступ сотруднику (или всем) валютного контроля.
// @Tags Compliance
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param body body GrantPermissionRequest true "Параметры доступа"
// @Success 200 {object} map[string]string
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 403 {object} CommonError
// @Router /api/compliance/permissions/currency-control [post]
func (h *ComplianceHandler) GrantPermission(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}

	var req GrantPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	targetLogin := strings.TrimSpace(req.Login)
	if targetLogin == "" {
		targetLogin = "*"
	}

	perm := domain.CurrencyControlPermission{
		Login:     targetLogin,
		CanEdit:   req.CanEdit,
		CanDelete: req.CanDelete,
		GrantedBy: login,
	}

	if err := h.permSvc.GrantCurrencyControlPermission(r.Context(), perm); err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "GRANT_PERMISSION", "currency_control_permissions", nil,
		fmt.Sprintf("Предоставлен доступ для %s: редактирование=%t, удаление=%t", targetLogin, req.CanEdit, req.CanDelete))

	writeJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Доступ для %s успешно предоставлен", targetLogin),
	})
}

// @Summary Отзыв доступа на редактирование/удаление у сотрудника валютного контроля
// @Description Сотрудник Комплаенса отзывает разрешение у конкретного пользователя.
// @Tags Compliance
// @Security ApiKeyAuth
// @Produce json
// @Param login path string true "Логин сотрудника или *"
// @Success 200 {object} map[string]string
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 403 {object} CommonError
// @Router /api/compliance/permissions/currency-control/{login} [delete]
func (h *ComplianceHandler) RevokePermission(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}

	targetLogin := strings.TrimSpace(mux.Vars(r)["login"])
	if targetLogin == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Логин не указан"})
		return
	}

	if err := h.permSvc.RevokeCurrencyControlPermission(r.Context(), targetLogin); err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "REVOKE_PERMISSION", "currency_control_permissions", nil,
		fmt.Sprintf("Отозван доступ для %s", targetLogin))

	writeJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Доступ для %s успешно отозван", targetLogin),
	})
}
