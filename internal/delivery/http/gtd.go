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
// @Description Возвращает список ГТД, привязанных к указанному инвойсу.
// @Tags GTD
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param invoice_id path int true "ID инвойса"
// @Success 200 {array} domain.GTD
// @Failure 401 {object} CommonError
// @Failure 500 {object} CommonError
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

	list, err := h.gtdSvc.GetListByInvoiceID(r.Context(), invoiceID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: err.Error()})
		return
	}
	if list == nil {
		list = []domain.GTD{}
	}

	writeJSON(w, http.StatusOK, list)
}

// @Summary Список всех ГТД по контракту
// @Description Возвращает все ГТД и акты выполненных работ, относящиеся к контракту.
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
func (h *InvoiceHandler) GetContractGTDs(w http.ResponseWriter, r *http.Request) {
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

	list, err := h.gtdSvc.GetByContractID(r.Context(), contractID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: err.Error()})
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
func (h *InvoiceHandler) GetGTDByID(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}
	gtdID, err := strconv.ParseInt(mux.Vars(r)["gtd_id"], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID ГТД"})
		return
	}

	gtd, err := h.gtdSvc.GetByID(r.Context(), gtdID)
	if err != nil || gtd == nil {
		writeJSON(w, http.StatusNotFound, CommonError{Error: "ГТД не найдена"})
		return
	}

	writeJSON(w, http.StatusOK, gtd)
}

// @Summary Добавить ГТД (Карточка ГТД)
// @Description Добавляет новую ГТД с привязкой к контракту и инвойсу. Позволяет прикрепить файл PDF/Word. Система автоматически сопоставляет дату загрузки ГТД с регламентированным сроком поставки по контракту и возвращает точное информационное уведомление о днях опережения или просрочки (delivery_notice, days_difference, delivery_status).
// @Tags GTD
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param invoice_id formData int false "ID инвойса (если создается не в контексте инвойса)"
// @Param gtd_number formData string true "Номер ГТД"
// @Param gtd_date formData string true "Дата ГТД (YYYY-MM-DD или DD.MM.YYYY)"
// @Param gtd_amount formData number true "Сумма ГТД (в валюте ГТД)"
// @Param gtd_currency formData string true "Валюта ГТД (например USD, EUR, TJS)"
// @Param closes_amount formData number true "Сколько закрывается по инвойсу (в валюте инвойса)"
// @Param hs_code formData string false "Код ТН ВЭД (HS CODE)"
// @Param destination_country formData string false "Страна поступления товара"
// @Param document_type formData string false "Тип документа (gtd или act)"
// @Param document formData file false "Файл документа ГТД (PDF / Word)"
// @Success 201 {object} domain.GTD
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/gtd [post]
func (h *InvoiceHandler) CreateGTD(w http.ResponseWriter, r *http.Request) {
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

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	invoiceIDStr := mux.Vars(r)["invoice_id"]
	if invoiceIDStr == "" {
		invoiceIDStr = r.FormValue("invoice_id")
	}
	invoiceID, err := strconv.ParseInt(invoiceIDStr, 10, 64)
	if err != nil || invoiceID <= 0 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле invoice_id обязательно и должно указывать на существующий инвойс"})
		return
	}

	gtdNumber := strings.TrimSpace(r.FormValue("gtd_number"))
	gtdDateStr := r.FormValue("gtd_date")
	gtdCurrencyStr := strings.ToUpper(strings.TrimSpace(r.FormValue("gtd_currency")))

	if gtdNumber == "" || gtdDateStr == "" || gtdCurrencyStr == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поля gtd_number, gtd_date и gtd_currency обязательны"})
		return
	}

	gtdDate := parseDate(gtdDateStr)
	if gtdDate == nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Неверный формат gtd_date. Ожидается YYYY-MM-DD или DD.MM.YYYY"})
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

	hsCode := strings.TrimSpace(r.FormValue("hs_code"))
	destinationCountry := strings.TrimSpace(r.FormValue("destination_country"))
	if destinationCountry == "" {
		destinationCountry = strings.TrimSpace(r.FormValue("country_of_destination"))
	}
	if destinationCountry == "" {
		destinationCountry = strings.TrimSpace(r.FormValue("country"))
	}

	var docPath *string
	var origName *string
	file, handler, err := r.FormFile("document")
	if err == nil {
		defer file.Close()
		ext := strings.ToLower(filepath.Ext(handler.Filename))
		if ext != ".pdf" && ext != ".doc" && ext != ".docx" {
			writeJSON(w, http.StatusBadRequest, CommonError{Error: "Разрешены только файлы форматов PDF и Word (.pdf, .doc, .docx)"})
			return
		}

		_ = os.MkdirAll("uploads/gtd", os.ModePerm)
		uniqueFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
		filePath := filepath.Join("uploads/gtd", uniqueFileName)
		dst, err := os.Create(filePath)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, CommonError{Error: "Ошибка при сохранении файла на сервер"})
			return
		}
		defer dst.Close()
		if _, err := io.Copy(dst, file); err != nil {
			writeJSON(w, http.StatusInternalServerError, CommonError{Error: "Ошибка при записи файла"})
			return
		}
		docPath = &filePath
		origName = &handler.Filename
	}

	docType := domain.DocumentTypeGTD
	if v := strings.ToLower(strings.TrimSpace(r.FormValue("document_type"))); v == domain.DocumentTypeAct || v == "акт" || v == "акт выполненных работ" {
		docType = domain.DocumentTypeAct
	}

	g := domain.GTD{
		ContractID:           contractID,
		InvoiceID:            invoiceID,
		DocumentType:         docType,
		GTDNumber:            gtdNumber,
		GTDAmount:            gtdAmount,
		GTDCurrency:          &gtdCurrencyStr,
		GTDDate:              gtdDate,
		ClosesAmount:         closesAmount,
		HSCode:               hsCode,
		DestinationCountry:   destinationCountry,
		DocumentPath:         docPath,
		OriginalDocumentName: origName,
		CreatedBy:            login,
	}

	created, err := h.gtdSvc.Create(r.Context(), g)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}

	actionDesc := "Создание ГТД № " + created.GTDNumber
	if created.DocumentType == domain.DocumentTypeAct {
		actionDesc = "Создание Акта выполненных работ № " + created.GTDNumber
	}
	LogUserAction(r, "CREATE", "gtd", &created.ID, actionDesc)

	writeJSON(w, http.StatusCreated, created)
}

// @Summary Редактирование ГТД
// @Description Позволяет обновить данные ГТД (номер, дату, сумму, валюту, закрытие, HS CODE, страну поступления)
// @Tags GTD
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param gtd_id path int true "ID ГТД"
// @Param gtd_number formData string false "Номер ГТД"
// @Param gtd_amount formData number false "Сумма ГТД"
// @Param gtd_currency formData string false "Валюта ГТД"
// @Param gtd_date formData string false "Дата ГТД (YYYY-MM-DD или DD.MM.YYYY)"
// @Param closes_amount formData number false "Сколько закрывается по инвойсу"
// @Param hs_code formData string false "Код ТН ВЭД (HS CODE)"
// @Param destination_country formData string false "Страна поступления товара"
// @Param document_type formData string false "Тип документа"
// @Param document formData file false "Новый файл ГТД (.pdf, .doc, .docx)"
// @Success 200 {object} domain.GTD
// @Failure 400 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 404 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /admin/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/gtd/{gtd_id} [put]
func (h *InvoiceHandler) UpdateGTD(w http.ResponseWriter, r *http.Request) {
	gtdIDStr := mux.Vars(r)["gtd_id"]
	gtdID, err := strconv.ParseInt(gtdIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Неверный ID ГТД"})
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	existing, err := h.gtdSvc.GetByID(r.Context(), gtdID)
	if err != nil || existing == nil {
		writeJSON(w, http.StatusNotFound, CommonError{Error: "ГТД не найдена"})
		return
	}

	if v := strings.TrimSpace(r.FormValue("gtd_number")); v != "" {
		existing.GTDNumber = v
	}
	if v := r.FormValue("gtd_date"); v != "" {
		if d := parseDate(v); d != nil {
			existing.GTDDate = d
		}
	}
	if v := r.FormValue("gtd_amount"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			existing.GTDAmount = f
		}
	}
	if v := r.FormValue("closes_amount"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			existing.ClosesAmount = f
		}
	}
	if v := strings.TrimSpace(r.FormValue("gtd_currency")); v != "" {
		upper := strings.ToUpper(v)
		existing.GTDCurrency = &upper
	}
	if v := strings.TrimSpace(r.FormValue("hs_code")); v != "" {
		existing.HSCode = v
	}
	if v := strings.TrimSpace(r.FormValue("destination_country")); v != "" {
		existing.DestinationCountry = v
	} else if v := strings.TrimSpace(r.FormValue("country_of_destination")); v != "" {
		existing.DestinationCountry = v
	} else if v := strings.TrimSpace(r.FormValue("country")); v != "" {
		existing.DestinationCountry = v
	}
	if v := strings.ToLower(strings.TrimSpace(r.FormValue("document_type"))); v != "" {
		if v == domain.DocumentTypeAct || v == "акт" || v == "акт выполненных работ" {
			existing.DocumentType = domain.DocumentTypeAct
		} else {
			existing.DocumentType = domain.DocumentTypeGTD
		}
	}

	file, handler, err := r.FormFile("document")
	if err == nil {
		defer file.Close()
		ext := strings.ToLower(filepath.Ext(handler.Filename))
		if ext != ".pdf" && ext != ".doc" && ext != ".docx" {
			writeJSON(w, http.StatusBadRequest, CommonError{Error: "Разрешены только файлы форматов PDF и Word (.pdf, .doc, .docx)"})
			return
		}
		_ = os.MkdirAll("uploads/gtd", os.ModePerm)
		uniqueFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
		filePath := filepath.Join("uploads/gtd", uniqueFileName)
		dst, err := os.Create(filePath)
		if err == nil {
			_, _ = io.Copy(dst, file)
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

	LogUserAction(r, "UPDATE", "gtd", &updated.ID, "Обновление ГТД № "+updated.GTDNumber)

	writeJSON(w, http.StatusOK, updated)
}

// @Summary Загрузить документ к ГТД (PDF / Word)
// @Description Кнопка добавления файла (PDF/Word) в карточке ГТД
// @Tags GTD
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param gtd_id path int true "ID ГТД"
// @Param document formData file true "Файл документа (.pdf, .doc, .docx)"
// @Success 200 {object} domain.GTD
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 404 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/gtd/{gtd_id}/document [post]
func (h *InvoiceHandler) UploadDocumentGTD(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}

	gtdID, err := strconv.ParseInt(mux.Vars(r)["gtd_id"], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID ГТД"})
		return
	}

	existing, err := h.gtdSvc.GetByID(r.Context(), gtdID)
	if err != nil || existing == nil {
		writeJSON(w, http.StatusNotFound, CommonError{Error: "ГТД не найдена"})
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	file, handler, err := r.FormFile("document")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле 'document' с файлом обязательно"})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(handler.Filename))
	if ext != ".pdf" && ext != ".doc" && ext != ".docx" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Разрешены только файлы форматов PDF и Word (.pdf, .doc, .docx)"})
		return
	}

	_ = os.MkdirAll("uploads/gtd", os.ModePerm)
	uniqueFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
	filePath := filepath.Join("uploads/gtd", uniqueFileName)
	dst, err := os.Create(filePath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: "Ошибка при сохранении файла на сервер"})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, file); err != nil {
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: "Ошибка при записи файла"})
		return
	}

	pathStr := filePath
	nameStr := handler.Filename
	existing.DocumentPath = &pathStr
	existing.OriginalDocumentName = &nameStr

	updated, err := h.gtdSvc.Update(r.Context(), existing.ID, *existing)
	if err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "UPLOAD_DOCUMENT", "gtd", &updated.ID, fmt.Sprintf("Загрузка документа к ГТД №%s: %s", updated.GTDNumber, handler.Filename))

	writeJSON(w, http.StatusOK, updated)
}

// @Summary Просмотр или скачивание документа ГТД
// @Description Отдает файл ГТД (PDF / Word) для просмотра или скачивания
// @Tags GTD
// @Security ApiKeyAuth
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param gtd_id path int true "ID ГТД"
// @Success 200 {file} file
// @Failure 401 {object} CommonError
// @Failure 404 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/gtd/{gtd_id}/document [get]
func (h *InvoiceHandler) GetDocumentGTD(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}

	gtdID, err := strconv.ParseInt(mux.Vars(r)["gtd_id"], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID ГТД"})
		return
	}

	gtd, err := h.gtdSvc.GetByID(r.Context(), gtdID)
	if err != nil || gtd == nil {
		writeJSON(w, http.StatusNotFound, CommonError{Error: "ГТД не найдена"})
		return
	}

	if gtd.DocumentPath == nil || *gtd.DocumentPath == "" {
		writeJSON(w, http.StatusNotFound, CommonError{Error: "Документ не прикреплен к данной ГТД"})
		return
	}

	http.ServeFile(w, r, *gtd.DocumentPath)
}

// @Summary Удаление ГТД
// @Description Позволяет администратору удалить ГТД
// @Tags Admin
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param gtd_id path int true "ID ГТД"
// @Success 200 {object} map[string]string "Сообщение об успешном удалении"
// @Failure 400 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /admin/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/gtd/{gtd_id} [delete]
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

	LogUserAction(r, "DELETE", "gtd", &gtdID, "Удаление ГТД")

	writeJSON(w, http.StatusOK, map[string]string{"message": "ГТД успешно удалена"})
}
