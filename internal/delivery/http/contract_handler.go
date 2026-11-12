package http

import (
	"net/http"
	"strconv"
	"strings"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/errs"
	"CurrencyControl/internal/service/ports"

	"github.com/gorilla/mux"
)

type ContractHandler struct {
	service ports.ContractService
}

func NewContractHandler(service ports.ContractService) *ContractHandler {
	return &ContractHandler{service: service}
}



// @Summary Создание контракта
// @Description Создает новый контракт
// @Tags Contracts
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании (ЧДММ)"
// @Param contract_number formData string true "Номер контракта"
// @Param contract_date formData string true "Дата контракта (YYYY-MM-DD или DD.MM.YYYY)"
// @Param delivery_date formData string true "Дата поставки (YYYY-MM-DD или DD.MM.YYYY)"
// @Param contract_end_date formData string true "Дата окончания контракта (YYYY-MM-DD или DD.MM.YYYY)"
// @Param return_days formData integer true "Срок возврата денежных средств в днях (число дней, > 0)"
// @Param total_amount formData number true "Сумма контракта"
// @Param contract_currency formData string true "Валюта контракта"
// @Param sender_name formData string true "Отправитель"
// @Param receiver_name formData string true "Получатель"
// @Param receiver_bank formData string true "Банк получатель"
// @Param receiver_country formData string true "Страна получателя"
// @Param subject formData string true "Предмет"
// @Param document formData file true "PDF документ (.pdf)"
// @Success 201 {object} domain.Contract
// @Failure 400 {object} map[string]string "Обязательные поля не заполнены"
// @Failure 401 {object} map[string]string "Не авторизован"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts [post]
func (h *ContractHandler) Create(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())

	branchStr := mux.Vars(r)["id"]
	branchID, err := strconv.Atoi(branchStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID филиала"})
		return
	}

	companyStr := mux.Vars(r)["company_id"]
	clientID, err := strconv.ParseInt(companyStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID компании"})
		return
	}

	v := NewFormValidator(r, 10<<20)
	v.RequirePositiveInt(branchID, "branch_id")
	v.RequirePositiveID(clientID, "client_id")
	v.RequireStrings(contractRequiredFields)

	contractDate := v.Date("contract_date")
	deliveryDate := v.Date("delivery_date")
	endDate := v.Date("contract_end_date")
	totalAmount := v.Float("total_amount")
	returnDaysVal := v.Int("return_days")
	contractCurrency := strings.ToUpper(v.String("contract_currency"))
	receiverCountry := v.String("receiver_country")

	if v.Respond(w) {
		return
	}

	currencyExists, err := h.service.CheckCurrency(r.Context(), contractCurrency)
	if err != nil {
		handleError(w, err)
		return
	}
	if !currencyExists {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Указанная валюта не найдена в справочнике"})
		return
	}

	countryExists, err := h.service.CheckCountry(r.Context(), receiverCountry)
	if err != nil {
		handleError(w, err)
		return
	}
	if !countryExists {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Указанная страна получателя не найдена в справочнике"})
		return
	}

	returnDays := &returnDaysVal

	contract := domain.Contract{
		ClientID:         &clientID,
		BranchID:         &branchID,
		ContractNumber:   r.FormValue("contract_number"),
		ContractDate:     *contractDate,
		DeliveryDate:     *deliveryDate,
		ContractEndDate:  endDate,
		ReturnDays:       returnDays,
		TotalAmount:      totalAmount,
		ContractCurrency: contractCurrency,
		Subject:          r.FormValue("subject"),
		CreatedBy:        login,
		ReceiverName:     strings.TrimSpace(r.FormValue("receiver_name")),
		ReceiverBank:     strings.TrimSpace(r.FormValue("receiver_bank")),
		ReceiverCountry:  receiverCountry,
		SenderName:       strings.TrimSpace(r.FormValue("sender_name")),
	}

	pathStr, err := saveUploadedFile(r, "document", "uploads/contracts", true)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}
	contract.DocumentPath = &pathStr

	created, err := h.service.Create(r.Context(), login, contract)
	if err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "CREATE", "contract", &created.ID, "Создание контракта № "+created.ContractNumber)

	writeJSON(w, http.StatusCreated, created)
}

// @Summary Получить список контрактов ЧДММ
// @Description Возвращает список контрактов для выбранной компании. Поддерживает поиск по сумме контракта.
// @Tags Contracts
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании (ЧДММ)"
// @Param amount query number false "Поиск по сумме контракта"
// @Success 200 {array} domain.Contract
// @Failure 400 {object} map[string]string "Неверные параметры"
// @Failure 401 {object} map[string]string "Не авторизован"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts [get]
func (h *ContractHandler) GetContractsByCompany(w http.ResponseWriter, r *http.Request) {
	clientID, ok := requireID(w, r, "company_id")
	if !ok {
		return
	}

	filter := dto.ContractFilter{}
	if rawAmount := strings.TrimSpace(r.URL.Query().Get("amount")); rawAmount != "" {
		cleanAmt := strings.ReplaceAll(rawAmount, " ", "")
		cleanAmt = strings.ReplaceAll(cleanAmt, ",", ".")
		amt, err := strconv.ParseFloat(cleanAmt, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный формат суммы для поиска"})
			return
		}
		filter.Amount = &amt
	}
	login := GetLoginFromContext(r.Context())
	contracts, err := h.service.GetByClientID(r.Context(), login, clientID, filter)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, contracts)
}
// @Summary Карточка контракта (получить по ID)
// @Description Возвращает полную информацию по карточке контракта
// @Tags Contracts
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Success 200 {object} domain.Contract
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 404 {object} map[string]string
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id} [get]
func (h *ContractHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	contractID, ok := requireID(w, r, "contract_id")
	if !ok {
		return
	}

	login := GetLoginFromContext(r.Context())
	contract, err := h.service.GetByID(r.Context(), login, contractID)
	if err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "VIEW", "contract", &contractID, "Просмотр карточки контракта")
	writeJSON(w, http.StatusOK, contract)
}


// @Summary Редактирование контракта
// @Description Редактирование данных контракта. Доступно: Валютный контроль (по разрешению Комплаенса), Комплаенс, Администратор.
// @Tags Contracts
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании (ЧДММ)"
// @Param contract_id path int true "ID контракта"
// @Param contract_number formData string false "Номер контракта"
// @Param contract_date formData string false "Дата контракта (YYYY-MM-DD или DD.MM.YYYY)"
// @Param delivery_date formData string false "Дата поставки (YYYY-MM-DD или DD.MM.YYYY)"
// @Param contract_end_date formData string false "Дата окончания контракта (YYYY-MM-DD или DD.MM.YYYY)"
// @Param return_days formData integer false "Срок возврата денежных средств в днях (число дней, > 0)"
// @Param total_amount formData number false "Сумма контракта"
// @Param contract_currency formData string false "Валюта контракта"
// @Param receiver_name formData string false "Получатель"
// @Param receiver_bank formData string false "Банк получатель"
// @Param receiver_country formData string false "Страна получателя"
// @Param subject formData string false "Предмет"
// @Param sender_name formData string false "Отправитель"
// @Param document formData file false "Новый PDF документ (опционально)"
// @Success 200 {object} domain.Contract
// @Failure 400 {object} map[string]string "Некорректный запрос"
// @Failure 403 {object} map[string]string "Доступ запрещен"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id} [put]
func (h *ContractHandler) Update(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	contractID, ok := requireID(w, r, "contract_id")
	if !ok {
		return
	}
	if err := r.ParseMultipartForm(32 << 20); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}
	existing, err := h.service.GetByID(r.Context(), login, contractID)
	if err != nil {
		handleError(w, err)
		return
	}
	if v := r.FormValue("contract_number"); v != "" {
		existing.ContractNumber = v
	}
	if v := r.FormValue("contract_date"); v != "" {
		if d := parseDate(v); d != nil {
			existing.ContractDate = *d
		} else {
			writeJSON(w, http.StatusBadRequest, CommonError{Error: "Неверный формат contract_date"})
			return
		}
	}
	if v := r.FormValue("delivery_date"); v != "" {
		if d := parseDate(v); d != nil {
			existing.DeliveryDate = *d
		} else {
			writeJSON(w, http.StatusBadRequest, CommonError{Error: "Неверный формат delivery_date"})
			return
		}
	}
	if v := r.FormValue("contract_end_date"); v != "" {
		if d := parseDate(v); d != nil {
			existing.ContractEndDate = d
		} else {
			writeJSON(w, http.StatusBadRequest, CommonError{Error: "Неверный формат contract_end_date"})
			return
		}
	}
	if rawReturn := strings.TrimSpace(r.FormValue("return_days")); rawReturn != "" {
		days, err := strconv.Atoi(rawReturn)
		if err != nil || days <= 0 {
			writeJSON(w, http.StatusBadRequest, CommonError{Error: "Срок возврата денежных средств должен быть целым числом больше 0"})
			return
		}
		existing.ReturnDays = &days
	}
	if v := r.FormValue("total_amount"); v != "" {
		f, err := strconv.ParseFloat(v, 64)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, CommonError{Error: "Неверный формат total_amount"})
			return
		}
		existing.TotalAmount = f
	}
	if v := r.FormValue("contract_currency"); v != "" {
		existing.ContractCurrency = strings.ToUpper(v)
	}
	if v := r.FormValue("subject"); v != "" {
		existing.Subject = v
	}
	if v := strings.TrimSpace(r.FormValue("receiver_name")); v != "" {
		existing.ReceiverName = v
	}
	if v := strings.TrimSpace(r.FormValue("receiver_bank")); v != "" {
		existing.ReceiverBank = v
	}
	if rc := strings.TrimSpace(r.FormValue("receiver_country")); rc != "" {
		countryExists, err := h.service.CheckCountry(r.Context(), rc)
		if err != nil {
			handleError(w, err)
			return
		}
		if !countryExists {
			writeJSON(w, http.StatusBadRequest, CommonError{Error: "Указанная страна получателя не найдена в справочнике"})
			return
		}
		existing.ReceiverCountry = rc
	}
	if v := strings.TrimSpace(r.FormValue("sender_name")); v != "" {
		existing.SenderName = v
	}
	if pathStr, err := saveUploadedFile(r, "document", "uploads/contracts", false); err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	} else if pathStr != "" {
		existing.DocumentPath = &pathStr
	}
	updated, err := h.service.Update(r.Context(), existing.ID, existing)
	if err != nil {
		handleError(w, err)
		return
	}
	LogUserAction(r, "UPDATE", "contract", &updated.ID, "Обновление контракта № "+updated.ContractNumber)
	writeJSON(w, http.StatusOK, updated)
}

// @Summary Удаление контракта (в корзину)
// @Description Удаление контракта в корзину (soft delete). Доступно: Сотрудники Валютного контроля, Комплаенс, Администратор.
// @Tags Contracts
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Success 200 {object} map[string]string "Сообщение об успешном удалении"
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string "Доступ запрещен"
// @Failure 500 {object} map[string]string
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id} [delete]
func (h *ContractHandler) Delete(w http.ResponseWriter, r *http.Request) {
	contractIDStr := mux.Vars(r)["contract_id"]
	contractID, err := strconv.ParseInt(contractIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID контракта"})
		return
	}

	if err := h.service.SoftDelete(r.Context(), contractID); err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "DELETE", "contract", &contractID, "Удаление контракта")

	writeJSON(w, http.StatusOK, map[string]string{"message": "Контракт успешно удален"})
}