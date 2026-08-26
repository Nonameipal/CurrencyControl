package http

import (
	"fmt"
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
// @Param branch_id query integer false "ID филиала"
// @Success 200 {array} dto.DashboardSearchResult
// @Failure 400 {object} dto.ErrorResponse "Неверные параметры"
// @Failure 401 {object} dto.ErrorResponse "Не авторизован"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/dashboard [get]
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
	
	if branchStr := r.URL.Query().Get("branch_id"); branchStr != "" {
		if branchID, err := strconv.Atoi(branchStr); err == nil {
			req.BranchID = branchID
		}
	}

	results, err := h.service.SearchDashboard(r.Context(), login, req)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, results)
}
// @Summary Создание контракта
// @Description Создает новый контракт с загрузкой PDF-файла
// @Tags Contracts
// @Accept multipart/form-data
// @Produce json
// @Param Login header string true "Логин пользователя"
// @Param client_id formData integer true "ID компании (ҶДММ)"
// @Param branch_id formData integer true "ID филиала"
// @Param contract_number formData string true "Номер контракта"
// @Param contract_name formData string true "Название контракта"
// @Param contract_date formData string true "Дата контракта (YYYY-MM-DD)"
// @Param delivery_date formData string true "Дата поставки (YYYY-MM-DD)"
// @Param contract_end_date formData string false "Дата окончания контракта (YYYY-MM-DD) (Необязательно)"
// @Param delivery_term_days formData integer true "Срок поставки (дни)"
// @Param return_term_days formData integer true "Срок возврата (дни)"
// @Param total_amount formData number true "Сумма контракта"
// @Param contract_currency formData string true "Валюта контракта"
// @Param receiver_name formData string true "Наименование получателя"
// @Param receiver_account formData string true "Счет получателя"
// @Param receiver_country formData string true "Страна получателя"
// @Param subject formData string true "Предмет"
// @Param delivery_conditions formData string false "Условия поставки"
// @Param document formData file true "PDF документ (Обязательно)"
// @Success 201 {object} domain.Contract
// @Failure 400 {object} dto.ErrorResponse "Неверные данные формы"
// @Failure 401 {object} dto.ErrorResponse "Не авторизован"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/contracts [post]
func (h *ContractHandler) Create(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}

	err := r.ParseMultipartForm(10 << 20)
	if err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	clientID, _ := strconv.ParseInt(r.FormValue("client_id"), 10, 64)
	branchID, _ := strconv.Atoi(r.FormValue("branch_id"))
	deliveryTermDays, _ := strconv.Atoi(r.FormValue("delivery_term_days"))
	returnTermDays, _ := strconv.Atoi(r.FormValue("return_term_days"))
	totalAmount, _ := strconv.ParseFloat(r.FormValue("total_amount"), 64)

	contractDate, _ := time.Parse(time.DateOnly, r.FormValue("contract_date"))
	deliveryDate, _ := time.Parse(time.DateOnly, r.FormValue("delivery_date"))
	
	var endDate *time.Time
	if val := r.FormValue("contract_end_date"); val != "" {
		if parsed, err := time.Parse(time.DateOnly, val); err == nil {
			endDate = &parsed
		}
	}

	contractCurrency := strings.ToUpper(strings.TrimSpace(r.FormValue("contract_currency")))
	if len(contractCurrency) != 3 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Код валюты должен состоять ровно из 3 букв (например: USD, RUB, CNY)"})
		return
	}

	receiverCountry := strings.ToUpper(strings.TrimSpace(r.FormValue("receiver_country")))
	if len(receiverCountry) != 2 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Код страны должен состоять ровно из 2 букв (например: RU, CN, TJ)"})
		return
	}

	contract := domain.Contract{
		ClientID:           &clientID,
		BranchID:           &branchID,
		ContractNumber:     r.FormValue("contract_number"),
		ContractName:       r.FormValue("contract_name"),
		ContractDate:       contractDate,
		DeliveryDate:       deliveryDate,
		ContractEndDate:    endDate,
		DeliveryConditions: r.FormValue("delivery_conditions"),
		DeliveryTermDays:   deliveryTermDays,
		ReturnTermDays:     returnTermDays,
		TotalAmount:        totalAmount,
		ContractCurrency:   contractCurrency,
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
	
	// Чтобы файлы с одинаковым названием не перезаписывали друг друга
	uniqueFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
	filePath := filepath.Join("uploads/contracts", uniqueFileName)
	
	dst, err := os.Create(filePath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: "Ошибка при сохранении файла на сервер"})
		return
	}
	defer dst.Close()
	io.Copy(dst, file)
	
	doc := &domain.Document{
		EntityType:   "contract",
		OriginalName: handler.Filename,
		FilePath:     filePath,
		FileSize:     handler.Size,
		MimeType:     handler.Header.Get("Content-Type"),
	}

	// Pass both to service.
	created, err := h.service.CreateWithDocument(r.Context(), login, contract, doc)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

