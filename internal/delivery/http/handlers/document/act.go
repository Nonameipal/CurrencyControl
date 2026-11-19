package document

import (
	"fmt"
	"net/http"
	"strings"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"
)

type ActHandler struct {
	svc ports.ActService
}

func NewActHandler(svc ports.ActService) *ActHandler {
	return &ActHandler{svc: svc}
}

// @Summary Просмотр Актов инвойса
// @Description Возвращает список Актов выполненных работ, привязанных к указанному инвойсу.
// @Tags Acts
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param invoice_id path int true "ID инвойса"
// @Success 200 {array} domain.Act
// @Failure 401 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/acts [get]
func (h *ActHandler) GetActsByInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceID, ok := requireID(w, r, "invoice_id")
	if !ok {
		return
	}
	list, err := h.svc.GetByInvoiceID(r.Context(), invoiceID)
	if err != nil {
		handleError(w, err)
		return
	}
	if list == nil {
		list = []domain.Act{}
	}
	writeJSON(w, http.StatusOK, list)
}

// @Summary Список всех Актов по контракту
// @Description Возвращает все Акты выполненных работ, относящиеся к контракту.
// @Tags Acts
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Success 200 {array} domain.Act
// @Failure 401 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/acts [get]
func (h *ActHandler) GetContractActs(w http.ResponseWriter, r *http.Request) {
	contractID, ok := requireID(w, r, "contract_id")
	if !ok {
		return
	}
	list, err := h.svc.GetByContractID(r.Context(), contractID)
	if err != nil {
		handleError(w, err)
		return
	}
	if list == nil {
		list = []domain.Act{}
	}
	writeJSON(w, http.StatusOK, list)
}

// @Summary Карточка Акта (получить по ID)
// @Description Возвращает полную информацию по карточке Акта выполненных работ
// @Tags Acts
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param act_id path int true "ID Акта"
// @Success 200 {object} domain.Act
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 404 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/acts/{act_id} [get]
func (h *ActHandler) GetActByID(w http.ResponseWriter, r *http.Request) {
	actID, ok := requireID(w, r, "act_id")
	if !ok {
		return
	}
	act, err := h.svc.GetByID(r.Context(), actID)
	if err != nil {
		handleError(w, err)
		return
	}
	if act == nil {
		writeJSON(w, http.StatusNotFound, CommonError{Error: "Акт не найден"})
		return
	}
	writeJSON(w, http.StatusOK, act)
}

// @Summary Добавить Акт выполненных работ (Карточка Акта)
// @Description Создает новый Акт выполненных работ с привязкой к инвойсу. Валюта наследуется из инвойса.
// @Tags Acts
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param invoice_id path int true "ID инвойса"
// @Param act_number formData string true "Номер Акта"
// @Param act_date formData string true "Дата Акта (YYYY-MM-DD или DD.MM.YYYY)"
// @Param act_amount formData number true "Сумма Акта"
// @Param document formData file true "Файл документа Акта (.pdf)"
// @Success 201 {object} domain.Act
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/acts [post]
func (h *ActHandler) CreateAct(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	contractID, ok := requireID(w, r, "contract_id")
	if !ok {
		return
	}
	invoiceID, ok := requireID(w, r, "invoice_id")
	if !ok {
		return
	}

	v := NewFormValidator(r, 32<<20)
	v.RequireStrings(actRequiredFields)
	actNumber := v.String("act_number")
	actDate := v.Date("act_date")
	actAmount := v.Float("act_amount")

	if v.Respond(w) {
		return
	}

	var docPath *string
	if savedPath, saveErr := saveUploadedFile(r, "document", "uploads/acts", true); saveErr != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: saveErr.Error()})
		return
	} else if savedPath != "" {
		docPath = &savedPath
	}

	act := domain.Act{
		ContractID:   contractID,
		InvoiceID:    invoiceID,
		ActNumber:    actNumber,
		ActDate:      *actDate,
		ActAmount:    actAmount,
		DocumentPath: docPath,
		CreatedBy:    login,
	}

	created, err := h.svc.Create(r.Context(), act)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}

	LogUserAction(r, "CREATE", "act", &created.ID, fmt.Sprintf("Создание Акта выполненных работ № %s", created.ActNumber))
	writeJSON(w, http.StatusCreated, created)
}

// @Summary Редактирование Акта
// @Description Редактирование Акта выполненных работ. Доступно: Сотрудник валютного контроля (при наличии разрешения), Комплаенс, Администратор.
// @Tags Acts
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param act_id path int true "ID Акта"
// @Param act_number formData string false "Номер Акта"
// @Param act_date formData string false "Дата Акта (YYYY-MM-DD или DD.MM.YYYY)"
// @Param act_amount formData number false "Сумма Акта"
// @Param document formData file false "Новый файл Акта (.pdf)"
// @Success 200 {object} domain.Act
// @Failure 400 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 404 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/acts/{act_id} [put]
func (h *ActHandler) UpdateAct(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	actID, ok := requireID(w, r, "act_id")
	if !ok {
		return
	}

	existing, err := h.svc.GetByID(r.Context(), actID)
	if err != nil {
		handleError(w, err)
		return
	}
	if existing == nil {
		writeJSON(w, http.StatusNotFound, CommonError{Error: "Акт не найден"})
		return
	}

	diff := NewDiffTracker()
	v := NewFormValidator(r, 32<<20)
	if num := strings.TrimSpace(r.FormValue("act_number")); num != "" {
		diff.TrackString("Номер Акта", existing.ActNumber, num)
		existing.ActNumber = num
	}
	if amtStr := strings.TrimSpace(r.FormValue("act_amount")); amtStr != "" {
		amt := v.Float("act_amount")
		if amt > 0 {
			diff.TrackFloat("Сумма Акта", existing.ActAmount, amt)
			existing.ActAmount = amt
		}
	}
	if dateStr := strings.TrimSpace(r.FormValue("act_date")); dateStr != "" {
		d := v.Date("act_date")
		if d != nil {
			diff.TrackDate("Дата Акта", existing.ActDate, d)
			existing.ActDate = *d
		}
	}

	if v.Respond(w) {
		return
	}
	if savedPath, saveErr := saveUploadedFile(r, "document", "uploads/acts", false); saveErr != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: saveErr.Error()})
		return
	} else if savedPath != "" {
		diff.TrackCustom("Загружен новый файл документа")
		existing.DocumentPath = &savedPath
	} else if docPathStr := strings.TrimSpace(r.FormValue("document_path")); docPathStr != "" {
		diff.TrackCustom("Изменен путь к документу")
		existing.DocumentPath = &docPathStr
	}

	existing.UpdatedBy = login
	updated, err := h.svc.Update(r.Context(), actID, *existing)
	if err != nil {
		handleError(w, err)
		return
	}

	details := fmt.Sprintf("Редактирование Акта № %s. Изменения: %s", updated.ActNumber, diff.String())
	LogUserAction(r, "UPDATE", "act", &updated.ID, details)
	writeJSON(w, http.StatusOK, updated)
}

// @Summary Удаление Акта (в корзину)
// @Description Помещает Акт выполненных работ в корзину (soft delete). Доступно: Сотрудники валютного контроля, Комплаенс, Администратор.
// @Tags Acts
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param act_id path int true "ID Акта"
// @Success 200 {object} map[string]string "Сообщение об успешном удалении"
// @Failure 400 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/acts/{act_id} [delete]
func (h *ActHandler) DeleteAct(w http.ResponseWriter, r *http.Request) {
	actID, ok := requireID(w, r, "act_id")
	if !ok {
		return
	}
	if err := h.svc.SoftDelete(r.Context(), actID); err != nil {
		handleError(w, err)
		return
	}
	LogUserAction(r, "DELETE", "act", &actID, "Удаление Акта выполненных работ")
	writeJSON(w, http.StatusOK, map[string]string{"message": "Акт успешно удален"})
}
