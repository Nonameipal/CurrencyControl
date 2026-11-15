package document

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/errs"
	"CurrencyControl/internal/service/ports"
)

type InvoiceHandler struct {
	invoiceSvc ports.InvoiceService
	gtdSvc     ports.GTDService
	addlSvc    ports.AdditionalAgreementService
	poSvc      ports.PaymentOrderService
}

func NewInvoiceHandler(invoiceSvc ports.InvoiceService, gtdSvc ports.GTDService, addlSvc ports.AdditionalAgreementService, poSvc ports.PaymentOrderService) *InvoiceHandler {
	return &InvoiceHandler{invoiceSvc: invoiceSvc, gtdSvc: gtdSvc, addlSvc: addlSvc, poSvc: poSvc}
}

// @Summary Список инвойсов контракта
// @Description Возвращает список инвойсов с ГТД, товарным остатком и PDF документами.
// @Tags Invoices
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании (ЧДММ)"
// @Param contract_id path int true "ID контракта"
// @Success 200 {array} domain.InvoiceWithDetails
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices [get]
func (h *InvoiceHandler) GetInvoices(w http.ResponseWriter, r *http.Request) {
	contractID, ok := requireID(w, r, "contract_id")
	if !ok {
		return
	}
	invoices, err := h.invoiceSvc.GetByContractID(r.Context(), contractID)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, invoices)
}

// @Summary Создать инвойс
// @Description Создает инвойс к контракту
// @Tags Invoices
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании (ЧДММ)"
// @Param contract_id path int true "ID контракта"
// @Param invoice_number formData string true "Номер инвойса"
// @Param invoice_date formData string true "Дата инвойса (YYYY-MM-DD или DD.MM.YYYY)"
// @Param amount formData number true "Сумма инвойса (в валюте инвойса)"
// @Param currency formData string true "Валюта инвойса"
// @Param document formData file true "PDF файл"
// @Success 201 {object} domain.Invoice
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices [post]
func (h *InvoiceHandler) CreateInvoice(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	contractID, ok := requireID(w, r, "contract_id")
	if !ok {
		return
	}
	v := NewFormValidator(r, 10<<20)
	v.RequireStrings(invoiceRequiredFields)
	invoiceDate := v.Date("invoice_date")
	amount := v.Float("amount")

	if v.Respond(w) {
		return
	}

	var docPath *string
	if filePath, err := saveUploadedFile(r, "document", "uploads/invoices", false); err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	} else if filePath != "" {
		docPath = &filePath
	}

	inv := domain.Invoice{
		ContractID:    contractID,
		InvoiceNumber: strings.TrimSpace(r.FormValue("invoice_number")),
		InvoiceDate:   *invoiceDate,
		Amount:        amount,
		Currency:      strings.ToUpper(strings.TrimSpace(r.FormValue("currency"))),
		DocumentPath:  docPath,
		CreatedBy:     login,
	}

	created, err := h.invoiceSvc.Create(r.Context(), inv, "")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}

	LogUserAction(r, "CREATE", "invoice", &created.ID, "Создание инвойса № "+created.InvoiceNumber)

	writeJSON(w, http.StatusCreated, created)
}

// @Summary Редактирование инвойса
// @Description Редактирование инвойса. Доступно: Сотрудник валютного контроля (при наличии разрешения), Комплаенс, Администратор.
// @Tags Invoices
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param invoice_id path int true "ID инвойса"
// @Param invoice_number formData string true "Номер инвойса"
// @Param invoice_date formData string true "Дата (YYYY-MM-DD или DD.MM.YYYY)"
// @Param amount formData number true "Сумма"
// @Param currency formData string true "Валюта"
// @Param document formData file false "Новый PDF файл"
// @Success 200 {object} domain.Invoice
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string "Доступ запрещен"
// @Failure 500 {object} map[string]string
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id} [put]
func (h *InvoiceHandler) UpdateInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceID, ok := requireID(w, r, "invoice_id")
	if !ok {
		return
	}
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}
	existing, err := h.invoiceSvc.GetByID(r.Context(), invoiceID)
	if err != nil {
		handleError(w, err)
		return
	}
	diff := NewDiffTracker()
	if v := strings.TrimSpace(r.FormValue("invoice_number")); v != "" {
		diff.TrackString("Номер инвойса", existing.InvoiceNumber, v)
		existing.InvoiceNumber = v
	}
	if v := r.FormValue("invoice_date"); v != "" {
		if d := parseDate(v); d != nil {
			diff.TrackDate("Дата инвойса", existing.InvoiceDate, d)
			existing.InvoiceDate = *d
		}
	}
	if v := r.FormValue("amount"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			diff.TrackFloat("Сумма инвойса", existing.Amount, f)
			existing.Amount = f
		}
	}
	if v := strings.TrimSpace(r.FormValue("currency")); v != "" {
		cur := strings.ToUpper(v)
		diff.TrackString("Валюта инвойса", existing.Currency, cur)
		existing.Currency = cur
	}

	if pathStr, err := saveUploadedFile(r, "document", "uploads/invoices", false); err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	} else if pathStr != "" {
		diff.TrackCustom("Загружен новый файл документа")
		existing.DocumentPath = &pathStr
	}

	updated, err := h.invoiceSvc.Update(r.Context(), existing.ID, existing)
	if err != nil {
		handleError(w, err)
		return
	}

	details := fmt.Sprintf("Обновление инвойса № %s. Изменения: %s", updated.InvoiceNumber, diff.String())
	LogUserAction(r, "UPDATE", "invoice", &updated.ID, details)

	writeJSON(w, http.StatusOK, updated)
}

// @Summary Удаление инвойса (в корзину)
// @Description Помещает инвойс в корзину (soft delete). Доступно: Сотрудники валютного контроля, Комплаенс, Администратор.
// @Tags Invoices
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param invoice_id path int true "ID инвойса"
// @Success 200 {object} map[string]string "Сообщение об успешном удалении"
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id} [delete]
func (h *InvoiceHandler) DeleteInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceID, ok := requireID(w, r, "invoice_id")
	if !ok {
		return
	}

	if err := h.invoiceSvc.SoftDelete(r.Context(), invoiceID); err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "DELETE", "invoice", &invoiceID, "Удаление инвойса")

	writeJSON(w, http.StatusOK, map[string]string{"message": "Инвойс успешно удален"})
}

// @Summary Карточка инвойса (получить по ID)
// @Description Возвращает подробную информацию по карточке инвойса
// @Tags Invoices
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param invoice_id path int true "ID инвойса"
// @Success 200 {object} domain.InvoiceWithDetails
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id} [get]
func (h *InvoiceHandler) GetInvoiceByID(w http.ResponseWriter, r *http.Request) {
	invoiceID, ok := requireID(w, r, "invoice_id")
	if !ok {
		return
	}

	inv, err := h.invoiceSvc.GetByID(r.Context(), invoiceID)
	if err != nil {
		handleError(w, err)
		return
	}

	details := domain.InvoiceWithDetails{Invoice: inv}
	gtdData, _ := h.gtdSvc.GetByInvoiceID(r.Context(), invoiceID)
	if gtdData != nil {
		details.GTD = gtdData
	}
	if h.poSvc != nil {
		pos, _ := h.poSvc.GetByInvoiceID(r.Context(), invoiceID)
		details.PaymentOrders = pos
		var paid float64
		for _, p := range pos {
			paid += p.Amount
		}
		details.PaidAmount = paid
		rem := inv.Amount - paid
		if rem < 0 {
			rem = 0
		}
		details.RemainingPaymentAmount = rem
	}

	writeJSON(w, http.StatusOK, details)
}
