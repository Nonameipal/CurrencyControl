package http

import (
	"net/http"
	"strconv"
	"strings"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/errs"
	"CurrencyControl/internal/service/ports"

	"github.com/gorilla/mux"
)

type PaymentOrderHandler struct {
	svc ports.PaymentOrderService
}

func NewPaymentOrderHandler(svc ports.PaymentOrderService) *PaymentOrderHandler {
	return &PaymentOrderHandler{svc: svc}
}

// @Summary Создать платежное поручение к инвойсу
// @Description Создает новое платежное поручение, привязанное к инвойсу. .
// @Tags PaymentOrders
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param invoice_id path int true "ID инвойса"
// @Param agreement_id path int false "ID доп. соглашения (если применимо)"
// @Param operation_date formData string true "Дата операции (YYYY-MM-DD или DD.MM.YYYY)"
// @Param payment_order_number formData string true "Номер платежного поручения"
// @Param amount formData number true "Сумма платежа"
// @Param currency formData string true "Валюта платежа (должна совпадать с валютой инвойса)"
// @Param payer formData string true "Плательщик"
// @Param receiver_name formData string true "Получатель"
// @Param receiver_bank formData string true "Банк получателя"
// @Param payment_purpose formData string true "Назначение платежа"
// @Param receiver_country formData string true "Страна получателя"
// @Param contract_number formData string false "Номер контракта (заполнится автоматически при отсутствии)"
// @Param invoice_number formData string false "Номер инвойса (заполнится автоматически при отсутствии)"
// @Param value_date formData string true "Дата валютирования (YYYY-MM-DD или DD.MM.YYYY)"
// @Param document formData file false "Файл платежного поручения (PDF / Word)"
// @Success 201 {object} domain.PaymentOrder
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/payment-orders [post]
func (h *PaymentOrderHandler) CreatePaymentOrder(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())

	contractID, ok := requireID(w, r, "contract_id")
	if !ok {
		return
	}

	invoiceID, ok := requireID(w, r, "invoice_id")
	if !ok {
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	var addlID *int64
	addlStr := mux.Vars(r)["agreement_id"]
	if addlStr == "" {
		addlStr = r.FormValue("additional_agreement_id")
	}
	if addlStr == "" {
		addlStr = r.FormValue("agreement_id")
	}
	if addlStr != "" {
		if id, err := strconv.ParseInt(addlStr, 10, 64); err == nil && id > 0 {
			addlID = &id
		}
	}

	opDateStr := strings.TrimSpace(r.FormValue("operation_date"))
	if opDateStr == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле operation_date обязательно"})
		return
	}
	opDate := parseDate(opDateStr)
	if opDate == nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Неверный формат operation_date (ожидается YYYY-MM-DD или DD.MM.YYYY)"})
		return
	}

	poNumber := strings.TrimSpace(r.FormValue("payment_order_number"))
	if poNumber == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле payment_order_number обязательно"})
		return
	}

	amountStr := strings.TrimSpace(r.FormValue("amount"))
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil || amount <= 0 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле amount обязательно и должно быть числом больше 0"})
		return
	}

	currency := strings.ToUpper(strings.TrimSpace(r.FormValue("currency")))
	if currency == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле currency обязательно"})
		return
	}

	payer := strings.TrimSpace(r.FormValue("payer"))
	if payer == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле payer обязательно"})
		return
	}

	receiverName := strings.TrimSpace(r.FormValue("receiver_name"))
	if receiverName == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле receiver_name обязательно"})
		return
	}

	receiverBank := strings.TrimSpace(r.FormValue("receiver_bank"))
	if receiverBank == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле receiver_bank обязательно"})
		return
	}

	paymentPurpose := strings.TrimSpace(r.FormValue("payment_purpose"))
	if paymentPurpose == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле payment_purpose обязательно"})
		return
	}

	receiverCountry := strings.TrimSpace(r.FormValue("receiver_country"))
	if receiverCountry == "" {
		receiverCountry = strings.TrimSpace(r.FormValue("recipient_country"))
	}
	if receiverCountry == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле receiver_country обязательно"})
		return
	}

	contractNumber := strings.TrimSpace(r.FormValue("contract_number"))
	invoiceNumber := strings.TrimSpace(r.FormValue("invoice_number"))

	valDateStr := strings.TrimSpace(r.FormValue("value_date"))
	if valDateStr == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле value_date обязательно"})
		return
	}
	valDate := parseDate(valDateStr)
	if valDate == nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Неверный формат value_date (ожидается YYYY-MM-DD или DD.MM.YYYY)"})
		return
	}
	var docPath *string
	if filePath, err := saveUploadedFile(r, "document", "uploads/payment_orders", false); err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	} else if filePath != "" {
		docPath = &filePath
	}

	po := domain.PaymentOrder{
		ContractID:            contractID,
		AdditionalAgreementID: addlID,
		InvoiceID:             invoiceID,
		OperationDate:         *opDate,
		PaymentOrderNumber:    poNumber,
		Amount:                amount,
		Currency:              currency,
		Payer:                 payer,
		ReceiverName:          receiverName,
		ReceiverBank:          receiverBank,
		PaymentPurpose:        paymentPurpose,
		ReceiverCountry:       receiverCountry,
		ContractNumber:        contractNumber,
		InvoiceNumber:         invoiceNumber,
		ValueDate:             *valDate,
		CreatedBy:             login,
		DocumentPath:          docPath,
	}

	created, err := h.svc.Create(r.Context(), po)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}

	LogUserAction(r, "CREATE", "payment_order", &created.ID, "Создание платежного поручения № "+created.PaymentOrderNumber)

	writeJSON(w, http.StatusCreated, created)
}

// @Summary Список платежных поручений инвойса
// @Description Возвращает все платежные поручения, привязанные к указанному инвойсу
// @Tags PaymentOrders
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param invoice_id path int true "ID инвойса"
// @Success 200 {array} domain.PaymentOrder
// @Failure 401 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/payment-orders [get]
func (h *PaymentOrderHandler) GetPaymentOrdersByInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceID, ok := requireID(w, r, "invoice_id")
	if !ok {
		return
	}

	list, err := h.svc.GetByInvoiceID(r.Context(), invoiceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, list)
}

// @Summary Карточка платежного поручения (получить по ID)
// @Description Возвращает полную информацию по карточке платежного поручения
// @Tags PaymentOrders
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param invoice_id path int true "ID инвойса"
// @Param po_id path int true "ID платежного поручения"
// @Success 200 {object} domain.PaymentOrder
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 404 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/payment-orders/{po_id} [get]
func (h *PaymentOrderHandler) GetPaymentOrderByID(w http.ResponseWriter, r *http.Request) {
	poID, ok := requireID(w, r, "po_id")
	if !ok {
		return
	}

	po, err := h.svc.GetByID(r.Context(), poID)
	if err != nil || po == nil {
		writeJSON(w, http.StatusNotFound, CommonError{Error: "Платежное поручение не найдено"})
		return
	}

	writeJSON(w, http.StatusOK, po)
}

// @Summary Редактирование платежного поручения
// @Description Редактирование данных платежного поручения с автоматическим контролем остатка инвойса
// @Tags PaymentOrders
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param invoice_id path int true "ID инвойса"
// @Param po_id path int true "ID платежного поручения"
// @Param operation_date formData string true "Дата операции (YYYY-MM-DD или DD.MM.YYYY)"
// @Param payment_order_number formData string true "Номер платежного поручения"
// @Param amount formData number true "Сумма платежа"
// @Param currency formData string true "Валюта платежа"
// @Param payer formData string true "Плательщик"
// @Param receiver_name formData string true "Получатель"
// @Param receiver_bank formData string true "Банк получателя"
// @Param payment_purpose formData string true "Назначение платежа"
// @Param receiver_country formData string true "Страна получателя"
// @Param value_date formData string true "Дата валютирования (YYYY-MM-DD или DD.MM.YYYY)"
// @Param document formData file false "Новый файл документа (.pdf, .doc, .docx)"
// @Success 200 {object} domain.PaymentOrder
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 404 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/payment-orders/{po_id} [put]
func (h *PaymentOrderHandler) UpdatePaymentOrder(w http.ResponseWriter, r *http.Request) {
	poID, ok := requireID(w, r, "po_id")
	if !ok {
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	existing, err := h.svc.GetByID(r.Context(), poID)
	if err != nil || existing == nil {
		writeJSON(w, http.StatusNotFound, CommonError{Error: "Платежное поручение не найдено"})
		return
	}

	if v := r.FormValue("operation_date"); v != "" {
		if d := parseDate(v); d != nil {
			existing.OperationDate = *d
		}
	}
	if v := strings.TrimSpace(r.FormValue("payment_order_number")); v != "" {
		existing.PaymentOrderNumber = v
	}
	if v := r.FormValue("amount"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			existing.Amount = f
		}
	}
	if v := strings.TrimSpace(r.FormValue("currency")); v != "" {
		existing.Currency = strings.ToUpper(v)
	}
	if v := strings.TrimSpace(r.FormValue("payer")); v != "" {
		existing.Payer = v
	}
	if v := strings.TrimSpace(r.FormValue("receiver_name")); v != "" {
		existing.ReceiverName = v
	}
	if v := strings.TrimSpace(r.FormValue("receiver_bank")); v != "" {
		existing.ReceiverBank = v
	}
	if v := strings.TrimSpace(r.FormValue("payment_purpose")); v != "" {
		existing.PaymentPurpose = v
	}
	if v := strings.TrimSpace(r.FormValue("receiver_country")); v != "" {
		existing.ReceiverCountry = v
	} else if v := strings.TrimSpace(r.FormValue("recipient_country")); v != "" {
		existing.ReceiverCountry = v
	}
	if v := r.FormValue("value_date"); v != "" {
		if d := parseDate(v); d != nil {
			existing.ValueDate = *d
		}
	}

	if pathStr, err := saveUploadedFile(r, "document", "uploads/payment_orders", false); err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	} else if pathStr != "" {
		existing.DocumentPath = &pathStr
	}

	updated, err := h.svc.Update(r.Context(), existing.ID, *existing)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}

	LogUserAction(r, "UPDATE", "payment_order", &updated.ID, "Обновление платежного поручения № "+updated.PaymentOrderNumber)

	writeJSON(w, http.StatusOK, updated)
}

// @Summary Удаление платежного поручения (в корзину)
// @Description Помещает платежное поручение в корзину (soft delete) и возвращает сумму в доступный остаток инвойса
// @Tags PaymentOrders
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param invoice_id path int true "ID инвойса"
// @Param po_id path int true "ID платежного поручения"
// @Success 200 {object} map[string]string "Сообщение об успешном удалении"
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/payment-orders/{po_id} [delete]
func (h *PaymentOrderHandler) DeletePaymentOrder(w http.ResponseWriter, r *http.Request) {
	poID, ok := requireID(w, r, "po_id")
	if !ok {
		return
	}

	if err := h.svc.SoftDelete(r.Context(), poID); err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "DELETE", "payment_order", &poID, "Удаление платежного поручения")

	writeJSON(w, http.StatusOK, map[string]string{"message": "Платежное поручение успешно удалено"})
}

// @Summary Создать платежное поручение к инвойсу доп. соглашения
// @Description Создает новое платежное поручение, привязанное к инвойсу дополнительного соглашения.
// @Tags PaymentOrders
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param agreement_id path int true "ID доп. соглашения"
// @Param invoice_id path int true "ID инвойса"
// @Param operation_date formData string true "Дата операции (YYYY-MM-DD или DD.MM.YYYY)"
// @Param payment_order_number formData string true "Номер платежного поручения"
// @Param amount formData number true "Сумма платежа"
// @Param currency formData string true "Валюта платежа (должна совпадать с валютой инвойса)"
// @Param payer formData string true "Плательщик"
// @Param receiver_name formData string true "Получатель"
// @Param receiver_bank formData string true "Банк получателя"
// @Param payment_purpose formData string true "Назначение платежа"
// @Param receiver_country formData string true "Страна получателя"
// @Param contract_number formData string false "Номер контракта"
// @Param invoice_number formData string false "Номер инвойса"
// @Param value_date formData string true "Дата валютирования (YYYY-MM-DD или DD.MM.YYYY)"
// @Param document formData file false "Файл платежного поручения (PDF / Word)"
// @Success 201 {object} domain.PaymentOrder
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/invoices/{invoice_id}/payment-orders [post]
func (h *PaymentOrderHandler) CreateAdditionalAgreementPaymentOrder(w http.ResponseWriter, r *http.Request) {
	h.CreatePaymentOrder(w, r)
}
