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

	"github.com/gorilla/mux"
)

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
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Failure 500 {object} map[string]string
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
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 500 {object} map[string]string
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
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string "Доступ запрещен"
// @Failure 500 {object} map[string]string
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
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
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

