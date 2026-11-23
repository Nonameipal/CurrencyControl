package workflow

import (
	"net/http"
	"strconv"
	"strings"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/errs"
	"CurrencyControl/internal/service/ports"

	"github.com/gorilla/mux"
)

type ApprovalHandler struct {
	svc ports.ApprovalService
}
func NewApprovalHandler(svc ports.ApprovalService) *ApprovalHandler {
	return &ApprovalHandler{svc: svc}
}


// @Summary Проверка документа отделом валютного контроля
// @Description Сотрудник валютного контроля изучает предмет документа и выносит одно из трех решений: "accepted" (Принято), "revision" (На доработку), "rejected" (Отклонено). При принятии ("accepted") комментарий не обязателен. При "revision" и "rejected" комментарий обязателен.
// @Tags Approvals
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param entity_type path string true "Тип документа (contract, invoice, gtd, additional_agreement, payment_order)"
// @Param id path int true "ID документа"
// @Param body body dto.CurrencyControlDecisionRequest true "Решение валютного контроля"
// @Success 200 {object} dto.ApprovalItemResponse
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 404 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/approvals/{entity_type}/{id}/currency-control [post]
func (h *ApprovalHandler) ReviewCurrencyControl(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	role := GetRoleFromContext(r.Context())
	switch role {
	case domain.RoleCurrencyControl, domain.RoleCurrencyController, domain.RoleAdmin:
	default:
		writeJSON(w, http.StatusForbidden, CommonError{Error: "только сотрудники отдела валютного контроля могут выносить решение на данном этапе"})
		return
	}
	id, ok := requireID(w, r, "id")
	if !ok {
		return
	}
	entityType := mux.Vars(r)["entity_type"]
	var req dto.CurrencyControlDecisionRequest
	if err := decodeJSON(r, &req); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}
	res, err := h.svc.ReviewCurrencyControl(r.Context(), role, login, entityType, id, req)
	if err != nil {
		handleError(w, err)
		return
	}
	LogUserAction(r, "CURRENCY_CONTROL_DECISION", entityType, &id, "Решение валютного контроля: " + req.Decision)
	writeJSON(w, http.StatusOK, res)
}

// @Summary Проверка документа отделом комплаенс-контроля
// @Description Сотрудник комплаенса может одобрить документ ("approve"), отправить на доработку ("revision") или отклонить ("reject"). При одобрении ("approve") комментарий не обязателен. При отправке на доработку ("revision") или отклонении ("reject") указание причины в комментарии обязательно.
// @Tags Approvals
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param entity_type path string true "Тип документа (contract, invoice, gtd, additional_agreement, payment_order)"
// @Param id path int true "ID документа"
// @Param body body dto.ComplianceDecisionRequest true "Решение комплаенс-контроля"
// @Success 200 {object} dto.ApprovalItemResponse
// @Failure 400 {object} CommonError "Причина обязательна при отправке на доработку или отклонении"
// @Failure 401 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 404 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/approvals/{entity_type}/{id}/compliance [post]
func (h *ApprovalHandler) ReviewCompliance(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	role := GetRoleFromContext(r.Context())
	switch role {
	case domain.RoleCompliance, domain.RoleAdmin:
	default:
		writeJSON(w, http.StatusForbidden, CommonError{Error: "только сотрудники отдела комплаенса могут выносить решение на данном этапе"})
		return
	}
	id, ok := requireID(w, r, "id")
	if !ok {
		return
	}
	entityType := mux.Vars(r)["entity_type"]
	var req dto.ComplianceDecisionRequest
	if err := decodeJSON(r, &req); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}
	res, err := h.svc.ReviewCompliance(r.Context(), role, login, entityType, id, req)
	if err != nil {
		handleError(w, err)
		return
	}
	LogUserAction(r, "COMPLIANCE_DECISION", entityType, &id, "Решение комплаенса: " + req.Decision)
	writeJSON(w, http.StatusOK, res)
}

// @Summary Список документов на согласовании
// @Description Возвращает документы, ожидающие проверки валютным контролем или комплаенсом, либо возвращенные на доработку.
// @Tags Approvals
// @Security ApiKeyAuth
// @Produce json
// @Param stage query string false "Этап согласования: currency_control, compliance, revision"
// @Param entity_type query string false "Тип документа: contract, invoice, gtd, additional_agreement, payment_order"
// @Param branch_id query int false "ID филиала (для фильтрации)"
// @Param page query int false "Номер страницы (по умолчанию 1)"
// @Param page_size query int false "Размер страницы (по умолчанию 20, макс 100)"
// @Success 200 {object} dto.PendingApprovalsResponse
// @Failure 401 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/approvals/pending [get]
func (h *ApprovalHandler) GetPendingApprovals(w http.ResponseWriter, r *http.Request) {
	role := GetRoleFromContext(r.Context())
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	var branchIDPtr *int
	if bIDStr := q.Get("branch_id"); bIDStr != "" {
		if bID, err := strconv.Atoi(bIDStr); err == nil && bID > 0 {
			branchIDPtr = &bID
		}
	}
	filter := dto.PendingApprovalsFilter{
		Stage:      strings.TrimSpace(q.Get("stage")),
		EntityType: strings.TrimSpace(q.Get("entity_type")),
		BranchID:   branchIDPtr,
		Page:       page,
		PageSize:   pageSize,
	}
	userBranchID := GetBranchIDFromContext(r.Context())
	res, err := h.svc.GetPendingApprovals(r.Context(), role, userBranchID, filter)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// @Summary Список принятых документов (одобрены и валютным контролем, и комплаенсом)
// @Description Возвращает документы (ГТД и платежные поручения), успешно прошедшие оба этапа проверки. Доступно только администратору.
// @Tags Approvals
// @Security ApiKeyAuth
// @Produce json
// @Param entity_type query string false "Тип документа: gtd, payment_order"
// @Param branch_id query int false "ID филиала (для фильтрации)"
// @Param page query int false "Номер страницы (по умолчанию 1)"
// @Param page_size query int false "Размер страницы (по умолчанию 20, макс 100)"
// @Success 200 {object} dto.AcceptedApprovalsResponse
// @Failure 401 {object} CommonError
// @Failure 403 {object} CommonError "Доступ разрешён только администраторам"
// @Failure 500 {object} CommonError
// @Router /api/approvals/accepted [get]
func (h *ApprovalHandler) GetAcceptedApprovals(w http.ResponseWriter, r *http.Request) {
	role := GetRoleFromContext(r.Context())
	if role != domain.RoleAdmin {
		writeJSON(w, http.StatusForbidden, CommonError{Error: "доступ разрешён только администраторам"})
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
	filter := dto.AcceptedApprovalsFilter{
		EntityType: strings.TrimSpace(q.Get("entity_type")),
		BranchID:   branchIDPtr,
		Page:       page,
		PageSize:   pageSize,
	}
	res, err := h.svc.GetAcceptedApprovals(r.Context(), role, filter)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}

// @Summary Детальная информация о статусе авторизации документа
// @Description Возвращает текущий статус двухэтапной проверки документа, историю решений и комментарии.
// @Tags Approvals
// @Security ApiKeyAuth
// @Produce json
// @Param entity_type path string true "Тип документа (contract, invoice, gtd, additional_agreement, payment_order)"
// @Param id path int true "ID документа"
// @Success 200 {object} dto.ApprovalItemResponse
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 404 {object} CommonError
// @Failure 500 {object} CommonError
// @Router /api/approvals/{entity_type}/{id} [get]
func (h *ApprovalHandler) GetApprovalDetail(w http.ResponseWriter, r *http.Request) {
	id, ok := requireID(w, r, "id")
	if !ok {
		return
	}
	entityType := mux.Vars(r)["entity_type"]
	res, err := h.svc.GetApprovalDetail(r.Context(), entityType, id)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
