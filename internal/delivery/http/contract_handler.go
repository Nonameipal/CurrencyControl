package http

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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
// @Router /api/v1/dashboard [get]
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
// @Param delivery_term_days formData integer true "Срок поставки (дни)"
// @Param return_term_days formData integer true "Срок возврата (дни)"
// @Param total_amount formData number true "Сумма контракта"
// @Param contract_currency formData string true "Валюта контракта"
// @Param receiver_name formData string true "Наименование получателя"
// @Param receiver_account formData string true "Счет получателя"
// @Param receiver_country formData string true "Страна получателя"
// @Param subject formData string true "Предмет"
// @Param delivery_conditions formData string false "Условия поставки"
// @Param document formData file false "PDF документ"
// @Success 201 {object} domain.Contract
// @Failure 400 {object} dto.ErrorResponse "Неверные данные формы"
// @Failure 401 {object} dto.ErrorResponse "Не авторизован"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
// @Router /api/v1/contracts [post]
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
	
	contract := domain.Contract{
		ClientID:           &clientID,
		BranchID:           &branchID,
		ContractNumber:     r.FormValue("contract_number"),
		ContractName:       r.FormValue("contract_name"),
		ContractDate:       contractDate,
		DeliveryDate:       deliveryDate,
		DeliveryConditions: r.FormValue("delivery_conditions"),
		DeliveryTermDays:   deliveryTermDays,
		ReturnTermDays:     returnTermDays,
		TotalAmount:        totalAmount,
		ContractCurrency:   r.FormValue("contract_currency"),
		ReceiverName:       r.FormValue("receiver_name"),
		ReceiverAccount:    r.FormValue("receiver_account"),
		ReceiverCountry:    r.FormValue("receiver_country"),
		Subject:            r.FormValue("subject"),
	}

	file, handler, err := r.FormFile("document")
	var doc *domain.Document
	if err == nil {
		defer file.Close()
		os.MkdirAll("uploads/contracts", os.ModePerm)
		
		filePath := filepath.Join("uploads/contracts", handler.Filename)
		dst, err := os.Create(filePath)
		if err == nil {
			defer dst.Close()
			io.Copy(dst, file)
			
			d := domain.Document{
				EntityType:   "contract",
				OriginalName: handler.Filename,
				FilePath:     filePath,
				FileSize:     handler.Size,
				MimeType:     handler.Header.Get("Content-Type"),
			}
			doc = &d
		}
	}

	created, err := h.service.CreateWithDocument(r.Context(), login, contract, doc)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

