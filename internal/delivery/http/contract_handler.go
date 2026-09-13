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
// @Security ApiKeyAuth
// @Produce json
// @Param company_name query string false "Название компании"
// @Param amount query number false "Сумма контракта"
// @Param inn query string false "ИНН компании"
// @Param id path int true "ID филиала"
// @Success 200 {array} dto.DashboardSearchResult
// @Failure 400 {object} map[string]string "Неверные параметры"
// @Failure 401 {object} map[string]string "Не авторизован"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
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
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
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
// @Failure 400 {object} map[string]string "Обязательные поля не заполнены"
// @Failure 401 {object} map[string]string "Не авторизован"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
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
	if returnTermDays <= 0 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле return_term_days обязательно и должно быть больше 0"})
		return
	}
	if totalAmount <= 0 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле total_amount обязательно и должно быть больше 0"})
		return
	}
	if strings.TrimSpace(r.FormValue("subject")) == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле subject обязательно"})
		return
	}

	var returnDate *time.Time
	if v := r.FormValue("return_date"); strings.TrimSpace(v) != "" {
		if d, err := time.Parse(time.DateOnly, strings.TrimSpace(v)); err == nil {
			returnDate = &d
		}
	}

	contract := domain.Contract{
		ClientID:       &clientID,
		BranchID:       &branchID,
		ContractNumber: r.FormValue("contract_number"),
		ContractDate:   contractDate,
		DeliveryDate:   deliveryDate,
		ContractEndDate: &parsedEndDate,
		ReturnTermDays: returnTermDays,
		ReturnDate:     returnDate,
		TotalAmount:    totalAmount,
		ContractCurrency: contractCurrency,
		Subject:        r.FormValue("subject"),
		CreatedBy:      login,
	}

	file, handler, err := r.FormFile("document")
	if err == nil {
		defer file.Close()
		ext := strings.ToLower(filepath.Ext(handler.Filename))
		if ext != ".pdf" && ext != ".doc" && ext != ".docx" {
			writeJSON(w, http.StatusBadRequest, CommonError{Error: "Разрешены только файлы форматов PDF и Word (.pdf, .doc, .docx)"})
			return
		}

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
		contract.DocumentPath = &pathStr
	}

	created, err := h.service.Create(r.Context(), login, contract)
	if err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "CREATE", "contract", &created.ID, "Создание контракта № "+created.ContractNumber)

	writeJSON(w, http.StatusCreated, created)
}

// @Summary Получить список контрактов ЧДММ
// @Description Возвращает список контрактов для выбранной компании. 
// @Tags Contracts
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании (ЧДММ)"
// @Success 200 {array} domain.Contract
// @Failure 400 {object} map[string]string "Неверные параметры"
// @Failure 401 {object} map[string]string "Не авторизован"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
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
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Success 200 {array} dto.NotificationResponse
// @Failure 401 {object} map[string]string "Не авторизован"
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
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
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
// @Failure 400 {object} map[string]string "Некорректный запрос"
// @Failure 403 {object} map[string]string "Доступ запрещен"
// @Failure 500 {object} map[string]string "Внутренняя ошибка сервера"
// @Router /admin/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id} [put]
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
	if v := r.FormValue("contract_date"); v != "" {
		if d, err := time.Parse(time.DateOnly, v); err == nil { existing.ContractDate = d }
	}
	if v := r.FormValue("delivery_date"); v != "" {
		if d, err := time.Parse(time.DateOnly, v); err == nil { existing.DeliveryDate = d }
	}
	if v := r.FormValue("contract_end_date"); v != "" {
		if d, err := time.Parse(time.DateOnly, v); err == nil { existing.ContractEndDate = &d }
	}
	if v := r.FormValue("return_term_days"); v != "" {
		if i, err := strconv.Atoi(v); err == nil { existing.ReturnTermDays = i }
	}
	if v := r.FormValue("return_date"); v != "" {
		if d, err := time.Parse(time.DateOnly, v); err == nil { existing.ReturnDate = &d }
	}
	if v := r.FormValue("total_amount"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			existing.TotalAmount = f
		}
	}
	if v := r.FormValue("contract_currency"); v != "" { existing.ContractCurrency = strings.ToUpper(v) }
	if v := r.FormValue("subject"); v != "" { existing.Subject = v }

	file, handler, err := r.FormFile("document")
	if err == nil {
		defer file.Close()
		ext := strings.ToLower(filepath.Ext(handler.Filename))
		if ext != ".pdf" && ext != ".doc" && ext != ".docx" {
			writeJSON(w, http.StatusBadRequest, CommonError{Error: "Разрешены только файлы форматов PDF и Word (.pdf, .doc, .docx)"})
			return
		}
		os.MkdirAll("uploads/contracts", os.ModePerm)
		uniqueFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
		filePath := filepath.Join("uploads/contracts", uniqueFileName)
		dst, err := os.Create(filePath)
		if err == nil {
			io.Copy(dst, file)
			dst.Close()
			pathStr := filePath
			existing.DocumentPath = &pathStr
		}
	}

	updated, err := h.service.Update(r.Context(), existing.ID, existing)
	if err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "UPDATE", "contract", &updated.ID, "Обновление контракта № "+updated.ContractNumber)

	writeJSON(w, http.StatusOK, updated)
}

// @Summary Удаление контракта
// @Description Позволяет администратору удалить контракт (soft delete)
// @Tags Admin
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
// @Router /admin/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id} [delete]
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
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}
	contractID, err := strconv.ParseInt(mux.Vars(r)["contract_id"], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID контракта"})
		return
	}

	contract, err := h.service.GetByID(r.Context(), login, contractID)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, contract)
}

// @Summary Список архивных контрактов по филиалу (с пагинацией)
// @Description Возвращает постраничный список архивных контрактов для указанного филиала
// @Tags Archive
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param page query int false "Номер страницы (по умолчанию 1)"
// @Param page_size query int false "Размер страницы (по умолчанию 20, макс 100)"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/branches/{id}/archive [get]
func (h *ContractHandler) GetArchivedContracts(w http.ResponseWriter, r *http.Request) {
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

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	contracts, total, err := h.service.GetArchived(r.Context(), branchID, page, pageSize)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"data":      contracts,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// @Summary Список архивных контрактов по компании
// @Description Возвращает все архивные контракты для выбранной компании
// @Tags Archive
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Success 200 {array} domain.Contract
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/branches/{id}/dashboard/companies/{company_id}/archive [get]
func (h *ContractHandler) GetArchivedByCompany(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}

	clientID, err := strconv.ParseInt(mux.Vars(r)["company_id"], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID компании"})
		return
	}

	contracts, err := h.service.GetArchivedByClientID(r.Context(), clientID)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, contracts)
}

// @Summary Разархивировать контракт (восстановить в active)
// @Description Переводит контракт из статуса archived обратно в active
// @Tags Archive
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /admin/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/restore [put]
func (h *ContractHandler) RestoreContract(w http.ResponseWriter, r *http.Request) {
	contractID, err := strconv.ParseInt(mux.Vars(r)["contract_id"], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID контракта"})
		return
	}

	if err := h.service.RestoreContract(r.Context(), contractID); err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "RESTORE", "contract", &contractID, "Восстановление контракта из архива")
	writeJSON(w, http.StatusOK, map[string]string{"message": "Контракт успешно восстановлен из архива"})
}
