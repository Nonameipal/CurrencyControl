package document

import (
	"net/http"
	"strconv"
	"strings"

	"CurrencyControl/internal/delivery/dto"

	"github.com/gorilla/mux"
)


// @Summary Поиск для дашборда
// @Description Поиск компаний по единому полю ввода с выбором одной из трёх кнопок: name (ЧДММ), inn (ИНН), amount (Сумма).
// @Tags Dashboard
// @Security ApiKeyAuth
// @Produce json
// @Param query query string false "Строка поиска (из единого поля ввода)"
// @Param search_type query string false "Кнопка фильтра: name (ЧДММ), inn (ИНН), amount (Сумма)" Enums(name, inn, amount)
// @Param id path int true "ID филиала"
// @Success 200 {array} dto.DashboardSearchResult
// @Failure 400 {object} map[string]string "Неверные параметры"
// @Failure 401 {object} map[string]string "Не авторизован"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /api/branches/{id}/dashboard [get]
func (h *ContractHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	req := dto.DashboardSearchRequest{}
	req.Query = strings.TrimSpace(r.URL.Query().Get("query"))
	req.SearchType = strings.TrimSpace(strings.ToLower(r.URL.Query().Get("search_type")))
	branchStr := mux.Vars(r)["id"]
	branchID, err := strconv.Atoi(branchStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID филиала"})
		return
	}
	req.BranchID = branchID
	results, err := h.service.SearchDashboard(r.Context(), login, req)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, results)
}

// @Summary Уведомления дашборда (контракты с истекающим сроком)
// @Description Возвращает список контрактов, срок действия которых истекает в ближайшие 10 дней или уже истек.
// @Tags Dashboard
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Success 200 {array} dto.NotificationResponse
// @Failure 401 {object} map[string]string "Не авторизован"
// @Router /api/branches/{id}/dashboard/notifications [get]
func (h *ContractHandler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	branchID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID филиала"})
		return
	}

	notifications, err := h.service.GetExpiringContracts(r.Context(), branchID)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"notifications": notifications,
		"total":         len(notifications),
	})
}