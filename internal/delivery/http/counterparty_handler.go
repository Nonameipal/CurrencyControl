package http

import (
	"fmt"
	"net/http"
	"strings"

	"CurrencyControl/internal/abs"
	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/errs"
	"CurrencyControl/internal/service/ports"
)

type CounterpartyHandler struct {
	service ports.CounterpartyService
	abs     abs.ABSClient
}

func NewCounterpartyHandler(service ports.CounterpartyService, absClient abs.ABSClient) *CounterpartyHandler {
	return &CounterpartyHandler{service: service, abs: absClient}
}

func toCompanyResponse(c domain.Counterparty) dto.CompanyResponse {
	innVal := ""
	if c.INN != nil {
		innVal = *c.INN
	}

	lowerType := strings.ToLower(c.ClientType)
	isSoleProprietor := c.ClientType == domain.ClientTypeSoleProprietor || strings.Contains(lowerType, "предприниматель") || strings.Contains(lowerType, "ип")
	isIndividual := c.ClientType == domain.ClientTypeIndividual || strings.Contains(lowerType, "физ")
	clientTypeName := "Юридическое лицо"
	displayName := ""
	displayLLC := ""

	if isSoleProprietor {
		clientTypeName = "Индивидуальный предприниматель"
		displayName = c.Name
		displayLLC = c.LLC
	} else if isIndividual {
		clientTypeName = "Физическое лицо"
		displayName = c.Name
		displayLLC = c.LLC
	} else {
		displayLLC = c.LLC
		if displayLLC == "" {
			displayLLC = c.Name
		}
		displayName = ""
	}

	return dto.CompanyResponse{
		ID:         c.ID,
		Name:       displayName,
		LLC:        displayLLC,
		INN:        innVal,
		ClientType: clientTypeName,
		Phones:     c.GetPhones(),
		BranchID:   c.BranchID,
		CreatedBy:  c.CreatedBy,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
}

// @Summary Создание контрагента: Юридическое лицо
// @Description Создаёт карточку юридического лица. Операционист вводит только ИНН. Система автоматически:
// @Description 1. Проверяет наличие ИНН в базе (защита от дубликатов).
// @Description 2. Ищет клиента в АБС банка по ИНН.
// @Description 3. Записывает наименование организации из АБС в ЧДММ (llc), тип "Юридическое лицо", телефоны. Поле name (ФИО) убирается.
// @Tags Companies
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path int true "ID филиала"
// @Param request body dto.CreateLegalEntityRequest true "ИНН юридического лица"
// @Success 201 {object} dto.CompanyResponse
// @Failure 400 {object} CommonError "ИНН не найден в АБС или клиент уже существует"
// @Failure 401 {object} CommonError "Не авторизован"
// @Failure 403 {object} CommonError "Доступ запрещён"
// @Failure 500 {object} CommonError "Внутренняя ошибка сервера"
// @Router /api/branches/{id}/dashboard/companies/legal-entity [post]
func (h *CounterpartyHandler) CreateLegalEntity(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())

	branchID, ok := requireID(w, r, "id")
	if !ok {
		return
	}

	var req dto.CreateLegalEntityRequest
	if err := decodeJSON(r, &req); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	inn := strings.TrimSpace(req.INN)
	if inn == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле inn обязательно"})
		return
	}

	created, err := h.service.CreateLegalEntityFromABS(r.Context(), login, inn, int(branchID))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}

	res := toCompanyResponse(created)
	LogUserAction(r, "CREATE", "company", &created.ID, fmt.Sprintf("Создание ЮЛ: %s (ИНН: %s)", created.LLC, res.INN))

	writeJSON(w, http.StatusCreated, res)
}

// @Summary Создание контрагента: Физическое лицо
// @Description Создаёт карточку физического лица. Операционист вводит ИНН и ЧДММ (название компании). Система автоматически:
// @Description 1. Проверяет наличие ИНН в базе (защита от дубликатов).
// @Description 2. Ищет клиента в АБС банка по ИНН.
// @Description 3. Записывает ФИО из АБС в name, введённое название компании в ЧДММ (llc), тип "Физическое лицо", телефоны.
// @Tags Companies
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path int true "ID филиала"
// @Param request body dto.CreateIndividualRequest true "ИНН и название компании (ЧДММ)"
// @Success 201 {object} dto.CompanyResponse
// @Failure 400 {object} CommonError "ИНН не найден в АБС или клиент уже существует"
// @Failure 401 {object} CommonError "Не авторизован"
// @Failure 403 {object} CommonError "Доступ запрещён"
// @Failure 500 {object} CommonError "Внутренняя ошибка сервера"
// @Router /api/branches/{id}/dashboard/companies/individual [post]
func (h *CounterpartyHandler) CreateIndividual(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())

	branchID, ok := requireID(w, r, "id")
	if !ok {
		return
	}

	var req dto.CreateIndividualRequest
	if err := decodeJSON(r, &req); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	inn := strings.TrimSpace(req.INN)
	if inn == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле inn обязательно"})
		return
	}
	llc := strings.TrimSpace(req.LLC)
	if llc == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле llc (название компании) обязательно для физического лица"})
		return
	}

	created, err := h.service.CreateIndividualFromABS(r.Context(), login, inn, llc, int(branchID))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}

	res := toCompanyResponse(created)
	LogUserAction(r, "CREATE", "company", &created.ID, fmt.Sprintf("Создание ФЛ: %s, ЧДММ: %s (ИНН: %s)", created.Name, created.LLC, res.INN))

	writeJSON(w, http.StatusCreated, res)
}

// @Summary Создание контрагента: Индивидуальный предприниматель
// @Description Создаёт карточку индивидуального предпринимателя. Операционист вводит ИНН и ЧДММ (название компании/ИП). Система автоматически:
// @Description 1. Проверяет наличие ИНН в базе (защита от дубликатов).
// @Description 2. Ищет клиента в АБС банка по ИНН.
// @Description 3. Записывает ФИО из АБС в name, введённое название в ЧДММ (llc), тип "Индивидуальный предприниматель", телефоны.
// @Tags Companies
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path int true "ID филиала"
// @Param request body dto.CreateSoleProprietorRequest true "ИНН и название компании/ИП (ЧДММ)"
// @Success 201 {object} dto.CompanyResponse
// @Failure 400 {object} CommonError "ИНН не найден в АБС или клиент уже существует"
// @Failure 401 {object} CommonError "Не авторизован"
// @Failure 403 {object} CommonError "Доступ запрещён"
// @Failure 500 {object} CommonError "Внутренняя ошибка сервера"
// @Router /api/branches/{id}/dashboard/companies/sole-proprietor [post]
func (h *CounterpartyHandler) CreateSoleProprietor(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())

	branchID, ok := requireID(w, r, "id")
	if !ok {
		return
	}

	var req dto.CreateSoleProprietorRequest
	if err := decodeJSON(r, &req); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	inn := strings.TrimSpace(req.INN)
	if inn == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле inn обязательно"})
		return
	}
	llc := strings.TrimSpace(req.LLC)
	if llc == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле  (название компании) обязательно для индивидуального предпринимателя"})
		return
	}

	created, err := h.service.CreateSoleProprietorFromABS(r.Context(), login, inn, llc, int(branchID))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}

	res := toCompanyResponse(created)
	LogUserAction(r, "CREATE", "company", &created.ID, fmt.Sprintf("Создание ИП: %s, ЧДММ: %s (ИНН: %s)", created.Name, created.LLC, res.INN))

	writeJSON(w, http.StatusCreated, res)
}

// @Summary Редактирование карточки клиента
// @Description Обновление данных карточки клиента. Доступно: Комплаенс, Администратор.
// @Tags Companies
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param request body dto.UpdateCompanyRequest true "Данные для обновления"
// @Success 200 {object} dto.CompanyResponse
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 404 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id} [put]
func (h *CounterpartyHandler) Update(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireID(w, r, "company_id")
	if !ok {
		return
	}

	var req dto.UpdateCompanyRequest
	if err := decodeJSON(r, &req); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	existing, err := h.service.GetByID(r.Context(), companyID)
	if err != nil {
		writeJSON(w, http.StatusNotFound, CommonError{Error: "Компания не найдена"})
		return
	}

	if reqName := strings.TrimSpace(req.Name); reqName != "" {
		existing.Name = reqName
	}

	if reqLLC := strings.TrimSpace(req.LLC); reqLLC != "" {
		existing.LLC = reqLLC
	}

	if req.INN != "" && (existing.INN == nil || *existing.INN != req.INN) {
		cleanINN := strings.TrimSpace(req.INN)
		exists, err := h.service.CheckExistsByINN(r.Context(), cleanINN)
		if err == nil && exists {
			writeJSON(w, http.StatusBadRequest, CommonError{Error: fmt.Sprintf("Клиент с ИНН '%s' уже зарегистрирован в базе данных", cleanINN)})
			return
		}
		existing.INN = &cleanINN
	}

	if req.ClientType != "" {
		innStr := ""
		if existing.INN != nil {
			innStr = *existing.INN
		}
		existing.ClientType = normalizeClientType(req.ClientType, innStr)
	}

	if req.Phones != nil {
		existing.SetPhones(req.Phones)
	}

	updated, err := h.service.Update(r.Context(), companyID, existing)
	if err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "UPDATE", "company", &updated.ID, "Обновление карточки клиента: "+updated.Name)

	writeJSON(w, http.StatusOK, toCompanyResponse(updated))
}

// @Summary Удаление карточки клиента (в корзину)
// @Description Удаление карточки клиента в корзину (soft delete). Доступно: Сотрудники Валютного контроля, Комплаенс, Администратор.
// @Tags Companies
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Success 200 {object} map[string]string "Сообщение об успешном удалении"
// @Failure 400 {object} CommonError
// @Failure 403 {object} CommonError "Доступ запрещен"
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id} [delete]
func (h *CounterpartyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireID(w, r, "company_id")
	if !ok {
		return
	}

	if err := h.service.SoftDelete(r.Context(), companyID); err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "DELETE", "company", &companyID, "Удаление карточки клиента")

	writeJSON(w, http.StatusOK, map[string]string{"message": "Компания успешно удалена"})
}

func normalizeClientType(rawType, inn string) string {
	rawType = strings.TrimSpace(rawType)
	lower := strings.ToLower(rawType)
	switch {
	case lower == domain.ClientTypeSoleProprietor || strings.Contains(lower, "предприниматель") || strings.Contains(lower, "ип"):
		return domain.ClientTypeSoleProprietor
	case lower == domain.ClientTypeIndividual || strings.Contains(lower, "физ"):
		return domain.ClientTypeIndividual
	case lower == domain.ClientTypeLegalEntity || strings.Contains(lower, "юр"):
		return domain.ClientTypeLegalEntity
	}
	if len(inn) == 14 {
		return domain.ClientTypeIndividual
	}
	return domain.ClientTypeLegalEntity
}

// @Summary Поиск клиента по ИНН в CBS
// @Description Возвращает данные клиента из банковской системы (CBS) по ИНН: ФИО, тип клиента, телефон.
// @Tags Companies
// @Security ApiKeyAuth
// @Produce json
// @Param inn query string true "ИНН клиента"
// @Success 200 {object} abs.ABSClientInfo
// @Failure 400 {object} CommonError "ИНН не указан или клиент не найден"
// @Failure 503 {object} CommonError "CBS недоступен"
// @Router /api/clients/by-inn [get]
func (h *CounterpartyHandler) LookupByINN(w http.ResponseWriter, r *http.Request) {
	inn := strings.TrimSpace(r.URL.Query().Get("inn"))
	if inn == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Параметр 'inn' обязателен"})
		return
	}

	info, err := h.abs.GetClientByINN(r.Context(), inn)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, info)
}
