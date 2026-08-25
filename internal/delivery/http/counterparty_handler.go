package http

import (
	"encoding/json"
	"net/http"

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

func (h *CounterpartyHandler) Create(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}

	var req dto.CreateCompanyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	if req.BranchID == 0 || req.Name == "" || req.INN == "" {
		handleError(w, errs.ErrInvalidFieldValue)
		return
	}

	c := domain.Counterparty{
		Name:         req.Name,
		INN:          &req.INN,
		BranchID:     req.BranchID,
		IsThirdParty: false, 
	}

	created, err := h.service.Create(r.Context(), login, c)
	if err != nil {
		handleError(w, err)
		return
	}

	res := dto.CompanyResponse{
		ID:       created.ID,
		Name:     created.Name,
		INN:      *created.INN,
		BranchID: created.BranchID,
	}

	writeJSON(w, http.StatusCreated, res)
}
