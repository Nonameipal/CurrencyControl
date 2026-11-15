package workflow

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/service/ports"
)

type ReportHandler struct {
	svc ports.ReportService
}

func NewReportHandler(svc ports.ReportService) *ReportHandler {
	return &ReportHandler{svc: svc}
}

// @Summary Формирование отчетности по контрактам и подразделениям (JSON)
// @Description Формирует аналитический отчет по контрактам, суммам, валютам, инвойсам и просрочкам.
// @Tags Reports
// @Security ApiKeyAuth
// @Produce json
// @Param branch_id query int false "ID подразделения"
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
	filter := ports.ReportFilter{
		Currency: q.Get("currency"),
	}
	if bIDStr := q.Get("branch_id"); bIDStr != "" {
		if bID, err := strconv.Atoi(bIDStr); err == nil && bID > 0 {
			filter.BranchID = &bID
		}
	}
	if fromStr := strings.TrimSpace(q.Get("from_date")); fromStr != "" {
		if t := parseDate(fromStr); t != nil {
			filter.FromDate = t
		}
	}
	if toStr := strings.TrimSpace(q.Get("to_date")); toStr != "" {
		if t := parseDate(toStr); t != nil {
			filter.ToDate = t
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

// @Summary Список доступных видов аналитических отчетов
// @Description Возвращает список доступных видов отчетов (contracts, invoices, gtd, clients, client_consolidated)
// @Tags Reports
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {array} dto.ReportTypeInfo
// @Router /api/reports/types [get]
func (h *ReportHandler) GetReportTypes(w http.ResponseWriter, r *http.Request) {
	types := h.svc.GetReportTypes()
	writeJSON(w, http.StatusOK, types)
}

// @Summary Выгрузка аналитического отчета в формате Excel (.xlsx)
// @Description Формирует и выгружает заполненный Excel-отчет строго по заданной форме выбранного вида.
// @Tags Reports
// @Security ApiKeyAuth
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param type query string true "Вид отчета: contracts, invoices, gtd, clients, client_consolidated"
// @Param inn query string false "ИНН клиента"
// @Param branch_id query int false "ID филиала"
// @Param from_date query string false "Дата начала периода (YYYY-MM-DD)"
// @Param to_date query string false "Дата окончания периода (YYYY-MM-DD)"
// @Param currency query string false "Фильтр по валюте"
// @Success 200 {file} binary "Файл Excel отчета"
// @Failure 400 {object} CommonError "Некорректные параметры"
// @Failure 401 {object} CommonError "Не авторизован"
// @Failure 403 {object} CommonError "Доступ запрещен"
// @Router /api/reports/export [get]
func (h *ReportHandler) ExportExcelReport(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	reportType := strings.TrimSpace(q.Get("type"))
	if reportType == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Параметр 'type' обязателен (contracts, invoices, gtd, clients, client_consolidated)"})
		return
	}
	role := GetRoleFromContext(r.Context())
	branchID := GetBranchIDFromContext(r.Context())
	filter := dto.ExcelReportFilter{
		Currency:  strings.ToUpper(strings.TrimSpace(q.Get("currency"))),
		ClientINN: strings.TrimSpace(q.Get("inn")),
	}
	if bIDStr := q.Get("branch_id"); bIDStr != "" {
		if bID, err := strconv.Atoi(bIDStr); err == nil && bID > 0 {
			filter.BranchID = &bID
		}
	}
	if fromStr := strings.TrimSpace(q.Get("from_date")); fromStr != "" {
		if t := parseDate(fromStr); t != nil {
			filter.FromDate = t
		}
	}
	if toStr := strings.TrimSpace(q.Get("to_date")); toStr != "" {
		if t := parseDate(toStr); t != nil {
			filter.ToDate = t
		}
	}
	fileBytes, fileName, err := h.svc.ExportExcelReport(r.Context(), role, branchID, reportType, filter)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}
	LogUserAction(r, "EXPORT_EXCEL_REPORT", reportType, nil, fmt.Sprintf("Выгружен отчет %s", fileName))
	w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, fileName))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(fileBytes)
}

// @Summary Доступные валюты клиента
// @Description Возвращает список валют (USD, EUR и т.д.), используемых в сделках клиента для фильтрации (чекбоксы/галочки).
// @Tags Reports
// @Security ApiKeyAuth
// @Produce json
// @Param client_id path int true "ID клиента"
// @Success 200 {array} string
// @Failure 400 {object} CommonError "Некорректный ID клиента"
// @Failure 401 {object} CommonError "Не авторизован"
// @Failure 500 {object} CommonError "Внутренняя ошибка сервера"
// @Router /api/reports/clients/{client_id}/currencies [get]
func (h *ReportHandler) GetClientCurrencies(w http.ResponseWriter, r *http.Request) {
	clientID, ok := requireID(w, r, "client_id")
	if !ok {
		return
	}
	currencies, err := h.svc.GetClientCurrencies(r.Context(), clientID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, currencies)
}
