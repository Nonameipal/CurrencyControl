package http
import (
	"net/http"
	"strconv"
	"github.com/gorilla/mux"

)

// @Summary Удаление контракта (в корзину)
// @Description Удаление контракта в корзину (soft delete). Доступно: Сотрудники Валютного контроля, Комплаенс, Администратор.
// @Tags Contracts
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Success 200 {object} map[string]string "Сообщение об успешном удалении"
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string "Доступ запрещен"
// @Failure 500 {object} map[string]string
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id} [delete]
func (h *ContractHandler) Delete(w http.ResponseWriter, r *http.Request) {
	contractIDStr := mux.Vars(r)["contract_id"]
	contractID, err := strconv.ParseInt(contractIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID контракта"})
		return
	}

	if err := h.service.SoftDelete(r.Context(), contractID); err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "DELETE", "contract", &contractID, "Удаление контракта")

	writeJSON(w, http.StatusOK, map[string]string{"message": "Контракт успешно удален"})
}