package http

import (
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
	branchSVC  ports.BranchService
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

// @Summary Список инвойсов доп. соглашения
// @Description Возвращает список инвойсов с ГТД, привязанных к доп. соглашению.
// @Tags Invoices
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании (ЧДММ)"
// @Param contract_id path int true "ID контракта"
// @Param agreement_id path int true "ID доп. соглашения"
// @Success 200 {array} domain.InvoiceWithDetails
// @Failure 401 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/invoices [get]
func (h *InvoiceHandler) GetAdditionalAgreementInvoices(w http.ResponseWriter, r *http.Request) {
	agreementID, ok := requireID(w, r, "agreement_id")
	if !ok {
		return
	}
	invoices, err := h.invoiceSvc.GetByAdditionalAgreementID(r.Context(), agreementID)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, invoices)
}

// @Summary Создать инвойс
// @Description Создает инвойс к контракту или к доп. соглашению
// @Tags Invoices
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании (ЧДММ)"
// @Param contract_id path int true "ID контракта"
// @Param agreement_id path int false "ID доп. соглашения (опционально, если инвойс к доп. соглашению)"
// @Param invoice_number formData string true "Номер инвойса"
// @Param invoice_date formData string true "Дата инвойса (YYYY-MM-DD или DD.MM.YYYY)"
// @Param amount formData number true "Сумма инвойса (в валюте инвойса)"
// @Param currency formData string true "Валюта инвойса"
// @Param hs_code formData string true "Код ТН ВЭД (HS CODE)"
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

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	addlID := parseOptionalAgreementID(r)

	invoiceNumber := strings.TrimSpace(r.FormValue("invoice_number"))
	currency := strings.ToUpper(strings.TrimSpace(r.FormValue("currency")))
	if invoiceNumber == "" || currency == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поля invoice_number и currency обязательны"})
		return
	}

	invoiceDatePtr := parseDate(r.FormValue("invoice_date"))
	if invoiceDatePtr == nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Неверный формат invoice_date. Ожидается дата (например 02.01.2006 или 2006-01-02)"})
		return
	}
	invoiceDate := *invoiceDatePtr

	amount, err := strconv.ParseFloat(r.FormValue("amount"), 64)
	if err != nil || amount <= 0 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле amount обязательно и должно быть числом больше 0"})
		return
	}

	deductAmount := amount
	if v := r.FormValue("deduct_amount"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			deductAmount = f
		}
	}
	hsCode := strings.TrimSpace(r.FormValue("hs_code"))
	if hsCode == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле hs_code обязательно"})
		return
	}

	var docPath *string
	if filePath, err := saveUploadedFile(r, "document", "uploads/invoices", false); err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	} else if filePath != "" {
		docPath = &filePath
	}

	var contractCurrency string
	contractCurrency = ""

	inv := domain.Invoice{
		ContractID:            contractID,
		AdditionalAgreementID: addlID,
		InvoiceNumber:         invoiceNumber,
		InvoiceDate:           invoiceDate,
		Amount:                amount,
		Currency:              currency,
		HSCode:                hsCode,
		DeductAmount:          deductAmount,
		DocumentPath:          docPath,
		CreatedBy:             login,
	}

	created, err := h.invoiceSvc.Create(r.Context(), inv, contractCurrency)
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
// @Param hs_code formData string true "Код ТН ВЭД"
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

	if v := strings.TrimSpace(r.FormValue("invoice_number")); v != "" {
		existing.InvoiceNumber = v
	}
	if v := r.FormValue("invoice_date"); v != "" {
		if d := parseDate(v); d != nil {
			existing.InvoiceDate = *d
		}
	}
	if v := r.FormValue("amount"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			existing.Amount = f
			existing.DeductAmount = f
		}
	}
	if v := r.FormValue("deduct_amount"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			existing.DeductAmount = f
		}
	}
	if v := strings.TrimSpace(r.FormValue("currency")); v != "" {
		existing.Currency = strings.ToUpper(v)
	}
	if v := strings.TrimSpace(r.FormValue("hs_code")); v != "" {
		existing.HSCode = v
	}

	if pathStr, err := saveUploadedFile(r, "document", "uploads/invoices", false); err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	} else if pathStr != "" {
		existing.DocumentPath = &pathStr
	}

	updated, err := h.invoiceSvc.Update(r.Context(), existing.ID, existing)
	if err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "UPDATE", "invoice", &updated.ID, "Обновление инвойса № "+updated.InvoiceNumber)

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
