package http

import (
	"net/http"
	"strconv"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/errs"
	"CurrencyControl/internal/service/ports"
)

type ContractHandler struct {
	service ports.ContractService
}

func NewContractHandler(service ports.ContractService) *ContractHandler {
	return &ContractHandler{service: service}
}

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
