package document

import (
	"fmt"
	"net/http"
	"strings"

	"CurrencyControl/internal/domain"
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

// @Summary Добавить дополнительное соглашение (Карточка ДС)
// @Description Создает новое дополнительное соглашение к контракту. Поля: Номер ДС, Дата ДС (календарь), Валюта (справочник), Сумма, Создал (операционист) + PDF документ.
// @Tags AdditionalAgreements
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param agreement_number formData string true "Номер доп. соглашения"
// @Param agreement_date formData string true "Дата доп. соглашения (YYYY-MM-DD или DD.MM.YYYY)"
// @Param currency formData string true "Валюта (справочник: USD, EUR, RUB, TJS, CNY...)"
// @Param amount formData number true "Сумма доп. соглашения"
// @Param document formData file true "Файл документа доп. соглашения (.pdf)"
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

	v := NewFormValidator(r, 32<<20)
	v.RequireStrings(additionalAgreementRequiredFields)
	agreementNumber := v.String("agreement_number")
	agreementDate := v.Date("agreement_date")
	currency := strings.ToUpper(v.String("currency"))
	amount := v.Float("amount")

	if v.Respond(w) {
		return
	}

	var docPath string
	if savedPath, saveErr := saveUploadedFile(r, "document", "uploads/additional_agreements", true); saveErr != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: saveErr.Error()})
		return
	} else if savedPath != "" {
		docPath = savedPath
	}

	ag := domain.AdditionalAgreement{
		ContractID:      contractID,
		AgreementNumber: &agreementNumber,
		AgreementDate:   agreementDate,
		Currency:        &currency,
		ForeignCurrency: &currency,
		Amount:          &amount,
		ForeignAmount:   &amount,
		RemainingAmount: amount,
		DocumentPath:    docPath,
		CreatedBy:       login,
		Status:          domain.AgreementStatusActive,
	}

	created, err := h.addlSvc.Create(r.Context(), ag)
	if err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "CREATE", "additional_agreement", &created.ID, fmt.Sprintf("Создание дополнительного соглашения № %s", *created.AgreementNumber))
	writeJSON(w, http.StatusCreated, created)
}

// @Summary Редактирование доп. соглашения (Карточка ДС)
// @Description Редактирование дополнительного соглашения к контракту. Доступно: Сотрудник валютного контроля (при наличии разрешения), Комплаенс, Администратор.
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
// @Param currency formData string false "Валюта (справочник)"
// @Param amount formData number false "Сумма доп. соглашения"
// @Param document formData file false "Новый PDF документ (.pdf, опционально)"
// @Success 200 {object} domain.AdditionalAgreement
// @Failure 400 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id} [put]
func (h *InvoiceHandler) UpdateAdditionalAgreement(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	agreementID, ok := requireID(w, r, "agreement_id")
	if !ok {
		return
	}

	existing, err := h.addlSvc.GetByID(r.Context(), agreementID)
	if err != nil {
		handleError(w, err)
		return
	}

	diff := NewDiffTracker()
	v := NewFormValidator(r, 32<<20)
	if num := strings.TrimSpace(r.FormValue("agreement_number")); num != "" {
		diff.TrackStringPtr("Номер доп. соглашения", existing.AgreementNumber, &num)
		existing.AgreementNumber = &num
	}
	if dateStr := strings.TrimSpace(r.FormValue("agreement_date")); dateStr != "" {
		d := v.Date("agreement_date")
		if d != nil {
			diff.TrackDatePtr("Дата доп. соглашения", existing.AgreementDate, d)
			existing.AgreementDate = d
		}
	}
	if currVal := strings.ToUpper(strings.TrimSpace(r.FormValue("currency"))); currVal != "" {
		diff.TrackStringPtr("Валюта доп. соглашения", existing.Currency, &currVal)
		existing.ForeignCurrency = &currVal
		existing.Currency = &currVal
	}
	if amtStr := strings.TrimSpace(r.FormValue("amount")); amtStr != "" {
		amt := v.Float("amount")
		if amt > 0 {
			oldAmt := 0.0
			if existing.Amount != nil {
				oldAmt = *existing.Amount
			}
			diff.TrackFloat("Сумма доп. соглашения", oldAmt, amt)
			existing.ForeignAmount = &amt
			existing.Amount = &amt
		}
	}

	if v.Respond(w) {
		return
	}

	if savedPath, saveErr := saveUploadedFile(r, "document", "uploads/additional_agreements", false); saveErr != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: saveErr.Error()})
		return
	} else if savedPath != "" {
		diff.TrackCustom("Загружен новый файл документа")
		existing.DocumentPath = savedPath
	} else if docPathStr := strings.TrimSpace(r.FormValue("document_path")); docPathStr != "" {
		diff.TrackCustom("Изменен путь к документу")
		existing.DocumentPath = docPathStr
	}

	existing.UpdatedBy = login
	updated, err := h.addlSvc.Update(r.Context(), existing.ID, existing)
	if err != nil {
		handleError(w, err)
		return
	}

	num := ""
	if updated.AgreementNumber != nil {
		num = *updated.AgreementNumber
	}
	details := fmt.Sprintf("Обновление доп. соглашения № %s. Изменения: %s", num, diff.String())
	LogUserAction(r, "UPDATE", "additional_agreement", &updated.ID, details)
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
	agreementID, ok := requireID(w, r, "agreement_id")
	if !ok {
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
