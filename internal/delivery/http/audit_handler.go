package http

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"
)

type AuditHandler struct {
	svc ports.AuditLogService
}

func NewAuditHandler(svc ports.AuditLogService) *AuditHandler {
	return &AuditHandler{svc: svc}
}

type AuditLogsResponse struct {
	Total  int64             `json:"total"`
	Limit  int               `json:"limit"`
	Offset int               `json:"offset"`
	Logs   []domain.AuditLog `json:"logs"`
}

// @Summary Просмотр журнала аудита и действий пользователей
// @Description Возвращает историю действий пользователей с фильтрацией по логину, действию, сущности, подразделению и датам. Доступно ролям compliance, internal_audit и admin.
// @Tags Audit
// @Security ApiKeyAuth
// @Produce json
// @Param user_login query string false "Фильтр по логину пользователя"
// @Param action query string false "Фильтр по действию (например: LOGIN, CREATE, UPDATE, DELETE)"
// @Param entity query string false "Фильтр по сущности (например: contract, invoice, company, branch)"
// @Param branch_id query int false "Фильтр по ID подразделения"
// @Param from_date query string false "Дата начала (YYYY-MM-DD)"
// @Param to_date query string false "Дата окончания (YYYY-MM-DD)"
// @Param limit query int false "Лимит записей (по умолчанию 50)"
// @Param offset query int false "Смещение (по умолчанию 0)"
// @Success 200 {object} AuditLogsResponse
// @Failure 401 {object} CommonError "Не авторизован"
// @Failure 403 {object} CommonError "Доступ запрещен"
// @Failure 500 {object} CommonError "Внутренняя ошибка сервера"
// @Router /api/audit-logs [get]
func (h *AuditHandler) GetLogs(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	filter := ports.AuditLogFilter{
		UserLogin: q.Get("user_login"),
		Action:    q.Get("action"),
		Entity:    q.Get("entity"),
		Limit:     50,
		Offset:    0,
	}

	if bIDStr := q.Get("branch_id"); bIDStr != "" {
		if bID, err := strconv.ParseInt(bIDStr, 10, 64); err == nil && bID > 0 {
			filter.BranchID = &bID
		}
	}

	if limitStr := q.Get("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil && limit > 0 {
			filter.Limit = limit
		}
	}

	if offsetStr := q.Get("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil && offset >= 0 {
			filter.Offset = offset
		}
	}

	if fromStr := strings.TrimSpace(q.Get("from_date")); fromStr != "" {
		if t := parseDate(fromStr); t != nil {
			filter.FromDate = t
		}
	}

	if toStr := strings.TrimSpace(q.Get("to_date")); toStr != "" {
		if t := parseDate(toStr); t != nil {
			endOfDay := t.Add(23*time.Hour + 59*time.Minute + 59*time.Second)
			filter.ToDate = &endOfDay
		}
	}

	logs, total, err := h.svc.List(r.Context(), filter)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: err.Error()})
		return
	}

	res := AuditLogsResponse{
		Total:  total,
		Limit:  filter.Limit,
		Offset: filter.Offset,
		Logs:   logs,
	}

	writeJSON(w, http.StatusOK, res)
}
