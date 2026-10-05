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

	if isSoleProprietor {
		clientTypeName = "Индивидуальный предприниматель"
	} else if isIndividual {
		clientTypeName = "Физическое лицо"
	}

	return dto.CompanyResponse{
		ID:         c.ID,
		LLC:        c.LLC,
		INN:        innVal,
		ClientType: clientTypeName,
		Phones:     c.GetPhones(),
		BranchID:   c.BranchID,
		CreatedBy:  c.CreatedBy,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
		Creator:    c.Creator,
	}
}

// @Summary Создание контрагента
// @Description Создаёт карточку контрагента (клиента банка). Операционист вводит только ИНН. Система автоматически:
// @Description 1. Проверяет наличие ИНН в базе (защита от дубликатов).
// @Description 2. Ищет клиента в АБС банка по ИНН.
// @Description 3. Записывает наименование/ФИО из АБС в название (llc).
// @Tags Companies
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path int true "ID филиала"
// @Param request body dto.CreateCompanyRequest true "ИНН клиента"
// @Success 201 {object} dto.CompanyResponse
// @Failure 400 {object} CommonError "ИНН не найден в АБС или клиент уже существует"
// @Failure 401 {object} CommonError "Не авторизован"
// @Failure 403 {object} CommonError "Доступ запрещён"
// @Failure 500 {object} CommonError "Внутренняя ошибка сервера"
// @Router /api/branches/{id}/dashboard/companies [post]
func (h *CounterpartyHandler) CreateCompany(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())

	branchID, ok := requireID(w, r, "id")
	if !ok {
		return
	}

	var req dto.CreateCompanyRequest
	if err := decodeJSON(r, &req); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	inn := strings.TrimSpace(req.INN)
	if inn == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле inn обязательно"})
		return
	}

	created, err := h.service.CreateByINNFromABS(r.Context(), login, inn, int(branchID))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}

	res := toCompanyResponse(created)
	LogUserAction(r, "CREATE", "company", &created.ID, fmt.Sprintf("Создание клиента: %s (ИНН: %s)", created.LLC, res.INN))

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
		handleError(w, err)
		return
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

	LogUserAction(r, "UPDATE", "company", &updated.ID, "Обновление карточки клиента: "+updated.LLC)

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

// @Summary Карточка клиента (получить по ID)
// @Description Возвращает полную информацию по карточке клиента
// @Tags Companies
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Success 200 {object} dto.CompanyResponse
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 404 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id} [get]
func (h *CounterpartyHandler) GetCompanyDetail(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireID(w, r, "company_id")
	if !ok {
		return
	}

	existing, err := h.service.GetByID(r.Context(), companyID)
	if err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "VIEW", "company", &companyID, "Просмотр карточки клиента")
	writeJSON(w, http.StatusOK, toCompanyResponse(existing))
}
