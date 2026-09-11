package http
import(
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

// @Summary Список доп. соглашений контракта
// @Description Возвращает список дополнительных соглашений по контракту.
// @Tags AdditionalAgreements
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Success 200 {array} domain.AdditionalAgreement
// @Failure 401 {object} CommonError
// @Failure 500 {object} CommonError
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
	if list == nil {
		list = []domain.AdditionalAgreement{}
	}
	writeJSON(w, http.StatusOK, list)
}

// @Summary Карточка доп. соглашения (получить по ID)
// @Description Возвращает полную информацию по карточке дополнительного соглашения
// @Tags AdditionalAgreements
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param agreement_id path int true "ID доп. соглашения"
// @Success 200 {object} domain.AdditionalAgreement
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 404 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id} [get]
func (h *InvoiceHandler) GetAdditionalAgreementByID(w http.ResponseWriter, r *http.Request) {
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

	ag, err := h.addlSvc.GetByID(r.Context(), agreementID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, CommonError{Error: "Доп. соглашение не найдено"})
		return
	}

	writeJSON(w, http.StatusOK, ag)
}

// @Summary Создать доп. соглашение (Карточка Доп. соглашения)
// @Description Создает доп. соглашение к контракту с поддержкой прикрепления файла PDF/Word.
// @Tags AdditionalAgreements
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param agreement_number formData string false "Номер доп. соглашения"
// @Param agreement_date formData string false "Дата доп. соглашения (YYYY-MM-DD или DD.MM.YYYY)"
// @Param subject formData string false "Предмет соглашения"
// @Param delivery_date formData string false "Срок поставки товара / оказания услуг (дата)"
// @Param delivery_term_days formData int false "Срок поставки товара (в днях)"
// @Param delivery_conditions formData string false "Условия поставки"
// @Param return_date formData string false "Срок возврата денежных средств (дата)"
// @Param return_term_days formData int false "Срок возврата денежных средств (в днях)"
// @Param amount formData number false "Сумма доп. соглашения"
// @Param currency formData string false "Валюта доп. соглашения (например USD, EUR, TJS)"
// @Param agreement_end_date formData string false "Дата окончания доп. соглашения (YYYY-MM-DD)"
// @Param amount_in_contract_currency formData number false "Сумма в валюте контракта (если валюты отличаются)"
// @Param doc_type formData string false "Тип документа (additional_agreement, specification, appendix)"
// @Param document formData file false "Файл доп. соглашения (.pdf, .doc, .docx)"
// @Success 201 {object} domain.AdditionalAgreement
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 500 {object} CommonError
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

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	docType := domain.DocTypeAdditionalAgreement
	if v := strings.ToLower(strings.TrimSpace(r.FormValue("doc_type"))); v != "" {
		if v == domain.DocTypeSpecification || v == "спецификация" {
			docType = domain.DocTypeSpecification
		} else if v == domain.DocTypeAppendix || v == "приложение" {
			docType = domain.DocTypeAppendix
		}
	}

	ag := domain.AdditionalAgreement{
		ContractID: contractID,
		DocType:    docType,
		CreatedBy:  login,
	}

	if v := strings.TrimSpace(r.FormValue("agreement_number")); v != "" {
		ag.AgreementNumber = &v
	}
	if v := r.FormValue("agreement_date"); v != "" {
		if d := parseDate(v); d != nil {
			ag.AgreementDate = d
		}
	}
	if v := strings.TrimSpace(r.FormValue("subject")); v != "" {
		ag.Subject = &v
	}
	if v := r.FormValue("delivery_date"); v != "" {
		if d := parseDate(v); d != nil {
			ag.DeliveryDate = d
		}
	}
	if v := r.FormValue("delivery_term_days"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			ag.DeliveryTermDays = &i
		}
	}
	if v := strings.TrimSpace(r.FormValue("delivery_conditions")); v != "" {
		ag.DeliveryConditions = &v
	}
	if v := r.FormValue("return_date"); v != "" {
		if d := parseDate(v); d != nil {
			ag.ReturnDate = d
		}
	}
	if v := r.FormValue("return_term_days"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			ag.ReturnTermDays = &i
		}
	}

	// Сумма
	amountVal := r.FormValue("amount")
	if amountVal == "" {
		amountVal = r.FormValue("foreign_amount")
	}
	if amountVal != "" {
		if f, err := strconv.ParseFloat(amountVal, 64); err == nil {
			ag.ForeignAmount = &f
			ag.Amount = &f
		}
	}

	// Валюта
	currVal := strings.ToUpper(strings.TrimSpace(r.FormValue("currency")))
	if currVal == "" {
		currVal = strings.ToUpper(strings.TrimSpace(r.FormValue("foreign_currency")))
	}
	if currVal != "" {
		ag.ForeignCurrency = &currVal
		ag.Currency = &currVal
	}

	// Дата окончания
	endDateVal := r.FormValue("agreement_end_date")
	if endDateVal == "" {
		endDateVal = r.FormValue("extend_date_to")
	}
	if endDateVal != "" {
		if d := parseDate(endDateVal); d != nil {
			ag.ExtendDateTo = d
			ag.AgreementEndDate = d
		}
	}

	if v := r.FormValue("amount_in_contract_currency"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			ag.AmountInContractCurrency = f
		}
	}

	// Файл (опционально)
	file, handler, err := r.FormFile("document")
	if err == nil {
		defer file.Close()
		ext := strings.ToLower(filepath.Ext(handler.Filename))
		if ext != ".pdf" && ext != ".doc" && ext != ".docx" {
			writeJSON(w, http.StatusBadRequest, CommonError{Error: "Разрешены только файлы форматов PDF и Word (.pdf, .doc, .docx)"})
			return
		}

		_ = os.MkdirAll("uploads/additional_agreements", os.ModePerm)
		uniqueFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
		filePath := filepath.Join("uploads/additional_agreements", uniqueFileName)
		dst, err := os.Create(filePath)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, CommonError{Error: "Ошибка при сохранении файла"})
			return
		}
		defer dst.Close()
		if _, err := io.Copy(dst, file); err != nil {
			writeJSON(w, http.StatusInternalServerError, CommonError{Error: "Ошибка при записи файла"})
			return
		}
		ag.DocumentPath = filePath
		ag.OriginalDocumentName = handler.Filename
	}

	created, err := h.addlSvc.Create(r.Context(), ag)
	if err != nil {
		handleError(w, err)
		return
	}

	actionDesc := "Создание доп. соглашения"
	if created.DocType == domain.DocTypeSpecification {
		actionDesc = "Создание спецификации"
	} else if created.DocType == domain.DocTypeAppendix {
		actionDesc = "Создание приложения к контракту"
	}
	LogUserAction(r, "CREATE", "additional_agreement", &created.ID, actionDesc)

	writeJSON(w, http.StatusCreated, created)
}

// @Summary Редактирование доп. соглашения
// @Description Позволяет обновить данные карточки дополнительного соглашения
// @Tags AdditionalAgreements
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param agreement_id path int true "ID доп. соглашения"
// @Param agreement_number formData string false "Номер доп. соглашения"
// @Param agreement_date formData string false "Дата доп. соглашения (YYYY-MM-DD или DD.MM.YYYY)"
// @Param subject formData string false "Предмет соглашения"
// @Param delivery_date formData string false "Срок поставки товара (дата)"
// @Param delivery_term_days formData integer false "Срок поставки (дни)"
// @Param delivery_conditions formData string false "Новые условия поставки"
// @Param return_date formData string false "Срок возврата денежных средств (дата)"
// @Param return_term_days formData integer false "Срок возврата денежных средств (дни)"
// @Param amount formData number false "Сумма доп. соглашения"
// @Param currency formData string false "Валюта доп. соглашения"
// @Param agreement_end_date formData string false "Дата окончания доп. соглашения"
// @Param amount_in_contract_currency formData number false "Сумма в валюте контракта (для изменения остатка)"
// @Param doc_type formData string false "Тип документа"
// @Param document formData file false "Новый PDF/Word документ (опционально)"
// @Success 200 {object} domain.AdditionalAgreement
// @Failure 400 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /admin/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id} [put]
func (h *InvoiceHandler) UpdateAdditionalAgreement(w http.ResponseWriter, r *http.Request) {
	agreementID, err := strconv.ParseInt(mux.Vars(r)["agreement_id"], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Неверный ID соглашения"})
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	existing, err := h.addlSvc.GetByID(r.Context(), agreementID)
	if err != nil {
		handleError(w, err)
		return
	}

	if v := strings.TrimSpace(r.FormValue("agreement_number")); v != "" {
		existing.AgreementNumber = &v
	}
	if v := r.FormValue("agreement_date"); v != "" {
		if d := parseDate(v); d != nil {
			existing.AgreementDate = d
		}
	}
	if v := strings.TrimSpace(r.FormValue("subject")); v != "" {
		existing.Subject = &v
	}
	if v := r.FormValue("delivery_date"); v != "" {
		if d := parseDate(v); d != nil {
			existing.DeliveryDate = d
		}
	}
	if v := strings.TrimSpace(r.FormValue("delivery_conditions")); v != "" {
		existing.DeliveryConditions = &v
	}
	if v := r.FormValue("delivery_term_days"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			existing.DeliveryTermDays = &i
		}
	}
	if v := r.FormValue("return_date"); v != "" {
		if d := parseDate(v); d != nil {
			existing.ReturnDate = d
		}
	}
	if v := r.FormValue("return_term_days"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			existing.ReturnTermDays = &i
		}
	}

	// Сумма
	amountVal := r.FormValue("amount")
	if amountVal == "" {
		amountVal = r.FormValue("foreign_amount")
	}
	if amountVal != "" {
		if f, err := strconv.ParseFloat(amountVal, 64); err == nil {
			existing.ForeignAmount = &f
			existing.Amount = &f
		}
	}

	// Валюта
	currVal := strings.ToUpper(strings.TrimSpace(r.FormValue("currency")))
	if currVal == "" {
		currVal = strings.ToUpper(strings.TrimSpace(r.FormValue("foreign_currency")))
	}
	if currVal != "" {
		existing.ForeignCurrency = &currVal
		existing.Currency = &currVal
	}

	// Дата окончания
	endDateVal := r.FormValue("agreement_end_date")
	if endDateVal == "" {
		endDateVal = r.FormValue("extend_date_to")
	}
	if endDateVal != "" {
		if d := parseDate(endDateVal); d != nil {
			existing.ExtendDateTo = d
			existing.AgreementEndDate = d
		}
	}

	if v := r.FormValue("amount_in_contract_currency"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			existing.AmountInContractCurrency = f
		}
	}
	if v := strings.ToLower(strings.TrimSpace(r.FormValue("doc_type"))); v != "" {
		if v == domain.DocTypeSpecification || v == "спецификация" {
			existing.DocType = domain.DocTypeSpecification
		} else if v == domain.DocTypeAppendix || v == "приложение" {
			existing.DocType = domain.DocTypeAppendix
		} else {
			existing.DocType = domain.DocTypeAdditionalAgreement
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
		_ = os.MkdirAll("uploads/additional_agreements", os.ModePerm)
		uniqueFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
		filePath := filepath.Join("uploads/additional_agreements", uniqueFileName)
		dst, err := os.Create(filePath)
		if err == nil {
			_, _ = io.Copy(dst, file)
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

	LogUserAction(r, "UPDATE", "additional_agreement", &updated.ID, "Обновление доп. соглашения")

	writeJSON(w, http.StatusOK, updated)
}

// @Summary Загрузить документ к доп. соглашению (PDF / Word)
// @Description Кнопка добавления файла (PDF/Word) в карточке доп. соглашения
// @Tags AdditionalAgreements
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param agreement_id path int true "ID доп. соглашения"
// @Param document formData file true "Файл документа (.pdf, .doc, .docx)"
// @Success 200 {object} domain.AdditionalAgreement
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 404 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/document [post]
func (h *InvoiceHandler) UploadDocumentAdditionalAgreement(w http.ResponseWriter, r *http.Request) {
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

	existing, err := h.addlSvc.GetByID(r.Context(), agreementID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, CommonError{Error: "Доп. соглашение не найдено"})
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

	_ = os.MkdirAll("uploads/additional_agreements", os.ModePerm)
	uniqueFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
	filePath := filepath.Join("uploads/additional_agreements", uniqueFileName)
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

	existing.DocumentPath = filePath
	existing.OriginalDocumentName = handler.Filename

	updated, err := h.addlSvc.Update(r.Context(), existing.ID, existing)
	if err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "UPLOAD_DOCUMENT", "additional_agreement", &updated.ID, fmt.Sprintf("Загрузка документа к доп. соглашению: %s", handler.Filename))

	writeJSON(w, http.StatusOK, updated)
}

// @Summary Просмотр или скачивание документа доп. соглашения
// @Description Отдает файл доп. соглашения (PDF / Word) для просмотра или скачивания
// @Tags AdditionalAgreements
// @Security ApiKeyAuth
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param agreement_id path int true "ID доп. соглашения"
// @Success 200 {file} file
// @Failure 401 {object} CommonError
// @Failure 404 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/document [get]
func (h *InvoiceHandler) GetDocumentAdditionalAgreement(w http.ResponseWriter, r *http.Request) {
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

	ag, err := h.addlSvc.GetByID(r.Context(), agreementID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, CommonError{Error: "Доп. соглашение не найдено"})
		return
	}

	if ag.DocumentPath == "" {
		writeJSON(w, http.StatusNotFound, CommonError{Error: "Документ не прикреплен к данному соглашению"})
		return
	}

	http.ServeFile(w, r, ag.DocumentPath)
}

// @Summary Удаление доп. соглашения
// @Description Позволяет администратору удалить доп. соглашение 
// @Tags Admin
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param agreement_id path int true "ID доп. соглашения"
// @Success 200 {object} map[string]string "Сообщение об успешном удалении"
// @Failure 400 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /admin/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id} [delete]
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

	LogUserAction(r, "DELETE", "additional_agreement", &agreementID, "Удаление доп. соглашения")

	writeJSON(w, http.StatusOK, map[string]string{"message": "Доп. соглашение успешно удалено"})
}
