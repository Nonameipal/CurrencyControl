package http

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/errs"
	"CurrencyControl/internal/service"

	"github.com/gorilla/mux"
)

type InvoiceHandler struct {
	invoiceSvc service.InvoiceService
	gtdSvc     service.GTDService
	docRepo    documentSaver
}

type documentSaver interface {
	SaveDocument(ctx interface{ Deadline() (interface{}, bool); Done() <-chan struct{}; Err() error; Value(interface{}) interface{} }, doc domain.Document) (domain.Document, error)
}

func NewInvoiceHandler(invoiceSvc service.InvoiceService, gtdSvc service.GTDService) *InvoiceHandler {
	return &InvoiceHandler{invoiceSvc: invoiceSvc, gtdSvc: gtdSvc}
}

// @Summary Список инвойсов контракта
// @Description Возвращает список инвойсов для выбранного контракта с ГТД и документами.
// @Tags Invoices
// @Produce json
// @Param Login header string true "Логин пользователя"
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании (ЧДММ)"
// @Param contract_id path int true "ID контракта"
// @Success 200 {array} domain.InvoiceWithDetails
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices [get]
func (h *InvoiceHandler) GetInvoices(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}

	contractStr := mux.Vars(r)["contract_id"]
	contractID, err := strconv.ParseInt(contractStr, 10, 64)
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
// @Description Создает инвойс для контракта и загружает PDF документ.
// @Tags Invoices
// @Accept multipart/form-data
// @Produce json
// @Param Login header string true "Логин пользователя"
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании (ЧДММ)"
// @Param contract_id path int true "ID контракта"
// @Param invoice_number formData string true "Номер инвойса"
// @Param invoice_name formData string true "Название инвойса"
// @Param invoice_date formData string true "Дата инвойса (YYYY-MM-DD)"
// @Param amount formData number true "Сумма инвойса"
// @Param currency formData string true "Валюта инвойса"
// @Param document formData file true "PDF файл"
// @Success 201 {object} domain.Invoice
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices [post]
func (h *InvoiceHandler) CreateInvoice(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}

	contractStr := mux.Vars(r)["contract_id"]
	contractID, err := strconv.ParseInt(contractStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID контракта"})
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	invoiceNumber := r.FormValue("invoice_number")
	invoiceName := r.FormValue("invoice_name")
	currency := r.FormValue("currency")

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

	file, handler, err := r.FormFile("document")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "PDF файл обязателен"})
		return
	}
	defer file.Close()

	inv := domain.Invoice{
		ContractID:    contractID,
		InvoiceNumber: invoiceNumber,
		InvoiceName:   invoiceName,
		InvoiceDate:   invoiceDate,
		Amount:        amount,
		Currency:      currency,
	}

	created, err := h.invoiceSvc.Create(r.Context(), inv)
	if err != nil {
		handleError(w, err)
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

	writeJSON(w, http.StatusCreated, created)
}

type createGTDRequest struct {
	GTDNumber string  `json:"gtd_number"`
	GTDAmount float64 `json:"gtd_amount"`
	GTDDate   string  `json:"gtd_date"`
}

// @Summary Добавить ГТД к инвойсу
// @Description Добавляет ГТД с файлом к указанному инвойсу.
// @Tags Invoices
// @Accept multipart/form-data
// @Produce json
// @Param Login header string true "Логин пользователя"
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании (ЧДММ)"
// @Param contract_id path int true "ID контракта"
// @Param invoice_id path int true "ID инвойса"
// @Param gtd_number formData string true "Номер ГТД"
// @Param gtd_amount formData number true "Сумма ГТД"
// @Param gtd_date formData string true "Дата ГТД (YYYY-MM-DD)"
// @Param document formData file true "PDF файл ГТД"
// @Success 201 {object} domain.GTD
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/gtd [post]
func (h *InvoiceHandler) CreateGTD(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}

	invoiceStr := mux.Vars(r)["invoice_id"]
	invoiceID, err := strconv.ParseInt(invoiceStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID инвойса"})
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	gtdNumber := r.FormValue("gtd_number")
	gtdDateStr := r.FormValue("gtd_date")

	if gtdNumber == "" || gtdDateStr == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поля gtd_number и gtd_date обязательны"})
		return
	}

	gtdDate, err := time.Parse(time.DateOnly, gtdDateStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Неверный формат gtd_date. Ожидается YYYY-MM-DD"})
		return
	}

	gtdAmount, err := strconv.ParseFloat(r.FormValue("gtd_amount"), 64)
	if err != nil || gtdAmount <= 0 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле gtd_amount обязательно и должно быть числом больше 0"})
		return
	}

	file, handler, err := r.FormFile("document")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "PDF файл ГТД обязателен"})
		return
	}
	defer file.Close()

	g := domain.GTD{
		InvoiceID: invoiceID,
		GTDNumber: gtdNumber,
		GTDAmount: gtdAmount,
		GTDDate:   &gtdDate,
	}

	created, err := h.gtdSvc.Create(r.Context(), g)
	if err != nil {
		handleError(w, err)
		return
	}

	os.MkdirAll("uploads/gtd", os.ModePerm)
	uniqueFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
	filePath := filepath.Join("uploads/gtd", uniqueFileName)

	dst, err := os.Create(filePath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: "Ошибка при сохранении файла ГТД"})
		return
	}
	defer dst.Close()
	io.Copy(dst, file)

	_ = json.NewEncoder(w)
	writeJSON(w, http.StatusCreated, created)
}