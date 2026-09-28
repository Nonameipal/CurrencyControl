package http

import (
	"net/http"
	"strconv"
	"strings"

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
	contractID, ok := requireID(w, r, "contract_id")
	if !ok {
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
	agreementID, ok := requireID(w, r, "agreement_id")
	if !ok {
		return
	}

	ag, err := h.addlSvc.GetByID(r.Context(), agreementID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, CommonError{Error: "Доп. соглашение не найдено"})
		return
	}

	writeJSON(w, http.StatusOK, ag)
}

// @Summary Создание дополнительного соглашения к контракту
// @Description Создает новое доп. соглашение, спецификацию или приложение к контракту.
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
// @Param delivery_date formData string false "Срок поставки товара / оказания услуг (дата YYYY-MM-DD или DD.MM.YYYY)"
// @Param return_days formData integer false "Срок возврата денежных средств в днях (число дней, > 0, опционально)"
// @Param amount formData number false "Сумма доп. соглашения"
// @Param currency formData string false "Валюта доп. соглашения (например USD, EUR, TJS)"
// @Param receiver_name formData string false "Получатель"
// @Param receiver_bank formData string false "Банк получатель"
// @Param receiver_country formData string false "Страна получателя"
// @Param agreement_end_date formData string false "Дата окончания доп. соглашения (YYYY-MM-DD или DD.MM.YYYY)"
// @Param document formData file false "Файл доп. соглашения (.pdf)"
// @Success 201 {object} domain.AdditionalAgreement
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements [post]
func (h *InvoiceHandler) CreateAdditionalAgreement(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	contractID, ok := requireID(w, r, "contract_id")
	if !ok {
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	ag := domain.AdditionalAgreement{
		ContractID: contractID,
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

	if v := strings.TrimSpace(r.FormValue("return_days")); v != "" {
		days, err := strconv.Atoi(v)
		if err != nil || days <= 0 {
			writeJSON(w, http.StatusBadRequest, CommonError{Error: "Срок возврата денежных средств (return_days) должен быть целым числом больше 0"})
			return
		}
		ag.ReturnDays = &days
	}

	if endDateVal := getFormValueFallback(r, "agreement_end_date", "extend_date_to"); endDateVal != "" {
		if d := parseDate(endDateVal); d != nil {
			ag.ExtendDateTo = d
			ag.AgreementEndDate = d
		}
	}



	if v := strings.TrimSpace(r.FormValue("receiver_name")); v != "" {
		ag.ReceiverName = v
	}
	if v := strings.TrimSpace(r.FormValue("receiver_bank")); v != "" {
		ag.ReceiverBank = v
	}
	if rc := getFormValueFallback(r, "receiver_country", "recipient_country"); rc != "" {
		ag.ReceiverCountry = rc
	}
	
	ag.SenderName = strings.TrimSpace(r.FormValue("sender_name"))
	ag.SenderBank = strings.TrimSpace(r.FormValue("sender_bank"))
	ag.SenderCountry = strings.TrimSpace(r.FormValue("sender_country"))

	if amountVal := getFormValueFallback(r, "amount", "foreign_amount"); amountVal != "" {
		if f, err := strconv.ParseFloat(amountVal, 64); err == nil {
			ag.ForeignAmount = &f
			ag.Amount = &f
		}
	}

	if currVal := strings.ToUpper(getFormValueFallback(r, "currency", "foreign_currency")); currVal != "" {
		ag.ForeignCurrency = &currVal
		ag.Currency = &currVal
	}

	if filePath, err := saveUploadedFile(r, "document", "uploads/additional_agreements", false); err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	} else if filePath != "" {
		ag.DocumentPath = filePath
	}

	created, err := h.addlSvc.Create(r.Context(), ag)
	if err != nil {
		handleError(w, err)
		return
	}

	actionDesc := "Создание доп. соглашения"
	LogUserAction(r, "CREATE", "additional_agreement", &created.ID, actionDesc)

	writeJSON(w, http.StatusCreated, created)
}

// @Summary Редактирование доп. соглашения
// @Description Редактирование дополнительного соглашения. Доступно: Сотрудник валютного контроля (при наличии разрешения), Комплаенс, Администратор.
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
// @Param delivery_date formData string false "Срок поставки товара (дата YYYY-MM-DD или DD.MM.YYYY)"
// @Param agreement_end_date formData string false "Дата окончания доп. соглашения (YYYY-MM-DD или DD.MM.YYYY)"
// @Param return_days formData integer false "Срок возврата денежных средств в днях (число дней, > 0, опционально)"
// @Param document formData file false "Новый PDF документ (.pdf, опционально)"
// @Success 200 {object} domain.AdditionalAgreement
// @Failure 400 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id} [put]
func (h *InvoiceHandler) UpdateAdditionalAgreement(w http.ResponseWriter, r *http.Request) {
	agreementID, ok := requireID(w, r, "agreement_id")
	if !ok {
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

	if v := strings.TrimSpace(r.FormValue("return_days")); v != "" {
		days, err := strconv.Atoi(v)
		if err != nil || days <= 0 {
			writeJSON(w, http.StatusBadRequest, CommonError{Error: "Срок возврата денежных средств (return_days) должен быть целым числом больше 0"})
			return
		}
		existing.ReturnDays = &days
	}

	if v := strings.TrimSpace(r.FormValue("receiver_name")); v != "" {
		existing.ReceiverName = v
	}
	if v := strings.TrimSpace(r.FormValue("receiver_bank")); v != "" {
		existing.ReceiverBank = v
	}
	if rc := getFormValueFallback(r, "receiver_country", "recipient_country"); rc != "" {
		existing.ReceiverCountry = rc
	}

	if v := strings.TrimSpace(r.FormValue("sender_name")); v != "" {
		existing.SenderName = v
	}
	if v := strings.TrimSpace(r.FormValue("sender_bank")); v != "" {
		existing.SenderBank = v
	}
	if v := strings.TrimSpace(r.FormValue("sender_country")); v != "" {
		existing.SenderCountry = v
	}

	if amountVal := getFormValueFallback(r, "amount", "foreign_amount"); amountVal != "" {
		if f, err := strconv.ParseFloat(amountVal, 64); err == nil {
			existing.ForeignAmount = &f
			existing.Amount = &f
		}
	}

	if currVal := strings.ToUpper(getFormValueFallback(r, "currency", "foreign_currency")); currVal != "" {
		existing.ForeignCurrency = &currVal
		existing.Currency = &currVal
	}

	if endDateVal := getFormValueFallback(r, "agreement_end_date", "extend_date_to"); endDateVal != "" {
		if d := parseDate(endDateVal); d != nil {
			existing.ExtendDateTo = d
			existing.AgreementEndDate = d
		}
	}



	if filePath, err := saveUploadedFile(r, "document", "uploads/additional_agreements", false); err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	} else if filePath != "" {
		existing.DocumentPath = filePath
	}

	updated, err := h.addlSvc.Update(r.Context(), existing.ID, existing)
	if err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "UPDATE", "additional_agreement", &updated.ID, "Обновление доп. соглашения")

	writeJSON(w, http.StatusOK, updated)
}

// @Summary Удаление доп. соглашения (в корзину)
// @Description Помещает дополнительное соглашение в корзину. Доступно: Сотрудники валютного контроля, Комплаенс, Администратор.
// @Tags AdditionalAgreements
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

	LogUserAction(r, "DELETE", "additional_agreement", &agreementID, "Удаление доп. соглашения")

	writeJSON(w, http.StatusOK, map[string]string{"message": "Доп. соглашение успешно удалено"})
}

// @Summary Разархивировать доп. соглашение (восстановить в active)
// @Description Переводит доп. соглашение из статуса archived обратно в active. Доступно: Сотрудники валютного контроля, Комплаенс, Администратор.
// @Tags Archive
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param agreement_id path int true "ID доп. соглашения"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/restore [put]
func (h *InvoiceHandler) RestoreAdditionalAgreement(w http.ResponseWriter, r *http.Request) {
	agreementID, ok := requireID(w, r, "agreement_id")
	if !ok {
		return
	}

	if err := h.addlSvc.RestoreAdditionalAgreement(r.Context(), agreementID); err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "RESTORE", "additional_agreement", &agreementID, "Восстановление доп. соглашения из архива")
	writeJSON(w, http.StatusOK, map[string]string{"message": "Доп. соглашение успешно восстановлено из архива"})
}
