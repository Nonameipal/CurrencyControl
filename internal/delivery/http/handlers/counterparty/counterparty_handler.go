package counterparty

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

	clientTypeName := "Юридическое лицо"
	switch c.ClientType {
	case domain.ClientTypeSoleProprietor:
		clientTypeName = "Индивидуальный предприниматель"
	case domain.ClientTypeIndividual:
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
// @Description Создаёт карточку контрагента (клиента банка). Операционист вводит только ИНН (ровно 9 цифр). Система автоматически:
// @Description 1. Проверяет наличие ИНН в базе (защита от дубликатов).
// @Description 2. Ищет клиента в АБС банка по ИНН.
// @Description 3. Записывает наименование/ФИО из АБС в название (llc).
// @Tags Companies
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path int true "ID филиала"
// @Param request body dto.CreateCompanyRequest true "ИНН клиента (9 цифр)"
// @Success 201 {object} dto.CompanyResponse
// @Failure 400 {object} CommonError "Некорректная длина ИНН, клиент не найден в АБС или уже существует"
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
	if err := domain.ValidateINN(inn); err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
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

func normalizeClientType(raw string) (string, bool) {
	norm := strings.ToLower(strings.TrimSpace(raw))
	switch norm {
	case "legal_entity", "юридическое лицо", "юр лицо", "юрлицо":
		return domain.ClientTypeLegalEntity, true
	case "sole_proprietor", "индивидуальный предприниматель", "ип":
		
		return domain.ClientTypeSoleProprietor, true
	case "individual", "физическое лицо", "физ лицо", "физлицо":
		return domain.ClientTypeIndividual, true
	default:
		return "", false
	}
}

// @Summary Редактирование карточки клиента
// @Description Обновление данных карточки клиента. Доступно: Комплаенс, Администратор, Операционист.
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
	diff := NewDiffTracker()
	if reqLLC := strings.TrimSpace(req.LLC); reqLLC != "" {
		diff.TrackString("Наименование клиента", existing.LLC, reqLLC)
		existing.LLC = reqLLC
	}
	cleanINN := strings.TrimSpace(req.INN)
	currentINN := ""
	if existing.INN != nil {
		currentINN = strings.TrimSpace(*existing.INN)
	}
	if cleanINN != "" && cleanINN != currentINN {
		if err := domain.ValidateINN(cleanINN); err != nil {
			writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
			return
		}
		exists, err := h.service.CheckExistsByINN(r.Context(), cleanINN)
		if err == nil && exists {
			writeJSON(w, http.StatusBadRequest, CommonError{Error: fmt.Sprintf("Клиент с ИНН '%s' уже зарегистрирован в базе данных", cleanINN)})
			return
		}
		diff.TrackStringPtr("ИНН", existing.INN, &cleanINN)
		existing.INN = &cleanINN
	}
	if req.ClientType != "" {
		ct, ok := normalizeClientType(req.ClientType)
		if !ok {
			writeJSON(w, http.StatusBadRequest, CommonError{Error: "Недопустимый тип клиента: допустимы legal_entity (Юридическое лицо), sole_proprietor (Индивидуальный предприниматель), individual (Физическое лицо)"})
			return
		}
		diff.TrackString("Тип клиента", string(existing.ClientType), string(ct))
		existing.ClientType = ct
	}   
	var phonesToSet []string
	hasPhones := false
	if len(req.Phones) > 0 {
		hasPhones = true
		for _, p := range req.Phones {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				phonesToSet = append(phonesToSet, trimmed)
			}
		}
	}
	if trimmedPhone := strings.TrimSpace(req.Phone); trimmedPhone != "" {
		hasPhones = true
		phonesToSet = append(phonesToSet, trimmedPhone)
	}
	if hasPhones {
		uniquePhones := make([]string, 0, len(phonesToSet))
		seen := make(map[string]struct{}, len(phonesToSet))
		for _, p := range phonesToSet {
			if _, ok := seen[p]; !ok {
				seen[p] = struct{}{}
				uniquePhones = append(uniquePhones, p)
			}
		}
		oldPhones := strings.Join(existing.GetPhones(), ", ")
		newPhones := strings.Join(uniquePhones, ", ")
		diff.TrackString("Телефоны", oldPhones, newPhones)
		existing.SetPhones(uniquePhones)
	}
	updated, err := h.service.Update(r.Context(), companyID, existing)
	if err != nil {
		handleError(w, err)
		return
	}
	details := fmt.Sprintf("Обновление карточки клиента: %s. Изменения: %s", updated.LLC, diff.String())
	LogUserAction(r, "UPDATE", "company", &updated.ID, details)
	writeJSON(w, http.StatusOK, toCompanyResponse(updated))
}

// @Summary Добавление номеров телефона компании
// @Description Добавляет новые номера телефона к существующим номерам компании без дублирования. Доступно: Операционист, Комплаенс, Администратор.
// @Tags Companies
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param request body dto.AddCompanyPhonesRequest true "Номера телефона для добавления"
// @Success 200 {object} dto.CompanyResponse
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 404 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/phones [post]
func (h *CounterpartyHandler) AddPhones(w http.ResponseWriter, r *http.Request) {
	companyID, ok := requireID(w, r, "company_id")
	if !ok {
		return
	}
	var req dto.AddCompanyPhonesRequest
	if err := decodeJSON(r, &req); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}
	var newPhones []string
	for _, p := range req.Phones {
		if trimmed := strings.TrimSpace(p); trimmed != "" {
			newPhones = append(newPhones, trimmed)
		}
	}
	if len(newPhones) == 0 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Не указан ни один номер телефона"})
		return
	}
	existing, err := h.service.GetByID(r.Context(), companyID)
	if err != nil {
		handleError(w, err)
		return
	}
	currentPhones := existing.GetPhones()
	existingSet := make(map[string]struct{}, len(currentPhones))
	for _, p := range currentPhones {
		existingSet[p] = struct{}{}
	}
	for _, p := range newPhones {
		if _, exists := existingSet[p]; !exists {
			existingSet[p] = struct{}{}
			currentPhones = append(currentPhones, p)
		}
	}
	existing.SetPhones(currentPhones)
	updated, err := h.service.Update(r.Context(), companyID, existing)
	if err != nil {
		handleError(w, err)
		return
	}
	details := fmt.Sprintf("Добавление номеров телефона для компании %s. Добавлены: %s; Итоговый список: %s", updated.LLC, strings.Join(newPhones, ", "), strings.Join(currentPhones, ", "))
	LogUserAction(r, "UPDATE", "company", &updated.ID, details)
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

// @Summary Поиск клиента по ИНН в CBS
// @Description Возвращает данные клиента из банковской системы (CBS) по ИНН (ровно 9 цифр): ФИО, тип клиента, телефон.
// @Tags Companies
// @Security ApiKeyAuth
// @Produce json
// @Param inn query string true "ИНН клиента (9 цифр)"
// @Success 200 {object} abs.ABSClientInfo
// @Failure 400 {object} CommonError "ИНН не указан, некорректной длины или клиент не найден"
// @Failure 503 {object} CommonError "CBS недоступен"
// @Router /api/clients/by-inn [get]
func (h *CounterpartyHandler) LookupByINN(w http.ResponseWriter, r *http.Request) {
	inn := strings.TrimSpace(r.URL.Query().Get("inn"))
	if inn == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Параметр 'inn' обязателен"})
		return
	}
	if err := domain.ValidateINN(inn); err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
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
