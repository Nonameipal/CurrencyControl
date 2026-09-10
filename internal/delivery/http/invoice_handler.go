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

// @Summary Создать инвойс
// @Description Создает инвойс. 
// @Tags Invoices
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании (ЧДММ)"
// @Param contract_id path int true "ID контракта"
// @Param invoice_number formData string true "Номер инвойса"
// @Param invoice_name formData string true "Название инвойса"
// @Param invoice_date formData string true "Дата инвойса (YYYY-MM-DD)"
// @Param amount formData number true "Сумма инвойса (в валюте инвойса)"
// @Param currency formData string true "Валюта инвойса"
// @Param deduct_amount formData number false "Сколько списать с баланса контракта (обязательно, если валюты разные)"
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

	invoiceNumber := strings.TrimSpace(r.FormValue("invoice_number"))
	invoiceName := strings.TrimSpace(r.FormValue("invoice_name"))
	currency := strings.ToUpper(strings.TrimSpace(r.FormValue("currency")))
	if invoiceNumber == "" || invoiceName == "" || currency == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поля invoice_number, invoice_name и currency обязательны"})
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

	deductAmount, _ := strconv.ParseFloat(r.FormValue("deduct_amount"), 64)

	file, handler, err := r.FormFile("document")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "файл обязателен"})
		return
	}
	defer file.Close()

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

	var contractCurrency string
	contractCurrency = ""

	inv := domain.Invoice{
		ContractID:           contractID,
		InvoiceNumber:        invoiceNumber,
		InvoiceName:          invoiceName,
		InvoiceDate:          invoiceDate,
		Amount:               amount,
		Currency:             currency,
		DeductAmount:         deductAmount,
		DocumentPath:         &filePath,
		OriginalDocumentName: &handler.Filename,
		CreatedBy:            login,
	}

	created, err := h.invoiceSvc.Create(r.Context(), inv, contractCurrency)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}

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
// @Param invoice_name formData string false "Наименование"
// @Param invoice_date formData string false "Дата (YYYY-MM-DD)"
// @Param amount formData number false "Сумма"
// @Param deduct_amount formData number false "Сумма списания"
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
	if v := strings.TrimSpace(r.FormValue("invoice_name")); v != "" { existing.InvoiceName = v }
	if v := r.FormValue("invoice_date"); v != "" {
		if d, err := time.Parse(time.DateOnly, v); err == nil { existing.InvoiceDate = d }
	}
	if v := r.FormValue("amount"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil { existing.Amount = f }
	}
	if v := r.FormValue("deduct_amount"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil { existing.DeductAmount = f }
	}
	if v := strings.TrimSpace(r.FormValue("currency")); v != "" { existing.Currency = strings.ToUpper(v) }

	file, handler, err := r.FormFile("document")
	if err == nil {
		defer file.Close()
		os.MkdirAll("uploads/invoices", os.ModePerm)
		uniqueFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
		filePath := filepath.Join("uploads/invoices", uniqueFileName)
		dst, err := os.Create(filePath)
		if err == nil {
			io.Copy(dst, file)
			dst.Close()
			pathStr := filePath
			nameStr := handler.Filename
			existing.DocumentPath = &pathStr
			existing.OriginalDocumentName = &nameStr
		}
	}

	updated, err := h.invoiceSvc.Update(r.Context(), existing.ID, existing)
	if err != nil {
		handleError(w, err)
		return
	}
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
	writeJSON(w, http.StatusOK, map[string]string{"message": "Инвойс успешно удален"})
}

