package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	_ "CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/repository"
	"CurrencyControl/internal/service"
)

type ReportHandler struct {
	svc service.ReportService
}

func NewReportHandler(svc service.ReportService) *ReportHandler {
	return &ReportHandler{svc: svc}
}

// @Summary Формирование отчетности по контрактам и подразделениям
// @Description Формирует аналитический отчет по контрактам, суммам, валютам, инвойсам и просрочкам. Для начальника подразделения доступ ограничен его филиалом. Для валютного контроля, комплаенса, аудита и админа доступен отчет по любому филиалу или сводный отчет.
// @Tags Reports
// @Security ApiKeyAuth
// @Produce json
// @Param branch_id query int false "ID подразделения (для начальника подразделения игнорируется и берется свой филиал)"
// @Param from_date query string false "Дата заключения контракта С (YYYY-MM-DD)"
// @Param to_date query string false "Дата заключения контракта ПО (YYYY-MM-DD)"
// @Param currency query string false "Валюта контракта (USD, EUR, TJS, RUB и др.)"
// @Success 200 {object} dto.ContractsReportResponse
// @Failure 400 {object} CommonError "Некорректные параметры"
// @Failure 401 {object} CommonError "Не авторизован"
// @Failure 403 {object} CommonError "Доступ запрещен"
// @Failure 500 {object} CommonError "Внутренняя ошибка сервера"
// @Router /api/reports/contracts [get]
func (h *ReportHandler) GetContractsReport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	role := GetRoleFromContext(r.Context())
	branchID := GetBranchIDFromContext(r.Context())

	filter := repository.ReportFilter{
		Currency: q.Get("currency"),
	}

	if bIDStr := q.Get("branch_id"); bIDStr != "" {
		if bID, err := strconv.Atoi(bIDStr); err == nil && bID > 0 {
			filter.BranchID = &bID
		}
	}

	if fromStr := strings.TrimSpace(q.Get("from_date")); fromStr != "" {
		if t, err := time.Parse("2006-01-02", fromStr); err == nil {
			filter.FromDate = &t
		}
	}

	if toStr := strings.TrimSpace(q.Get("to_date")); toStr != "" {
		if t, err := time.Parse("2006-01-02", toStr); err == nil {
			filter.ToDate = &t
		}
	}

	report, err := h.svc.GetContractsReport(r.Context(), role, branchID, filter)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}

	LogUserAction(r, "GENERATE_REPORT", "contracts_report", nil, "Сформирован отчет по контрактам")

	writeJSON(w, http.StatusOK, report)
}
