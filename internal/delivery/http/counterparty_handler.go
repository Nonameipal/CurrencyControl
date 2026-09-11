package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	_ "CurrencyControl/internal/abs"
	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/errs"
	"CurrencyControl/internal/service/ports"

	"github.com/gorilla/mux"
)

type CounterpartyHandler struct {
	service ports.CounterpartyService
}

func NewCounterpartyHandler(service ports.CounterpartyService) *CounterpartyHandler {
	return &CounterpartyHandler{
		service: service,
	}
}

// @Summary Поиск данных клиента в АБС банка по ИНН
// @Description Запрашивает данные клиента из АБС банка (Colvir) по его ИНН. Проверяет, нет ли уже такого клиента в базе, и возвращает полное наименование, тип, телефоны, счета и операциониста.
// @Tags Companies
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param inn query string true "ИНН клиента"
// @Success 200 {object} abs.ABSClientInfo
// @Failure 400 {object} CommonError "Клиент уже существует или неверный ИНН"
// @Failure 401 {object} CommonError "Не авторизован"
// @Failure 500 {object} CommonError "Ошибка АБС"
// @Router /api/branches/{id}/dashboard/companies/abs-lookup [get]
func (h *CounterpartyHandler) ABSLookup(w http.ResponseWriter, r *http.Request) {
	inn := strings.TrimSpace(r.URL.Query().Get("inn"))
	if inn == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Параметр inn обязателен"})
		return
	}

	exists, err := h.service.CheckExistsByINN(r.Context(), inn)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: err.Error()})
		return
	}
	if exists {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: fmt.Sprintf("Клиент с ИНН '%s' уже зарегистрирован в базе данных", inn)})
		return
	}

	info, err := h.service.ABSLookup(r.Context(), inn)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, CommonError{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, info)
}

// @Summary Создание карточки клиента (ЧДММ / юр.лицо / физ.лицо)
// @Description Создает карточку клиента с автоматической фиксацией автора и проверкой на дубликаты по ИНН. При необходимости наименование и тип клиента могут автоматически подтягиваться из АБС.
// @Tags Companies
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path int true "ID филиала"
// @Param request body dto.CreateCompanyRequest true "Данные для создания карточки клиента"
// @Success 201 {object} dto.CompanyResponse
// @Failure 400 {object} CommonError "Обязательные поля не заполнены или клиент с таким ИНН уже существует"
// @Failure 401 {object} CommonError "Не авторизован"
// @Failure 403 {object} CommonError "Доступ запрещен"
// @Failure 500 {object} CommonError "Внутренняя ошибка сервера"
// @Router /api/branches/{id}/dashboard/companies [post]
func (h *CounterpartyHandler) Create(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}

	branchStr := mux.Vars(r)["id"]
	branchID, err := strconv.Atoi(branchStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID филиала"})
		return
	}

	var req dto.CreateCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	inn := strings.TrimSpace(req.INN)
	if inn == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле inn обязательно"})
		return
	}

	// 1. Проверка на дубликат по ИНН
	exists, err := h.service.CheckExistsByINN(r.Context(), inn)
	if err != nil {
		handleError(w, err)
		return
	}
	if exists {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: fmt.Sprintf("Клиент с ИНН '%s' уже зарегистрирован в базе данных", inn)})
		return
	}

	name := strings.TrimSpace(req.Name)
	if name == "" {
		name = strings.TrimSpace(req.LLC)
	}

	clientType := normalizeClientType(req.ClientType, inn)

	c := domain.Counterparty{
		Name:       name,
		INN:        &inn,
		BranchID:   branchID,
		ClientType: clientType,
		CreatedBy:  login,
	}
	c.SetPhones(req.Phones)
	c.SetAccounts(req.Accounts)

	created, err := h.service.Create(r.Context(), login, c)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}

	innVal := ""
	if created.INN != nil {
		innVal = *created.INN
	}

	res := dto.CompanyResponse{
		ID:         created.ID,
		Name:       created.Name,
		LLC:        created.Name,
		INN:        innVal,
		ClientType: created.ClientType,
		Phones:     created.GetPhones(),
		Accounts:   created.GetAccounts(),
		BranchID:   created.BranchID,
		CreatedBy:  created.CreatedBy,
		CreatedAt:  created.CreatedAt,
		UpdatedAt:  created.UpdatedAt,
	}

	LogUserAction(r, "CREATE", "company", &created.ID, fmt.Sprintf("Создание карточки клиента: %s (ИНН: %s, тип: %s)", created.Name, innVal, created.ClientType))

	writeJSON(w, http.StatusCreated, res)
}

// @Summary Редактирование карточки клиента
// @Description Позволяет администратору обновить данные карточки клиента
// @Tags Admin
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
// @Router /admin/branches/{id}/dashboard/companies/{company_id} [put]
func (h *CounterpartyHandler) Update(w http.ResponseWriter, r *http.Request) {
	companyIDStr := mux.Vars(r)["company_id"]
	companyID, err := strconv.ParseInt(companyIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID компании"})
		return
	}

	var req dto.UpdateCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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

	innVal := ""
	if updated.INN != nil {
		innVal = *updated.INN
	}

	res := dto.CompanyResponse{
		ID:         updated.ID,
		Name:       updated.Name,
		LLC:        updated.Name,
		INN:        innVal,
		ClientType: updated.ClientType,
		Phones:     updated.GetPhones(),
		Accounts:   updated.GetAccounts(),
		BranchID:   updated.BranchID,
		CreatedBy:  updated.CreatedBy,
		CreatedAt:  updated.CreatedAt,
		UpdatedAt:  updated.UpdatedAt,
	}

	LogUserAction(r, "UPDATE", "company", &updated.ID, "Обновление карточки клиента: "+updated.Name)

	writeJSON(w, http.StatusOK, res)
}

// @Summary Удаление карточки клиента
// @Description Позволяет администратору удалить карточку клиента (soft delete)
// @Tags Admin
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Success 200 {object} map[string]string "Сообщение об успешном удалении"
// @Failure 400 {object} CommonError
// @Failure 403 {object} CommonError "Доступ запрещен"
// @Failure 500 {object} CommonError
// @Router /admin/branches/{id}/dashboard/companies/{company_id} [delete]
func (h *CounterpartyHandler) Delete(w http.ResponseWriter, r *http.Request) {
	companyIDStr := mux.Vars(r)["company_id"]
	companyID, err := strconv.ParseInt(companyIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID компании"})
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
