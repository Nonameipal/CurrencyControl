package workflow

import (
	"fmt"
	"net/http"
	"strings"

	"CurrencyControl/internal/delivery/dto"
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
	CanCreate bool   `json:"can_create"`
	CanEdit   bool   `json:"can_edit"`
	CanDelete bool   `json:"can_delete"`
}

// @Summary Поиск сотрудников валютного контроля
// @Description Поиск сотрудников с ролью Валютный контроль для назначения прав. Поиск выполняется по части логина, имени или фамилии (без учета регистра).
// @Tags Compliance
// @Security ApiKeyAuth
// @Produce json
// @Param q query string false "Строка поиска (логин, имя, фамилия)"
// @Success 200 {array} dto.CurrencyControlUserItem
// @Failure 401 {object} CommonError
// @Failure 403 {object} CommonError
// @Router /api/compliance/permissions/currency-control/users [get]
func (h *ComplianceHandler) SearchCurrencyControlUsers(w http.ResponseWriter, r *http.Request) {
	queryParam := strings.TrimSpace(r.URL.Query().Get("q"))
	users, err := h.permSvc.SearchCurrencyControlUsers(r.Context(), queryParam)
	if err != nil {
		handleError(w, err)
		return
	}
	if users == nil {
		users = []dto.CurrencyControlUserItem{}
	}
	writeJSON(w, http.StatusOK, users)
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
	perms, err := h.permSvc.GetCurrencyControlPermissions(r.Context())
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, perms)
}

// @Summary Предоставление доступа на редактирование/удаление файлов валютному контролю
// @Description Сотрудник Комплаенса предоставляет доступ сотруднику валютного контроля.
// @Tags Compliance
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param body body GrantPermissionRequest true "Параметры доступа"
// @Success 200 {object} map[string]string
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 404 {object} CommonError
// @Router /api/compliance/permissions/currency-control [post]
func (h *ComplianceHandler) GrantPermission(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())

	var req GrantPermissionRequest
	if err := decodeJSON(r, &req); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	targetLogin := strings.TrimSpace(req.Login)
	if targetLogin == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Укажите логин сотрудника валютного контроля"})
		return
	}
	perm := domain.CurrencyControlPermission{
		Login:     targetLogin,
		CanCreate: req.CanCreate,
		CanEdit:   req.CanEdit,
		CanDelete: req.CanDelete,
		GrantedBy: login,
	}
	if err := h.permSvc.GrantCurrencyControlPermission(r.Context(), perm); err != nil {
		handleError(w, err)
		return
	}
	LogUserAction(r, "GRANT_PERMISSION", "currency_control_permissions", nil,
		fmt.Sprintf("Предоставлен доступ для %s: создание=%t, редактирование=%t, удаление=%t", targetLogin, req.CanCreate, req.CanEdit, req.CanDelete))
	writeJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Доступ для %s успешно предоставлен", targetLogin),
	})
}

type UpdatePermissionRequest struct {
	CanCreate *bool `json:"can_create,omitempty"`
	CanEdit   *bool `json:"can_edit,omitempty"`
	CanDelete *bool `json:"can_delete,omitempty"`
}

// @Summary Обновление прав валютного контроля
// @Description Сотрудник Комплаенса обновляет разрешения (создание, редактирование, удаление) у конкретного сотрудника валютного контроля.
// @Tags Compliance
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param login path string true "Логин сотрудника валютного контроля"
// @Param body body UpdatePermissionRequest true "Параметры доступа"
// @Success 200 {object} map[string]string
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 404 {object} CommonError
// @Router /api/compliance/permissions/currency-control/{login} [put]
func (h *ComplianceHandler) UpdatePermission(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	targetLogin := strings.TrimSpace(mux.Vars(r)["login"])
	if targetLogin == "" || targetLogin == "{login}" || strings.HasPrefix(targetLogin, "{") {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Параметр пути login обязателен (укажите логин сотрудника валютного контроля)"})
		return
	}
	var req UpdatePermissionRequest
	if err := decodeJSON(r, &req); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}
	updatedPerm, err := h.permSvc.UpdateCurrencyControlPermission(r.Context(), targetLogin, req.CanCreate, req.CanEdit, req.CanDelete, login)
	if err != nil {
		handleError(w, err)
		return
	}
	LogUserAction(r, "UPDATE_PERMISSION", "currency_control_permissions", nil,
		fmt.Sprintf("Обновлен доступ для %s: создание=%t, редактирование=%t, удаление=%t", updatedPerm.Login, updatedPerm.CanCreate, updatedPerm.CanEdit, updatedPerm.CanDelete))
	writeJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Доступ для %s успешно обновлен", updatedPerm.Login),
	})
}

// @Summary Отзыв доступа на редактирование/удаление у сотрудника валютного контроля
// @Description Сотрудник Комплаенса отзывает разрешение у конкретного пользователя валютного контроля.
// @Tags Compliance
// @Security ApiKeyAuth
// @Produce json
// @Param login path string true "Логин сотрудника валютного контроля"
// @Success 200 {object} map[string]string
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 404 {object} CommonError
// @Router /api/compliance/permissions/currency-control/{login} [delete]
func (h *ComplianceHandler) RevokePermission(w http.ResponseWriter, r *http.Request) {
	targetLogin := strings.TrimSpace(mux.Vars(r)["login"])
	if targetLogin == "" || targetLogin == "{login}" || strings.HasPrefix(targetLogin, "{") {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Параметр пути login обязателен (укажите логин сотрудника валютного контроля)"})
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
