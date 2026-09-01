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
}

func NewInvoiceHandler(invoiceSvc service.InvoiceService, gtdSvc service.GTDService, addlSvc service.AdditionalAgreementService) *InvoiceHandler {
	return &InvoiceHandler{invoiceSvc: invoiceSvc, gtdSvc: gtdSvc, addlSvc: addlSvc}
}

// @Summary Список инвойсов контракта
// @Description Возвращает список инвойсов с ГТД, товарным остатком и PDF документами.
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
// @Accept multipart/form-data
// @Produce json
// @Param Login header string true "Логин пользователя"
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
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "PDF файл обязателен"})
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
// @Accept multipart/form-data
// @Produce json
// @Param Login header string true "Логин администратора (admin)"
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
// @Failure 400 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse "Доступ запрещен"
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id} [put]
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

// @Summary Просмотр ГТД инвойса
// @Description Возвращает ГТД, привязанную к указанному инвойсу.
// @Tags Invoices
// @Accept json
// @Produce json
// @Param Login header string true "Логин пользователя"
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param invoice_id path int true "ID инвойса"
// @Success 200 {object} domain.GTD
// @Failure 401 {object} dto.ErrorResponse
// @Failure 404 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/gtd [get]
func (h *InvoiceHandler) GetGTD(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}
	invoiceID, err := strconv.ParseInt(mux.Vars(r)["invoice_id"], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Неверный ID инвойса"})
		return
	}

	gtd, err := h.gtdSvc.GetByInvoiceID(r.Context(), invoiceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: err.Error()})
		return
	}
	if gtd == nil {
		writeJSON(w, http.StatusNotFound, CommonError{Error: "ГТД не найдена"})
		return
	}

	writeJSON(w, http.StatusOK, gtd)
}

// @Summary Добавить ГТД к инвойсу
// @Description Добавляет ГТД с файлом к инвойсу.
// @Tags Invoices
// @Accept multipart/form-data
// @Produce json
// @Param Login header string true "Логин пользователя"
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param invoice_id path int true "ID инвойса"
// @Param gtd_number formData string true "Номер ГТД"
// @Param gtd_amount formData number true "Сумма ГТД (в валюте ГТД)"
// @Param gtd_currency formData string true "Валюта ГТД (например EUR)"
// @Param gtd_date formData string true "Дата ГТД (YYYY-MM-DD)"
// @Param closes_amount formData number true "Сколько закрывается по инвойсу (в валюте инвойса)"
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
	invoiceID, err := strconv.ParseInt(mux.Vars(r)["invoice_id"], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID инвойса"})
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

	gtdNumber := strings.TrimSpace(r.FormValue("gtd_number"))
	gtdDateStr := r.FormValue("gtd_date")
	gtdCurrencyStr := strings.ToUpper(strings.TrimSpace(r.FormValue("gtd_currency")))

	if gtdNumber == "" || gtdDateStr == "" || gtdCurrencyStr == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поля gtd_number, gtd_date и gtd_currency обязательны"})
		return
	}

	gtdDate, err := time.Parse(time.DateOnly, gtdDateStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Неверный формат gtd_date. Ожидается YYYY-MM-DD"})
		return
	}

	gtdAmount, err := strconv.ParseFloat(r.FormValue("gtd_amount"), 64)
	if err != nil || gtdAmount <= 0 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле gtd_amount обязательно и должно быть больше 0"})
		return
	}

	closesAmount, err := strconv.ParseFloat(r.FormValue("closes_amount"), 64)
	if err != nil || closesAmount <= 0 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле closes_amount обязательно и должно быть больше 0"})
		return
	}

	file, handler, err := r.FormFile("document")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "PDF файл ГТД обязателен"})
		return
	}
	defer file.Close()

	os.MkdirAll("uploads/gtd", os.ModePerm)
	uniqueFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
	filePath := filepath.Join("uploads/gtd", uniqueFileName)
	dst, _ := os.Create(filePath)
	if dst != nil {
		defer dst.Close()
		io.Copy(dst, file)
	}

	g := domain.GTD{
		ContractID:           contractID,
		InvoiceID:            invoiceID,
		GTDNumber:            gtdNumber,
		GTDAmount:            gtdAmount,
		GTDCurrency:          &gtdCurrencyStr,
		GTDDate:              &gtdDate,
		ClosesAmount:         closesAmount,
		DocumentPath:         &filePath,
		OriginalDocumentName: &handler.Filename,
		CreatedBy:            login,
	}

	created, err := h.gtdSvc.Create(r.Context(), g)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

// @Summary Редактирование ГТД
// @Description Позволяет администратору обновить данные ГТД
// @Tags Admin
// @Accept multipart/form-data
// @Produce json
// @Param Login header string true "Логин администратора (admin)"
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param invoice_id path int true "ID инвойса"
// @Param gtd_id path int true "ID ГТД"
// @Param gtd_number formData string false "Номер ГТД"
// @Param gtd_amount formData number false "Сумма ГТД"
// @Param gtd_currency formData string false "Валюта ГТД"
// @Param gtd_date formData string false "Дата ГТД (YYYY-MM-DD)"
// @Param closes_amount formData number false "Сколько закрывается по инвойсу"
// @Param document formData file false "Новый PDF файл ГТД"
// @Success 200 {object} domain.GTD
// @Failure 400 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse "Доступ запрещен"
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/gtd/{gtd_id} [put]
func (h *InvoiceHandler) UpdateGTD(w http.ResponseWriter, r *http.Request) {
	invoiceID, err := strconv.ParseInt(mux.Vars(r)["invoice_id"], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Неверный ID инвойса"})
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	existing, err := h.gtdSvc.GetByInvoiceID(r.Context(), invoiceID)
	if err != nil || existing == nil {
		if err == nil {
			err = fmt.Errorf("ГТД не найдена")
		}
		handleError(w, err)
		return
	}

	if v := strings.TrimSpace(r.FormValue("gtd_number")); v != "" { existing.GTDNumber = v }
	if v := r.FormValue("gtd_date"); v != "" {
		if d, err := time.Parse(time.DateOnly, v); err == nil { existing.GTDDate = &d }
	}
	if v := r.FormValue("gtd_amount"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil { existing.GTDAmount = f }
	}
	if v := r.FormValue("closes_amount"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil { existing.ClosesAmount = f }
	}
	if v := strings.TrimSpace(r.FormValue("gtd_currency")); v != "" {
		upper := strings.ToUpper(v)
		existing.GTDCurrency = &upper
	}

	file, handler, err := r.FormFile("document")
	if err == nil {
		defer file.Close()
		os.MkdirAll("uploads/gtd", os.ModePerm)
		uniqueFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
		filePath := filepath.Join("uploads/gtd", uniqueFileName)
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

	updated, err := h.gtdSvc.Update(r.Context(), existing.ID, *existing)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// @Summary Список доп. соглашений контракта
// @Description Возвращает список дополнительных соглашений по контракту.
// @Tags AdditionalAgreements
// @Produce json
// @Param Login header string true "Логин пользователя"
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Success 200 {array} domain.AdditionalAgreement
// @Failure 401 {object} dto.ErrorResponse
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements [get]
func (h *InvoiceHandler) GetAdditionalAgreements(w http.ResponseWriter, r *http.Request) {
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
	list, err := h.addlSvc.GetByContractID(r.Context(), contractID)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// @Summary Создать доп. соглашение
// @Description Создает доп. соглашение к контракту.
// @Tags AdditionalAgreements
// @Accept multipart/form-data
// @Produce json
// @Param Login header string true "Логин пользователя"
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param document formData file true "PDF файл доп. соглашения"
// @Param delivery_conditions formData string false "Новые условия доставки"
// @Param delivery_term_days formData int false "Новый срок доставки (дней)"
// @Param return_term_days formData int false "Новый срок возврата (дней)"
// @Param subject formData string false "Предмет соглашения"
// @Param extend_date_to formData string false "Продлить срок контракта до (YYYY-MM-DD)"
// @Param foreign_amount formData number false "Сумма в иностранной валюте"
// @Param foreign_currency formData string false "Валюта иностранной суммы (например USD)"
// @Param amount_in_contract_currency formData number false "Сумма в валюте контракта (прибавляется к лимиту)"
// @Success 201 {object} domain.AdditionalAgreement
// @Failure 400 {object} dto.ErrorResponse
// @Failure 401 {object} dto.ErrorResponse
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements [post]
func (h *InvoiceHandler) CreateAdditionalAgreement(w http.ResponseWriter, r *http.Request) {
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

	file, handler, err := r.FormFile("document")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "PDF файл доп. соглашения обязателен"})
		return
	}
	defer file.Close()

	ag := domain.AdditionalAgreement{
		ContractID: contractID,
		CreatedBy:  login,
	}


	os.MkdirAll("uploads/additional_agreements", os.ModePerm)
	uniqueFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
	filePath := filepath.Join("uploads/additional_agreements", uniqueFileName)
	dst, err := os.Create(filePath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: "Ошибка при сохранении файла"})
		return
	}
	defer dst.Close()
	io.Copy(dst, file)

	ag.DocumentPath = filePath
	ag.OriginalDocumentName = handler.Filename

	created, err := h.addlSvc.Create(r.Context(), ag)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

// @Summary Редактирование доп. соглашения
// @Description Позволяет администратору обновить данные доп. соглашения
// @Tags Admin
// @Accept multipart/form-data
// @Produce json
// @Param Login header string true "Логин администратора (admin)"
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param agreement_id path int true "ID доп. соглашения"
// @Param delivery_conditions formData string false "Условия поставки"
// @Param delivery_term_days formData integer false "Срок поставки"
// @Param return_term_days formData integer false "Срок возврата"
// @Param subject formData string false "Предмет соглашения"
// @Param extend_date_to formData string false "Продлить до (YYYY-MM-DD)"
// @Param foreign_amount formData number false "Сумма в валюте (если есть)"
// @Param foreign_currency formData string false "Валюта (если есть)"
// @Param amount_in_contract_currency formData number false "Сумма в валюте контракта"
// @Param document formData file false "Новый PDF файл"
// @Success 200 {object} domain.AdditionalAgreement
// @Failure 400 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse "Доступ запрещен"
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id} [put]
func (h *InvoiceHandler) UpdateAdditionalAgreement(w http.ResponseWriter, r *http.Request) {
	agreementID, err := strconv.ParseInt(mux.Vars(r)["agreement_id"], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Неверный ID соглашения"})
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	existing, err := h.addlSvc.GetByID(r.Context(), agreementID)
	if err != nil {
		handleError(w, err)
		return
	}

	if v := strings.TrimSpace(r.FormValue("delivery_conditions")); v != "" { existing.DeliveryConditions = &v }
	if v := r.FormValue("delivery_term_days"); v != "" {
		if i, err := strconv.Atoi(v); err == nil { existing.DeliveryTermDays = &i }
	}
	if v := r.FormValue("return_term_days"); v != "" {
		if i, err := strconv.Atoi(v); err == nil { existing.ReturnTermDays = &i }
	}
	if v := strings.TrimSpace(r.FormValue("subject")); v != "" { existing.Subject = &v }
	if v := r.FormValue("extend_date_to"); v != "" {
		if d, err := time.Parse(time.DateOnly, v); err == nil { existing.ExtendDateTo = &d }
	}
	if v := r.FormValue("foreign_amount"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil { existing.ForeignAmount = &f }
	}
	if v := r.FormValue("foreign_currency"); v != "" {
		upper := strings.ToUpper(v)
		existing.ForeignCurrency = &upper
	}
	if v := r.FormValue("amount_in_contract_currency"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil { existing.AmountInContractCurrency = f }
	}

	file, handler, err := r.FormFile("document")
	if err == nil {
		defer file.Close()
		os.MkdirAll("uploads/additional_agreements", os.ModePerm)
		uniqueFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
		filePath := filepath.Join("uploads/additional_agreements", uniqueFileName)
		dst, err := os.Create(filePath)
		if err == nil {
			io.Copy(dst, file)
			dst.Close()
			existing.DocumentPath = filePath
			existing.OriginalDocumentName = handler.Filename
		}
	}

	updated, err := h.addlSvc.Update(r.Context(), existing.ID, existing)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}

// @Summary Удаление инвойса
// @Description Позволяет администратору удалить инвойс 
// @Tags Admin
// @Accept json
// @Produce json
// @Param Login header string true "Логин администратора (admin)"
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param invoice_id path int true "ID инвойса"
// @Success 200 {object} map[string]string "Сообщение об успешном удалении"
// @Failure 400 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id} [delete]
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

// @Summary Удаление ГТД
// @Description Позволяет администратору удалить ГТД 
// @Tags Admin
// @Accept json
// @Produce json
// @Param Login header string true "Логин администратора (admin)"
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param invoice_id path int true "ID инвойса"
// @Param gtd_id path int true "ID ГТД"
// @Success 200 {object} map[string]string "Сообщение об успешном удалении"
// @Failure 400 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/gtd/{gtd_id} [delete]
func (h *InvoiceHandler) DeleteGTD(w http.ResponseWriter, r *http.Request) {
	gtdIDStr := mux.Vars(r)["gtd_id"]
	gtdID, err := strconv.ParseInt(gtdIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID ГТД"})
		return
	}

	if err := h.gtdSvc.SoftDelete(r.Context(), gtdID); err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "ГТД успешно удалена"})
}

// @Summary Удаление доп. соглашения
// @Description Позволяет администратору удалить доп. соглашение 
// @Tags Admin
// @Accept json
// @Produce json
// @Param Login header string true "Логин администратора (admin)"
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param agreement_id path int true "ID доп. соглашения"
// @Success 200 {object} map[string]string "Сообщение об успешном удалении"
// @Failure 400 {object} dto.ErrorResponse
// @Failure 403 {object} dto.ErrorResponse
// @Failure 500 {object} dto.ErrorResponse
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id} [delete]
func (h *InvoiceHandler) DeleteAdditionalAgreement(w http.ResponseWriter, r *http.Request) {
	agreementIDStr := mux.Vars(r)["agreement_id"]
	agreementID, err := strconv.ParseInt(agreementIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID доп. соглашения"})
		return
	}

	if err := h.addlSvc.SoftDelete(r.Context(), agreementID); err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Доп. соглашение успешно удалено"})
}