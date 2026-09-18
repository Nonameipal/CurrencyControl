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
	llc := c.LLC
	if llc == "" {
		llc = c.Name
	}
	return dto.CompanyResponse{
		ID:         c.ID,
		Name:       c.Name,
		LLC:        llc,
		INN:        innVal,
		ClientType: c.ClientType,
		Phones:     c.GetPhones(),
		Accounts:   c.GetAccounts(),
		BranchID:   c.BranchID,
		CreatedBy:  c.CreatedBy,
		CreatedAt:  c.CreatedAt,
		UpdatedAt:  c.UpdatedAt,
	}
}

// @Summary Создание карточки контрагента (ЧДММ)
// @Description Создаёт карточку ЧДММ. Доступно: Операционный сотрудник, Комплаенс, Администратор. Принимает только название (llc) и ИНН. Система автоматически:
// @Description 1. Проверяет наличие ИНН в базе (защита от дубликатов).
// @Description 2. Ищет клиента в АБС банка — если не найден, возвращает ошибку.
// @Description 3. Подставляет из АБС: тип клиента (ЮЛ / ФЛ), телефоны, счета, полное наименование (если llc не передан).
// @Tags Companies
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path int true "ID филиала"
// @Param request body dto.CreateCompanyRequest true "Название и ИНН"
// @Success 201 {object} dto.CompanyResponse
// @Failure 400 {object} CommonError "ИНН не найден в АБС или клиент уже существует"
// @Failure 401 {object} CommonError "Не авторизован"
// @Failure 403 {object} CommonError "Доступ запрещён"
// @Failure 500 {object} CommonError "Внутренняя ошибка сервера"
// @Router /api/branches/{id}/dashboard/companies [post]
func (h *CounterpartyHandler) Create(w http.ResponseWriter, r *http.Request) {
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

	created, err := h.service.CreateFromABS(r.Context(), login, req.LLC, inn, int(branchID))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}

	res := toCompanyResponse(created)
	LogUserAction(r, "CREATE", "company", &created.ID, fmt.Sprintf("Создание ЧДММ: %s (ИНН: %s, тип: %s)", created.Name, res.INN, created.ClientType))

	writeJSON(w, http.StatusCreated, res)
}

// @Summary Редактирование карточки клиента
// @Description Обновление данных карточки клиента. Доступно: Операционный сотрудник, Комплаенс, Администратор.
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

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = strings.TrimSpace(req.LLC)
	}
	if name != "" {
		existing.Name = name
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
	if req.Accounts != nil {
		existing.SetAccounts(req.Accounts)
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
	switch strings.ToLower(rawType) {
	case domain.ClientTypeIndividual, "физическое лицо", "физ. лицо", "физлицо", "фл":
		return domain.ClientTypeIndividual
	case domain.ClientTypeLegalEntity, "юридическое лицо", "юр. лицо", "юрлицо", "юл":
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
