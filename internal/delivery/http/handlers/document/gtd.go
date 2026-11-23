package document

import (
	"fmt"
	"net/http"
	"strings"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"
)

type GTDHandler struct {
	svc ports.GTDService
}

func NewGTDHandler(svc ports.GTDService) *GTDHandler {
	return &GTDHandler{svc: svc}
}

// @Summary Просмотр ГТД инвойса
// @Description Возвращает список ГТД, привязанных к указанному инвойсу.
// @Tags GTD
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param invoice_id path int true "ID инвойса"
// @Success 200 {array} domain.GTD
// @Failure 401 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/gtd [get]
func (h *GTDHandler) GetGTDByInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceID, ok := requireID(w, r, "invoice_id")
	if !ok {
		return
	}
	list, err := h.svc.GetListByInvoiceID(r.Context(), invoiceID)
	if err != nil {
		handleError(w, err)
		return
	}
	if list == nil {
		list = []domain.GTD{}
	}
	writeJSON(w, http.StatusOK, list)
}

// @Summary Список всех ГТД по контракту
// @Description Возвращает все ГТД, относящиеся к контракту.
// @Tags GTD
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Success 200 {array} domain.GTD
// @Failure 401 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/gtd [get]
func (h *GTDHandler) GetContractGTDs(w http.ResponseWriter, r *http.Request) {
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
		list = []domain.GTD{}
	}
	writeJSON(w, http.StatusOK, list)
}

// @Summary Карточка ГТД (получить по ID)
// @Description Возвращает полную информацию по карточке ГТД
// @Tags GTD
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param gtd_id path int true "ID ГТД"
// @Success 200 {object} domain.GTD
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 404 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/gtd/{gtd_id} [get]
func (h *GTDHandler) GetGTDByID(w http.ResponseWriter, r *http.Request) {
	gtdID, ok := requireID(w, r, "gtd_id")
	if !ok {
		return
	}
	gtd, err := h.svc.GetByID(r.Context(), gtdID)
	if err != nil {
		handleError(w, err)
		return
	}
	if gtd == nil {
		writeJSON(w, http.StatusNotFound, CommonError{Error: "ГТД не найдена"})
		return
	}
	writeJSON(w, http.StatusOK, gtd)
}

// @Summary Добавить ГТД (Карточка ГТД)
// @Description Создает новую ГТД с привязкой к инвойсу. Валюта наследуется из инвойса. Списывает доступный остаток инвойса.
// @Tags GTD
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param invoice_id path int true "ID инвойса"
// @Param gtd_number formData string true "Номер ГТД"
// @Param gtd_date formData string true "Дата ГТД (YYYY-MM-DD или DD.MM.YYYY)"
// @Param gtd_amount formData number true "Сумма ГТД"
// @Param document formData file true "Файл документа ГТД (.pdf)"
// @Success 201 {object} domain.GTD
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/gtd [post]
func (h *GTDHandler) CreateGTD(w http.ResponseWriter, r *http.Request) {
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
	v.RequireStrings(gtdRequiredFields)
	gtdNumber := v.String("gtd_number")
	gtdDate := v.Date("gtd_date")
	gtdAmount := v.Float("gtd_amount")

	if v.Respond(w) {
		return
	}

	var docPath *string
	if savedPath, saveErr := saveUploadedFile(r, "document", "uploads/gtd", true); saveErr != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: saveErr.Error()})
		return
	} else if savedPath != "" {
		docPath = &savedPath
	}

	var gDate domain.GTD
	gDate = domain.GTD{
		ContractID:   contractID,
		InvoiceID:    invoiceID,
		GTDNumber:    gtdNumber,
		GTDDate:      *gtdDate,
		GTDAmount:    gtdAmount,
		DocumentPath: docPath,
		CreatedBy:    login,
	}

	created, err := h.svc.Create(r.Context(), gDate)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}

	LogUserAction(r, "CREATE", "gtd", &created.ID, fmt.Sprintf("Создание ГТД № %s", created.GTDNumber))
	writeJSON(w, http.StatusCreated, created)
}

// @Summary Редактирование ГТД
// @Description Редактирование ГТД. Доступно: Сотрудник валютного контроля (при наличии разрешения), Комплаенс, Администратор.
// @Tags GTD
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param gtd_id path int true "ID ГТД"
// @Param gtd_number formData string false "Номер ГТД"
// @Param gtd_date formData string false "Дата ГТД (YYYY-MM-DD или DD.MM.YYYY)"
// @Param gtd_amount formData number false "Сумма ГТД"
// @Param document formData file false "Новый файл ГТД (.pdf)"
// @Success 200 {object} domain.GTD
// @Failure 400 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 404 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/gtd/{gtd_id} [put]
func (h *GTDHandler) UpdateGTD(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	gtdID, ok := requireID(w, r, "gtd_id")
	if !ok {
		return
	}
	existing, err := h.svc.GetByID(r.Context(), gtdID)
	if err != nil {
		handleError(w, err)
		return
	}
	if existing == nil {
		writeJSON(w, http.StatusNotFound, CommonError{Error: "ГТД не найдена"})
		return
	}
	diff := NewDiffTracker()
	v := NewFormValidator(r, 32<<20)
	if num := strings.TrimSpace(r.FormValue("gtd_number")); num != "" {
		diff.TrackString("Номер ГТД", existing.GTDNumber, num)
		existing.GTDNumber = num
	}
	if amtStr := strings.TrimSpace(r.FormValue("gtd_amount")); amtStr != "" {
		amt := v.Float("gtd_amount")
		if amt > 0 {
			diff.TrackFloat("Сумма ГТД", existing.GTDAmount, amt)
			existing.GTDAmount = amt
		}
	}
	if dateStr := strings.TrimSpace(r.FormValue("gtd_date")); dateStr != "" {
		d := v.Date("gtd_date")
		if d != nil {
			diff.TrackDate("Дата ГТД", existing.GTDDate, d)
			existing.GTDDate = *d
		}
	}
	if v.Respond(w) {
		return
	}
	if savedPath, saveErr := saveUploadedFile(r, "document", "uploads/gtd", false); saveErr != nil {
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
	updated, err := h.svc.Update(r.Context(), gtdID, *existing)
	if err != nil {
		handleError(w, err)
		return
	}
	details := fmt.Sprintf("Редактирование ГТД № %s. Изменения: %s", updated.GTDNumber, diff.String())
	LogUserAction(r, "UPDATE", "gtd", &updated.ID, details)
	writeJSON(w, http.StatusOK, updated)
}

// @Summary Удаление ГТД (в корзину)
// @Description Помещает ГТД в корзину (soft delete). Доступно: Сотрудники валютного контроля, Комплаенс, Администратор.
// @Tags GTD
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param gtd_id path int true "ID ГТД"
// @Success 200 {object} map[string]string "Сообщение об успешном удалении"
// @Failure 400 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/gtd/{gtd_id} [delete]
func (h *GTDHandler) DeleteGTD(w http.ResponseWriter, r *http.Request) {
	gtdID, ok := requireID(w, r, "gtd_id")
	if !ok {
		return
	}
	if err := h.svc.SoftDelete(r.Context(), gtdID); err != nil {
		handleError(w, err)
		return
	}
	LogUserAction(r, "DELETE", "gtd", &gtdID, "Удаление ГТД")
	writeJSON(w, http.StatusOK, map[string]string{"message": "ГТД успешно удалена"})
}
