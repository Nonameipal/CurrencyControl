package http

import (
	"fmt"
	"github.com/gorilla/mux"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/errs"
	"CurrencyControl/internal/service/ports"
)

type ContractHandler struct {
	service ports.ContractService
}

func NewContractHandler(service ports.ContractService) *ContractHandler {
	return &ContractHandler{service: service}
}

// @Summary Поиск для дашборда
// @Description Возвращает список компаний с возможностью фильтрации по названию, сумме, ИНН и филиалу
// @Tags Dashboard
// @Produce json
// @Param Login header string true "Логин пользователя"
// @Param company_name query string false "Название компании"
// @Param amount query number false "Сумма контракта"
// @Param inn query string false "ИНН компании"
// @Param id path int true "ID филиала"
// @Success 200 {array} dto.DashboardSearchResult
// @Failure 400 {object} dto.ErrorResponse "Неверные параметры"
// @Failure 401 {object} dto.ErrorResponse "Не авторизован"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/branches/{id}/dashboard [get]
func (h *ContractHandler) Dashboard(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}

	req := dto.DashboardSearchRequest{}
	
	req.CompanyName = r.URL.Query().Get("company_name")
	req.INN = r.URL.Query().Get("inn")
	
	if amtStr := r.URL.Query().Get("amount"); amtStr != "" {
		if amt, err := strconv.ParseFloat(amtStr, 64); err == nil {
			req.Amount = amt
		}
	}
	
	branchStr := mux.Vars(r)["id"]
	branchID, err := strconv.Atoi(branchStr)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID филиала"})
		return
	}
	req.BranchID = branchID

	results, err := h.service.SearchDashboard(r.Context(), login, req)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, results)
}
// @Summary Создание контракта
// @Description Создает новый контракт
// @Tags Contracts
// @Accept multipart/form-data
// @Produce json
// @Param Login header string true "Логин пользователя"
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании (ЧДММ)"
// @Param contract_number formData string true "Номер контракта"
// @Param contract_name formData string true "Название контракта"
// @Param contract_date formData string true "Дата контракта (YYYY-MM-DD)"
// @Param delivery_date formData string true "Дата поставки (YYYY-MM-DD)"
// @Param contract_end_date formData string true "Дата окончания контракта (YYYY-MM-DD)"
// @Param delivery_term_days formData integer true "Срок поставки (дни)"
// @Param return_term_days formData integer true "Срок возврата (дни)"
// @Param total_amount formData number true "Сумма контракта"
// @Param contract_currency formData string true "Валюта контракта"
// @Param sender_account formData string true "Счет отправителя"
// @Param receiver_name formData string true "Наименование получателя"
// @Param receiver_account formData string true "Счет получателя"
// @Param receiver_country formData string true "Страна получателя"
// @Param subject formData string true "Предмет"
// @Param delivery_conditions formData string true "Условия поставки"
// @Param document formData file true "PDF документ"
// @Success 201 {object} domain.Contract
// @Failure 400 {object} dto.ErrorResponse "Обязательные поля не заполнены"
// @Failure 401 {object} dto.ErrorResponse "Не авторизован"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts [post]
func (h *ContractHandler) Create(w http.ResponseWriter, r *http.Request) {
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

	companyStr := mux.Vars(r)["company_id"]
	clientID, err := strconv.ParseInt(companyStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID компании"})
		return
	}

	err = r.ParseMultipartForm(10 << 20)
	if err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}
	deliveryTermDays, _ := strconv.Atoi(r.FormValue("delivery_term_days"))
	returnTermDays, _ := strconv.Atoi(r.FormValue("return_term_days"))
	totalAmount, _ := strconv.ParseFloat(r.FormValue("total_amount"), 64)

	contractDate, err := time.Parse(time.DateOnly, r.FormValue("contract_date"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Неверный формат contract_date. Ожидается YYYY-MM-DD"})
		return
	}
	
	deliveryDate, err := time.Parse(time.DateOnly, r.FormValue("delivery_date"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Неверный формат delivery_date. Ожидается YYYY-MM-DD"})
		return
	}
	
	parsedEndDate, err := time.Parse(time.DateOnly, r.FormValue("contract_end_date"))
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле contract_end_date обязательно. Ожидается формат YYYY-MM-DD"})
		return
	}

	contractCurrency := strings.ToUpper(strings.TrimSpace(r.FormValue("contract_currency")))
	currencyExists, err := h.service.CheckCurrency(r.Context(), contractCurrency)
	if err != nil {
		handleError(w, err)
		return
	}
	if !currencyExists {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Указанная валюта не найдена в справочнике"})
		return
	}

	senderAccount := strings.TrimSpace(r.FormValue("sender_account"))
	if senderAccount == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле sender_account обязательно"})
		return
	}

	receiverCountry := strings.TrimSpace(r.FormValue("receiver_country"))
	if receiverCountry == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Страна получателя обязательна"})
		return
	}
	exists, err := h.service.CheckCountry(r.Context(), receiverCountry)
	if err != nil {
		handleError(w, err)
		return
	}
	if !exists {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Указанная страна не найдена в справочнике"})
		return
	}
	
	if clientID <= 0 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле client_id обязательно"})
		return
	}
	if branchID <= 0 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле branch_id обязательно"})
		return
	}
	if strings.TrimSpace(r.FormValue("contract_number")) == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле contract_number обязательно"})
		return
	}
	if strings.TrimSpace(r.FormValue("contract_name")) == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле contract_name обязательно"})
		return
	}
	if strings.TrimSpace(r.FormValue("delivery_conditions")) == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле delivery_conditions обязательно"})
		return
	}
	if deliveryTermDays <= 0 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле delivery_term_days обязательно и должно быть больше 0"})
		return
	}
	if returnTermDays <= 0 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле return_term_days обязательно и должно быть больше 0"})
		return
	}
	if totalAmount <= 0 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле total_amount обязательно и должно быть больше 0"})
		return
	}
	if strings.TrimSpace(r.FormValue("receiver_name")) == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле receiver_name обязательно"})
		return
	}
	if strings.TrimSpace(r.FormValue("receiver_account")) == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле receiver_account обязательно"})
		return
	}
	if strings.TrimSpace(r.FormValue("subject")) == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле subject обязательно"})
		return
	}
	if strings.TrimSpace(r.FormValue("contract_end_date")) == ""{
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле contract_end_date обязательно"})
		return
	}

	contract := domain.Contract{
		ClientID:           &clientID,
		BranchID:           &branchID,
		ContractNumber:     r.FormValue("contract_number"),
		ContractName:       r.FormValue("contract_name"),
		ContractDate:       contractDate,
		DeliveryDate:       deliveryDate,
		ContractEndDate:    &parsedEndDate,
		DeliveryConditions: r.FormValue("delivery_conditions"),
		DeliveryTermDays:   deliveryTermDays,
		ReturnTermDays:     returnTermDays,
		TotalAmount:        totalAmount,
		ContractCurrency:   contractCurrency,
		SenderAccount:      senderAccount,
		ReceiverName:       r.FormValue("receiver_name"),
		ReceiverAccount:    r.FormValue("receiver_account"),
		ReceiverCountry:    receiverCountry,
		Subject:            r.FormValue("subject"),
	}

	file, handler, err := r.FormFile("document")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле 'document' с PDF файлом обязательно"})
		return
	}
	defer file.Close()

	os.MkdirAll("uploads/contracts", os.ModePerm)
	uniqueFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
	filePath := filepath.Join("uploads/contracts", uniqueFileName)
	
	dst, err := os.Create(filePath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: "Ошибка при сохранении файла на сервер"})
		return
	}
	defer dst.Close()
	io.Copy(dst, file)
	
	
	pathStr := filePath
	nameStr := handler.Filename
	contract.DocumentPath = &pathStr
	contract.OriginalDocumentName = &nameStr

	created, err := h.service.Create(r.Context(), login, contract)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

// @Summary Получить список контрактов ЧДММ
// @Description Возвращает список контрактов для выбранной компании. 
// @Tags Contracts
// @Produce json
// @Param Login header string true "Логин пользователя"
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании (ЧДММ)"
// @Success 200 {array} domain.Contract
// @Failure 400 {object} dto.ErrorResponse "Неверные параметры"
// @Failure 401 {object} dto.ErrorResponse "Не авторизован"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts [get]
func (h *ContractHandler) GetContractsByCompany(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}

	companyStr := mux.Vars(r)["company_id"]
	clientID, err := strconv.ParseInt(companyStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID компании"})
		return
	}

	contracts, err := h.service.GetByClientID(r.Context(), login, clientID)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, contracts)
}


// @Summary Уведомления дашборда (контракты с истекающим сроком)
// @Description Возвращает список контрактов, срок действия которых истекает в ближайшие 10 дней или уже истек.
// @Tags Dashboard
// @Produce json
// @Param Login header string true "Логин пользователя"
// @Param id path int true "ID филиала"
// @Success 200 {array} dto.NotificationResponse
// @Failure 401 {object} dto.ErrorResponse "Не авторизован"
// @Router /api/branches/{id}/dashboard/notifications [get]
func (h *ContractHandler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}

	branchID, err := strconv.Atoi(mux.Vars(r)["id"])
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID филиала"})
		return
	}

	notifications, err := h.service.GetExpiringContracts(r.Context(), branchID)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"notifications": notifications,
		"total":         len(notifications),
	})
}

// @Summary Редактирование контракта
// @Description Позволяет администратору обновить данные контракта
// @Tags Admin
// @Accept multipart/form-data
// @Produce json
// @Param Login header string true "Логин администратора (admin)"
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании (ЧДММ)"
// @Param contract_id path int true "ID контракта"
// @Param contract_number formData string false "Номер контракта"
// @Param contract_name formData string false "Название контракта"
// @Param contract_date formData string false "Дата контракта (YYYY-MM-DD)"
// @Param delivery_date formData string false "Дата поставки (YYYY-MM-DD)"
// @Param contract_end_date formData string false "Дата окончания контракта (YYYY-MM-DD)"
// @Param delivery_term_days formData integer false "Срок поставки (дни)"
// @Param return_term_days formData integer false "Срок возврата (дни)"
// @Param total_amount formData number false "Сумма контракта"
// @Param contract_currency formData string false "Валюта контракта"
// @Param sender_account formData string false "Счет отправителя"
// @Param receiver_name formData string false "Наименование получателя"
// @Param receiver_account formData string false "Счет получателя"
// @Param receiver_country formData string false "Страна получателя"
// @Param subject formData string false "Предмет"
// @Param delivery_conditions formData string false "Условия поставки"
// @Param document formData file false "Новый PDF документ (опционально)"
// @Success 200 {object} domain.Contract
// @Failure 400 {object} dto.ErrorResponse "Некорректный запрос"
// @Failure 403 {object} dto.ErrorResponse "Доступ запрещен"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id} [put]
func (h *ContractHandler) Update(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	contractID, err := strconv.ParseInt(mux.Vars(r)["contract_id"], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID контракта"})
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

	if v := r.FormValue("contract_number"); v != "" { existing.ContractNumber = v }
	if v := r.FormValue("contract_name"); v != "" { existing.ContractName = v }
	if v := r.FormValue("contract_date"); v != "" {
		if d, err := time.Parse(time.DateOnly, v); err == nil { existing.ContractDate = d }
	}
	if v := r.FormValue("delivery_date"); v != "" {
		if d, err := time.Parse(time.DateOnly, v); err == nil { existing.DeliveryDate = d }
	}
	if v := r.FormValue("contract_end_date"); v != "" {
		if d, err := time.Parse(time.DateOnly, v); err == nil { existing.ContractEndDate = &d }
	}
	if v := r.FormValue("delivery_conditions"); v != "" { existing.DeliveryConditions = v }
	if v := r.FormValue("delivery_term_days"); v != "" {
		if i, err := strconv.Atoi(v); err == nil { existing.DeliveryTermDays = i }
	}
	if v := r.FormValue("return_term_days"); v != "" {
		if i, err := strconv.Atoi(v); err == nil { existing.ReturnTermDays = i }
	}
	if v := r.FormValue("total_amount"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			existing.TotalAmount = f
		}
	}
	if v := r.FormValue("contract_currency"); v != "" { existing.ContractCurrency = strings.ToUpper(v) }
	if v := r.FormValue("sender_account"); v != "" { existing.SenderAccount = v }
	if v := r.FormValue("receiver_name"); v != "" { existing.ReceiverName = v }
	if v := r.FormValue("receiver_account"); v != "" { existing.ReceiverAccount = v }
	if v := r.FormValue("receiver_country"); v != "" { existing.ReceiverCountry = v }
	if v := r.FormValue("subject"); v != "" { existing.Subject = v }

	file, handler, err := r.FormFile("document")
	if err == nil {
		defer file.Close()
		os.MkdirAll("uploads/contracts", os.ModePerm)
		uniqueFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
		filePath := filepath.Join("uploads/contracts", uniqueFileName)
		dst, err := os.Create(filePath)
		if err == nil {
			io.Copy(dst, file)
			dst.Close()
			pathStr := filePath
			nameStr := handler.Filename
			existing.DocumentPath = &pathStr
			existing.OriginalDocumentName = &nameStr
		}
	}

	updated, err := h.service.Update(r.Context(), existing.ID, existing)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}