package http

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/errs"
	"CurrencyControl/internal/service"

	"github.com/gorilla/mux"
)

type InvoiceHandler struct {
	invoiceSvc service.InvoiceService
	gtdSvc     service.GTDService
	addlSvc    service.AdditionalAgreementService
	branchSVC  service.BranchService
}

func NewInvoiceHandler(invoiceSvc service.InvoiceService, gtdSvc service.GTDService, addlSvc service.AdditionalAgreementService) *InvoiceHandler {
	return &InvoiceHandler{invoiceSvc: invoiceSvc, gtdSvc: gtdSvc, addlSvc: addlSvc}
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
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}
	contractID, err := strconv.ParseInt(mux.Vars(r)["contract_id"], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID контракта"})
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
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}
	agreementID, err := strconv.ParseInt(mux.Vars(r)["agreement_id"], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID доп. соглашения"})
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
// @Param invoice_date formData string true "Дата инвойса (YYYY-MM-DD)"
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
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}
	contractID, err := strconv.ParseInt(mux.Vars(r)["contract_id"], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID контракта"})
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
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

	invoiceNumber := strings.TrimSpace(r.FormValue("invoice_number"))
	currency := strings.ToUpper(strings.TrimSpace(r.FormValue("currency")))
	if invoiceNumber == "" || currency == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поля invoice_number и currency обязательны"})
		return
	}

	invoiceDate, err := time.Parse(time.DateOnly, r.FormValue("invoice_date"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Неверный формат invoice_date. Ожидается YYYY-MM-DD"})
		return
	}

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

	var docPath *string
	file, handler, err := r.FormFile("document")
	if err == nil {
		defer file.Close()
		ext := strings.ToLower(filepath.Ext(handler.Filename))
		if ext != ".pdf" && ext != ".doc" && ext != ".docx" {
			writeJSON(w, http.StatusBadRequest, CommonError{Error: "Разрешены только файлы форматов PDF и Word (.pdf, .doc, .docx)"})
			return
		}
		os.MkdirAll("uploads/invoices", os.ModePerm)
		uniqueFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
		filePath := filepath.Join("uploads/invoices", uniqueFileName)
		dst, err := os.Create(filePath)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, CommonError{Error: "Ошибка при сохранении файла"})
			return
		}
		defer dst.Close()
		io.Copy(dst, file)
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
// @Description Позволяет администратору обновить данные инвойса
// @Tags Admin
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param invoice_id path int true "ID инвойса"
// @Param invoice_number formData string false "Номер инвойса"
// @Param invoice_date formData string false "Дата (YYYY-MM-DD)"
// @Param amount formData number false "Сумма"
// @Param currency formData string false "Валюта"
// @Param document formData file false "Новый PDF файл"
// @Success 200 {object} domain.Invoice
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string "Доступ запрещен"
// @Failure 500 {object} map[string]string
// @Router /admin/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id} [put]
func (h *InvoiceHandler) UpdateInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceID, err := strconv.ParseInt(mux.Vars(r)["invoice_id"], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Неверный ID инвойса"})
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

	if v := strings.TrimSpace(r.FormValue("invoice_number")); v != "" { existing.InvoiceNumber = v }
	if v := r.FormValue("invoice_date"); v != "" {
		if d, err := time.Parse(time.DateOnly, v); err == nil { existing.InvoiceDate = d }
	}
	if v := r.FormValue("amount"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			existing.Amount = f
			existing.DeductAmount = f
		}
	}
	if v := r.FormValue("deduct_amount"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 { existing.DeductAmount = f }
	}
	if v := strings.TrimSpace(r.FormValue("currency")); v != "" { existing.Currency = strings.ToUpper(v) }
	if v := strings.TrimSpace(r.FormValue("hs_code")); v != "" { existing.HSCode = v }

	file, handler, err := r.FormFile("document")
	if err == nil {
		defer file.Close()
		ext := strings.ToLower(filepath.Ext(handler.Filename))
		if ext != ".pdf" && ext != ".doc" && ext != ".docx" {
			writeJSON(w, http.StatusBadRequest, CommonError{Error: "Разрешены только файлы форматов PDF и Word (.pdf, .doc, .docx)"})
			return
		}
		os.MkdirAll("uploads/invoices", os.ModePerm)
		uniqueFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
		filePath := filepath.Join("uploads/invoices", uniqueFileName)
		dst, err := os.Create(filePath)
		if err == nil {
			io.Copy(dst, file)
			dst.Close()
			pathStr := filePath
			existing.DocumentPath = &pathStr
		}
	}

	updated, err := h.invoiceSvc.Update(r.Context(), existing.ID, existing)
	if err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "UPDATE", "invoice", &updated.ID, "Обновление инвойса № "+updated.InvoiceNumber)

	writeJSON(w, http.StatusOK, updated)
}

// @Summary Удаление инвойса
// @Description Позволяет администратору удалить инвойс 
// @Tags Admin
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
// @Router /admin/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id} [delete]
func (h *InvoiceHandler) DeleteInvoice(w http.ResponseWriter, r *http.Request) {
	invoiceIDStr := mux.Vars(r)["invoice_id"]
	invoiceID, err := strconv.ParseInt(invoiceIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID инвойса"})
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
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}
	invoiceID, err := strconv.ParseInt(mux.Vars(r)["invoice_id"], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID инвойса"})
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

	writeJSON(w, http.StatusOK, details)
}

// @Summary Создать инвойс к доп. соглашению
// @Description Создает инвойс, привязанный к дополнительному соглашению. Валюта инвойса должна совпадать с валютой доп. соглашения, а сумма списывается с остатка этого соглашения.
// @Tags Invoices
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param agreement_id path int true "ID доп. соглашения"
// @Param invoice_number formData string true "Номер инвойса"
// @Param invoice_date formData string true "Дата инвойса (YYYY-MM-DD)"
// @Param amount formData number true "Сумма инвойса (в валюте соглашения)"
// @Param currency formData string true "Валюта инвойса (должна совпадать с валютой соглашения)"
// @Param hs_code formData string false "Код ТН ВЭД (HS CODE)"
// @Param document formData file true "Файл инвойса (.pdf, .doc, .docx)"
// @Success 201 {object} domain.Invoice
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/invoices [post]
func (h *InvoiceHandler) CreateAdditionalAgreementInvoice(w http.ResponseWriter, r *http.Request) {
	h.CreateInvoice(w, r)
}

// @Summary Редактирование инвойса доп. соглашения
// @Description Позволяет обновить данные инвойса, привязанного к дополнительному соглашению
// @Tags Admin
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param agreement_id path int true "ID доп. соглашения"
// @Param invoice_id path int true "ID инвойса"
// @Param invoice_number formData string false "Номер инвойса"
// @Param invoice_date formData string false "Дата (YYYY-MM-DD)"
// @Param amount formData number false "Сумма"
// @Param currency formData string false "Валюта"
// @Param hs_code formData string false "Код ТН ВЭД"
// @Param document formData file false "Новый файл (.pdf, .doc, .docx)"
// @Success 200 {object} domain.Invoice
// @Failure 400 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /admin/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/invoices/{invoice_id} [put]
func (h *InvoiceHandler) UpdateAdditionalAgreementInvoice(w http.ResponseWriter, r *http.Request) {
	h.UpdateInvoice(w, r)
}

// @Summary Удаление инвойса доп. соглашения
// @Description Позволяет администратору удалить инвойс доп. соглашения (с возвратом суммы в остаток соглашения)
// @Tags Admin
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param agreement_id path int true "ID доп. соглашения"
// @Param invoice_id path int true "ID инвойса"
// @Success 200 {object} map[string]string
// @Failure 400 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /admin/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/invoices/{invoice_id} [delete]
func (h *InvoiceHandler) DeleteAdditionalAgreementInvoice(w http.ResponseWriter, r *http.Request) {
	h.DeleteInvoice(w, r)
}
