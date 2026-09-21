package http

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/errs"
	"CurrencyControl/internal/service/ports"

	"github.com/gorilla/mux"
)

type GTDExtensionHandler struct {
	svc ports.GTDExtensionService
}

func NewGTDExtensionHandler(svc ports.GTDExtensionService) *GTDExtensionHandler {
	return &GTDExtensionHandler{svc: svc}
}

type reviewExtensionBody struct {
	Decision         string  `json:"decision" enums:"approve,reject" example:"approve"`
	ApprovedDeadline *string `json:"approved_deadline,omitempty" example:"2026-10-15"`
	Comment          string  `json:"comment,omitempty"`
}

// @Summary Подать заявку на увеличение срока ГТД
// @Description Операционист выбирает в календаре новую дату дедлайна и прикрепляет подтверждающий документ. Заявка отправляется на рассмотрение в Валютный контроль.
// @Tags GTD Extensions
// @Security ApiKeyAuth
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param invoice_id path int true "ID инвойса"
// @Param gtd_id path int true "ID ГТД"
// @Param requested_deadline formData string true "Календарь: новая дата срока ГТД (YYYY-MM-DD или DD.MM.YYYY)"
// @Param document formData file true "Файл документа-обоснования (.pdf, .doc, .docx)"
// @Success 201 {object} domain.GTDExtensionRequest
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/gtd/{gtd_id}/extend [post]
func (h *GTDExtensionHandler) RequestExtension(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	role := GetRoleFromContext(r.Context())
	if login == "" || role == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}

	gtdID, err := strconv.ParseInt(mux.Vars(r)["gtd_id"], 10, 64)
	if err != nil || gtdID <= 0 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID ГТД"})
		return
	}

	if err := r.ParseMultipartForm(32 << 20); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}
	deadlineStr := strings.TrimSpace(r.FormValue("requested_deadline"))
	if deadlineStr == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле requested_deadline (дата из календаря) обязательно для заполнения"})
		return
	}
	reqDeadline := parseDate(deadlineStr)
	if reqDeadline == nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Неверный формат даты в календаре. Ожидается YYYY-MM-DD или DD.MM.YYYY"})
		return
	}

	filePath, err := saveUploadedFile(r, "document", "uploads/gtd_extensions", true)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}

	created, err := h.svc.CreateRequest(r.Context(), login, role, gtdID, *reqDeadline, filePath)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}

	LogUserAction(r, "REQUEST_GTD_EXTENSION", "gtd", &gtdID, fmt.Sprintf("Подана заявка на увеличение срока ГТД №%d до %s", gtdID, reqDeadline.Format("02.01.2006")))

	writeJSON(w, http.StatusCreated, created)
}

// @Summary История продлений срока ГТД
// @Description Возвращает все заявки на продление срока по конкретной ГТД
// @Tags GTD Extensions
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param invoice_id path int true "ID инвойса"
// @Param gtd_id path int true "ID ГТД"
// @Success 200 {array} domain.GTDExtensionRequest
// @Failure 401 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/invoices/{invoice_id}/gtd/{gtd_id}/extension-history [get]
func (h *GTDExtensionHandler) GetExtensionHistory(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}

	gtdID, err := strconv.ParseInt(mux.Vars(r)["gtd_id"], 10, 64)
	if err != nil || gtdID <= 0 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID ГТД"})
		return
	}

	list, err := h.svc.GetByGTDID(r.Context(), gtdID)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, list)
}

// @Summary Заявки на увеличение срока ГТД (для Валютного контроля)
// @Description Возвращает список ожидающих рассмотрения заявок на продление сроков ГТД
// @Tags Approvals
// @Security ApiKeyAuth
// @Produce json
// @Param branch_id query int false "ID филиала"
// @Param page query int false "Номер страницы (по умолчанию 1)"
// @Param page_size query int false "Размер страницы (по умолчанию 20)"
// @Success 200 {object} map[string]interface{}
// @Failure 401 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/approvals/gtd-extensions/pending [get]
func (h *GTDExtensionHandler) GetPendingExtensions(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	role := GetRoleFromContext(r.Context())
	if login == "" || role == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}

	switch role {
	case domain.RoleCurrencyControl, domain.RoleCurrencyController, domain.RoleCompliance, domain.RoleAdmin:
	default:
		writeJSON(w, http.StatusForbidden, CommonError{Error: "Доступ разрешен только сотрудникам валютного контроля"})
		return
	}

	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	var branchIDPtr *int
	if bIDStr := q.Get("branch_id"); bIDStr != "" {
		if bID, err := strconv.Atoi(bIDStr); err == nil && bID > 0 {
			branchIDPtr = &bID
		}
	}

	items, total, err := h.svc.GetPendingRequests(r.Context(), branchIDPtr, page, pageSize)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"items":     items,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

// @Summary Рассмотрение заявки на увеличение срока ГТД
// @Description Сотрудник Валютного контроля утверждает (approve) или отклоняет (reject) заявку на продление срока ГТД. При утверждении срок ГТД автоматически продлевается и просрочка пересчитывается.
// @Tags Approvals
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param request_id path int true "ID заявки на продление"
// @Param body body reviewExtensionBody true "Решение валютного контроля"
// @Success 200 {object} domain.GTDExtensionRequest
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/approvals/gtd-extensions/{request_id}/review [post]
func (h *GTDExtensionHandler) ReviewExtension(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	role := GetRoleFromContext(r.Context())
	if login == "" || role == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}

	reqID, err := strconv.ParseInt(mux.Vars(r)["request_id"], 10, 64)
	if err != nil || reqID <= 0 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID заявки"})
		return
	}

	var body reviewExtensionBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	var approvedDate *time.Time
	if body.ApprovedDeadline != nil && *body.ApprovedDeadline != "" {
		approvedDate = parseDate(*body.ApprovedDeadline)
	}

	reviewed, err := h.svc.ReviewRequest(r.Context(), role, login, reqID, body.Decision, approvedDate, body.Comment)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}

	actionDesc := fmt.Sprintf("Валютный контроль вынес решение '%s' по заявке на продление срока ГТД №%d", reviewed.Status, reviewed.GTDID)
	LogUserAction(r, "REVIEW_GTD_EXTENSION", "gtd_extension_request", &reviewed.ID, actionDesc)

	writeJSON(w, http.StatusOK, reviewed)
}
