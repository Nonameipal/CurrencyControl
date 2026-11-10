package http

import (
	"net/http"
	"strconv"

	"CurrencyControl/internal/domain"

	"github.com/gorilla/mux"
)

var _ = domain.Contract{}


// @Summary Список архивных контрактов по филиалу (с пагинацией)
// @Description Возвращает постраничный список архивных контрактов для указанного филиала
// @Tags Archive
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param page query int false "Номер страницы (по умолчанию 1)"
// @Param page_size query int false "Размер страницы (по умолчанию 20, макс 100)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/branches/{id}/archive [get]
func (h *ContractHandler) GetArchivedContracts(w http.ResponseWriter, r *http.Request) {
	branchID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID филиала"})
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}

	contracts, total, err := h.service.GetArchived(r.Context(), branchID, page, pageSize)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":      contracts,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// @Summary Список архивных контрактов по компании
// @Description Возвращает все архивные контракты для выбранной компании
// @Tags Archive
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Success 200 {array} domain.Contract
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/branches/{id}/dashboard/companies/{company_id}/archive [get]
func (h *ContractHandler) GetArchivedByCompany(w http.ResponseWriter, r *http.Request) {
	clientID, ok := requireID(w, r, "company_id")
	if !ok {
		return
	}
	contracts, err := h.service.GetArchivedByClientID(r.Context(), clientID)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, contracts)
}

// @Summary Разархивировать контракт (восстановить в active)
// @Description Переводит контракт из статуса archived обратно в active. Доступно: Валютный контроль, Комплаенс, Администратор.
// @Tags Archive
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/restore [put]
func (h *ContractHandler) RestoreContract(w http.ResponseWriter, r *http.Request) {
	contractID, ok := requireID(w, r, "contract_id")
	if !ok {
		return
	}
	if err := h.service.RestoreContract(r.Context(), contractID); err != nil {
		handleError(w, err)
		return
	}
	LogUserAction(r, "RESTORE", "contract", &contractID, "Восстановление контракта из архива")
	writeJSON(w, http.StatusOK, map[string]string{"message": "Контракт успешно восстановлен из архива"})
}

