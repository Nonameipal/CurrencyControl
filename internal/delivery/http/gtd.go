package http

import (
	"net/http"
	"strconv"
	"strings"

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
	invoiceID, ok := requireID(w, r, "invoice_id")
	if !ok {
		return
	}
	list, err := h.gtdSvc.GetListByInvoiceID(r.Context(), invoiceID)
	if err != nil {
		handleError(w, err)
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
	contractID, ok := requireID(w, r, "contract_id")
	if !ok {
		return
	}

	list, err := h.gtdSvc.GetByContractID(r.Context(), contractID)
	if err != nil {
		handleError(w, err)
		return
	}
	if list == nil {
		list = []domain.GTD{}
	}

	writeJSON(w, http.StatusOK, list)
}

// @Summary Список всех ГТД по доп. соглашению
// @Description Возвращает все ГТД и акты выполненных работ, относящиеся к доп. соглашению.
// @Tags GTD
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param agreement_id path int true "ID доп. соглашения"
// @Success 200 {array} domain.GTD
// @Failure 401 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id}/gtd [get]
func (h *InvoiceHandler) GetAdditionalAgreementGTDs(w http.ResponseWriter, r *http.Request) {
	agreementID, ok := requireID(w, r, "agreement_id")
	if !ok {
		return
	}

	list, err := h.gtdSvc.GetByAdditionalAgreementID(r.Context(), agreementID)
	if err != nil {
		handleError(w, err)
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
	gtdID, ok := requireID(w, r, "gtd_id")
	if !ok {
		return
	}
	gtd, err := h.gtdSvc.GetByID(r.Context(), gtdID)
	if err != nil {
		handleError(w, err)
		return
	}
	if gtd == nil {
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
// @Param invoice_id path int true "ID инвойса"
// @Param gtd_number formData string true "Номер ГТД"
// @Param gtd_date formData string true "Дата ГТД (YYYY-MM-DD или DD.MM.YYYY)"
// @Param gtd_amount formData number true "Сумма ГТД (в валюте ГТД)"
// @Param gtd_currency formData string true "Валюта ГТД (например USD, EUR, TJS)"
// @Param hs_code formData string false "Код ТН ВЭД (HS CODE) (необязательно, по умолчанию 'Нет кода')"
// @Param sender_name formData string true "Отправитель"
// @Param sender_bank formData string true "Банк отправителя"
// @Param sender_country formData string true "Страна отправителя"
// @Param destination_country formData string false "Страна поступления товара (необязательно)"
// @Param document_type formData string true "Тип документа (gtd или act)"
// @Param document formData file true "Файл документа ГТД (.pdf)"
// @Success 201 {object} domain.GTD
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/gtd [post]
func (h *InvoiceHandler) CreateGTD(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	contractID, ok := requireID(w, r, "contract_id")
	if !ok {
		return
	}
	v := NewFormValidator(r, 32<<20)
	invoiceIDStr := mux.Vars(r)["invoice_id"]
	var invoiceID int64
	var err error
	if invoiceIDStr != "" && !strings.HasPrefix(invoiceIDStr, "{") {
		invoiceID, err = strconv.ParseInt(invoiceIDStr, 10, 64)
	}
	if invoiceID <= 0 {
		if formVal := strings.TrimSpace(r.FormValue("invoice_id")); formVal != "" {
			invoiceID, err = strconv.ParseInt(formVal, 10, 64)
		} else if qVal := strings.TrimSpace(r.URL.Query().Get("invoice_id")); qVal != "" {
			invoiceID, err = strconv.ParseInt(qVal, 10, 64)
		}
	}
	v.RequirePositiveID(invoiceID, "invoice_id")
	v.RequireStrings(gtdRequiredFields)
	gtdDate := v.Date("gtd_date")
	gtdAmount := v.Float("gtd_amount")
	destinationCountry := getFormValueFallback(r, "destination_country", "country_of_destination", "country")

	if v.Respond(w) {
		return
	}
	closesAmount := gtdAmount
	if v := r.FormValue("closes_amount"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			closesAmount = f
		}
	}
	hsCode := strings.TrimSpace(r.FormValue("hs_code"))
	if hsCode == "" {
		hsCode = domain.DefaultHSCode
	}
	addlID := parseOptionalAgreementID(r)
	docPathStr, err := saveUploadedFile(r, "document", "uploads/gtd", true)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}
	docPath := &docPathStr

	docTypeStr := strings.ToLower(strings.TrimSpace(r.FormValue("document_type")))
	docType := domain.DocumentTypeGTD
	if docTypeStr == domain.DocumentTypeAct || docTypeStr == "акт" || docTypeStr == "акт выполненных работ" {
		docType = domain.DocumentTypeAct
	}

	gtdCurrencyStr := strings.ToUpper(strings.TrimSpace(r.FormValue("gtd_currency")))

	g := domain.GTD{
		ContractID:            contractID,
		AdditionalAgreementID: addlID,
		InvoiceID:             invoiceID,
		DocumentType:          docType,
		GTDNumber:             strings.TrimSpace(r.FormValue("gtd_number")),
		GTDAmount:             gtdAmount,
		GTDCurrency:           &gtdCurrencyStr,
		GTDDate:               gtdDate,
		ClosesAmount:          closesAmount,
		HSCode:                hsCode,
		DestinationCountry:    destinationCountry,
		DocumentPath:          docPath,
		CreatedBy:             login,
		SenderName:            strings.TrimSpace(r.FormValue("sender_name")),
		SenderBank:            strings.TrimSpace(r.FormValue("sender_bank")),
		SenderCountry:         strings.TrimSpace(r.FormValue("sender_country")),
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
// @Description Редактирование ГТД или акта. Доступно: Сотрудник валютного контроля (при наличии разрешения), Комплаенс, Администратор.
// @Tags GTD
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param gtd_id path int true "ID ГТД"
// @Param gtd_number formData string true "Номер ГТД"
// @Param gtd_amount formData number true "Сумма ГТД"
// @Param gtd_currency formData string true "Валюта ГТД"
// @Param gtd_date formData string true "Дата ГТД (YYYY-MM-DD или DD.MM.YYYY)"
// @Param hs_code formData string false "Код ТН ВЭД (HS CODE) (необязательно, по умолчанию 'Нет кода')"
// @Param destination_country formData string true "Страна поступления товара"
// @Param document_type formData string true "Тип документа"
// @Param sender_name formData string false "Отправитель"
// @Param sender_bank formData string false "Банк отправителя"
// @Param sender_country formData string false "Страна отправителя"
// @Param document formData file false "Новый файл ГТД (.pdf)"
// @Success 200 {object} domain.GTD
// @Failure 400 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 404 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/gtd/{gtd_id} [put]
func (h *InvoiceHandler) UpdateGTD(w http.ResponseWriter, r *http.Request) {
	gtdID, ok := requireID(w, r, "gtd_id")
	if !ok {
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	existing, err := h.gtdSvc.GetByID(r.Context(), gtdID)
	if err != nil {
		handleError(w, err)
		return
	}
	if existing == nil {
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
			existing.ClosesAmount = f
		}
	}
	if v := r.FormValue("closes_amount"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil && f > 0 {
			existing.ClosesAmount = f
		}
	}
	if v := strings.TrimSpace(r.FormValue("gtd_currency")); v != "" {
		upper := strings.ToUpper(v)
		existing.GTDCurrency = &upper
	}
	if _, ok := r.Form["hs_code"]; ok {
		v := strings.TrimSpace(r.FormValue("hs_code"))
		if v == "" {
			existing.HSCode = domain.DefaultHSCode
		} else {
			existing.HSCode = v
		}
	} else if strings.TrimSpace(existing.HSCode) == "" {
		existing.HSCode = domain.DefaultHSCode
	}
	if v := getFormValueFallback(r, "destination_country", "country_of_destination", "country"); v != "" {
		existing.DestinationCountry = v
	}
	if v := strings.ToLower(strings.TrimSpace(r.FormValue("document_type"))); v != "" {
		if v == domain.DocumentTypeAct || v == "акт" || v == "акт выполненных работ" {
			existing.DocumentType = domain.DocumentTypeAct
		} else {
			existing.DocumentType = domain.DocumentTypeGTD
		}
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
	if pathStr, err := saveUploadedFile(r, "document", "uploads/gtd", false); err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	} else if pathStr != "" {
		existing.DocumentPath = &pathStr
	}

	updated, err := h.gtdSvc.Update(r.Context(), existing.ID, *existing)
	if err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "UPDATE", "gtd", &updated.ID, "Обновление ГТД № "+updated.GTDNumber)

	writeJSON(w, http.StatusOK, updated)
}

// @Summary Удаление ГТД (в корзину)
// @Description Помещает ГТД или акт в корзину (soft delete). Доступно: Сотрудники валютного контроля, Комплаенс, Администратор.
// @Tags GTD
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
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/gtd/{gtd_id} [delete]
func (h *InvoiceHandler) DeleteGTD(w http.ResponseWriter, r *http.Request) {
	gtdID, ok := requireID(w, r, "gtd_id")
	if !ok {
		return
	}
	if err := h.gtdSvc.SoftDelete(r.Context(), gtdID); err != nil {
		handleError(w, err)
		return
	}
	LogUserAction(r, "DELETE", "gtd", &gtdID, "Удаление ГТД")
	writeJSON(w, http.StatusOK, map[string]string{"message": "ГТД успешно удалена"})
}
