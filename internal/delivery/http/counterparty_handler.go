package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"github.com/gorilla/mux"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/errs"
	"CurrencyControl/internal/service/ports"
)

type CounterpartyHandler struct {
	service ports.CounterpartyService
}

func NewCounterpartyHandler(service ports.CounterpartyService) *CounterpartyHandler {
	return &CounterpartyHandler{service: service}
}

// @Summary Создание новой компании (ҶДММ)
// @Description Создает новую компанию (контрагента) с обязательной привязкой к филиалу
// @Tags Companies
// @Accept json
// @Produce json
// @Param Login header string true "Логин пользователя"
// @Param request body dto.CreateCompanyRequest true "Данные для создания компании"
// @Param id path int true "ID филиала"
// @Success 201 {object} dto.CompanyResponse
// @Failure 400 {object} dto.ErrorResponse "Обязательные поля не заполнены"
// @Failure 401 {object} dto.ErrorResponse "Не авторизован"
// @Failure 500 {object} dto.ErrorResponse "Внутренняя ошибка сервера"
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

	if req.LLC == "" || req.INN == "" {
		handleError(w, errs.ErrInvalidFieldValue)
		return
	}

	companyName := strings.TrimSpace(req.LLC)
	if !strings.HasPrefix(strings.ToUpper(companyName), "ҶДММ") && !strings.HasPrefix(companyName, "ЧДММ") {
		companyName = "ҶДММ " + companyName
	}

	c := domain.Counterparty{
		Name:         companyName,
		INN:          &req.INN,
		BranchID:     branchID,
		IsThirdParty: false, 
	}
	
	exists, err := h.service.CheckExistsInBranch(r.Context(), c.BranchID, c.Name)
	if err != nil {
		handleError(w, err)
		return
	}
	if exists {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Компания с таким названием уже существует в выбранном филиале"})
		return
	}

	created, err := h.service.Create(r.Context(), login, c)
	if err != nil {
		handleError(w, err)
		return
	}

	res := dto.CompanyResponse{
		ID:       created.ID,
		LLC:      created.Name,
		INN:      *created.INN,
		BranchID: created.BranchID,
	}

	writeJSON(w, http.StatusCreated, res)
}
