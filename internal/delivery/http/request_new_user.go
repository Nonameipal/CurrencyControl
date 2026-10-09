package http
import (
	"fmt"
	"log"
	"net/http"
	"strings"

	"CurrencyControl/internal/domain"
	_ "CurrencyControl/internal/service/ports"
)
// @Summary Список доступных филиалов
// @Description Возвращает список всех существующих филиалов
// @Tags Auth
// @Produce json
// @Success 200 {array} domain.Branch
// @Router /auth/branches [get]
func (h *AuthHandler) GetBranches(w http.ResponseWriter, r *http.Request) {
	if h.branchSvc == nil {
		writeJSON(w, http.StatusOK, []domain.Branch{})
		return
	}
	list, err := h.branchSvc.GetAll(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: err.Error()})
		return
	}
	if list == nil {
		list = []domain.Branch{}
	}
	writeJSON(w, http.StatusOK, list)
}

// @Summary Список доступных ролей
// @Description Возвращает список всех существующих ролей
// @Tags Auth
// @Produce json
// @Success 200 {array} RoleInfo
// @Router /auth/roles [get]
func (h *AuthHandler) GetRoles(w http.ResponseWriter, r *http.Request) {
	roles := []RoleInfo{
		{
			Role: domain.RoleOperator,
			Name: "Операционист",
		},
		{
			Role: domain.RoleBranchHead,
			Name: "Руководитель филиала",
		},
		{
			Role: domain.RoleCurrencyControl,
			Name: "Валютный контроль",
		},
		{
			Role: domain.RoleCompliance,
			Name: "Комплаенс",
		},
		{
			Role: domain.RoleInternalAudit,
			Name: "Внутренний аудит",
		},
	}
	writeJSON(w, http.StatusOK, roles)
}

// @Summary Запрос на доступ (для новых пользователей)
// @Description Новый пользователь, прошедший AD-проверку, выбирает свой филиал и роль из существующих.
// @Tags Auth
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param body body requestAccessBody true "Выбор филиала и роли (выберите из существующих вариантов)"
// @Success 202 {object} AccessRequestResponse
// @Failure 400 {object} CommonError
// @Router /auth/request-access [post]
func (h *AuthHandler) RequestAccess(w http.ResponseWriter, r *http.Request) {
	var req requestAccessBody
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректное тело запроса"})
		return
	}
	login := ""
	lastName := ""
	firstName := ""
	email := ""

	token := extractToken(r)
	if token != "" {
		sess, err := h.svc.ValidateSession(r.Context(), token)
		if err == nil && sess != nil && sess.Login != "" {
			login = sess.Login
			lastName = sess.LastName
			firstName = sess.FirstName
			email = sess.Email
		}
	}
	if login == "" {
		login = GetLoginFromContext(r.Context())
		lastName = GetLastNameFromContext(r.Context())
		firstName = GetFirstNameFromContext(r.Context())
		email = GetEmailFromContext(r.Context())
	}
	if login == "" {
		writeJSON(w, http.StatusUnauthorized, CommonError{Error: "Сессия не найдена. Выполните вход через /auth/login"})
		return
	}
	if req.BranchID <= 0 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Выберите филиал из списка (branch_id обязателен)"})
		return
	}
	if !isValidRole(req.Role) {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Выберите роль из списка: operator, branch_head, currency_control, compliance, internal_audit"})
		return
	}
	accessReq, err := h.svc.RequestAccess(r.Context(), login, lastName, firstName, email, req.BranchID, req.Role)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}

	if globalAuditSvc != nil {
		ip := getClientIP(r)
		bID := req.BranchID
		globalAuditSvc.Log(r.Context(), login, req.Role, &bID, "REQUEST_ACCESS", "access_request", &accessReq.ID,
			fmt.Sprintf("Запрос на доступ от логина '%s' (филиал: %s [ID: %d], роль: %s)", login, accessReq.BranchName, accessReq.BranchID, accessReq.Role), ip)
	}
	log.Printf("Запрос на доступ создан: логин='%s', филиал=%d (%s), роль=%s", login, accessReq.BranchID, accessReq.BranchName, accessReq.Role)

	resp := AccessRequestResponse{
		AccessRequest: accessReq,
		Message:       fmt.Sprintf("Запрос на доступ от пользователя '%s' успешно отправлен администратору (филиал: %s, роль: %s)", accessReq.Login, accessReq.BranchName, accessReq.Role),
	}
	writeJSON(w, http.StatusAccepted, resp)
}

// @Summary Проверить статус запроса на доступ
// @Description Пользователь опрашивает статус своей заявки. Если одобрено — возвращает токен сессии.
// @Tags Auth
// @Produce json
// @Param request_id path int true "ID заявки (из POST /auth/request-access)"
// @Success 200 {object} map[string]interface{}
// @Failure 404 {object} map[string]string
// @Router /auth/access-requests/{request_id}/status [get]
func (h *AuthHandler) GetRequestStatus(w http.ResponseWriter, r *http.Request) {
	requestID, ok := requireID(w, r, "request_id")
	if !ok {
		return
	}

	req, err := h.svc.GetRequestStatus(r.Context(), requestID)
	if err != nil || req == nil {
		writeJSON(w, http.StatusNotFound, CommonError{Error: "Запрос не найден"})
		return
	}

	switch req.Status {
	case "approved":
		if req.SessionToken == nil || strings.TrimSpace(*req.SessionToken) == "" {
			writeJSON(w, http.StatusOK, map[string]interface{}{
				"status":  "approved",
				"message": "Доступ предоставлен, выполните вход повторно",
			})
			return
		}
		result, err := h.svc.Refresh(r.Context(), *req.SessionToken)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, CommonError{Error: err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":                   "approved",
			"access_token":             result.AccessToken,
			"refresh_token":            result.RefreshToken,
			"access_token_expires_at":  result.AccessTokenExpires,
			"refresh_token_expires_at": result.RefreshTokenExpires,
			"message":                  "Доступ предоставлен",
		})
	case "rejected":
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "rejected",
			"message": "Доступ отклонён администратором",
		})
	default:
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "pending",
			"message": "Ожидайте подтверждения администратора",
		})
	}
}

// @Summary Выход из системы
// @Description Удаляет текущую refresh-сессию пользователя.
// @Tags Auth
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} map[string]string
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" && r.Body != nil {
		var req refreshRequest
		if err := decodeJSON(r, &req); err == nil {
			token = strings.TrimSpace(req.RefreshToken)
		}
	}
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, CommonError{Error: "Токен не указан"})
		return
	}
	var session *domain.Session
	if h.svc != nil {
		session, _ = h.svc.ValidateSession(r.Context(), token)
	}

	_ = h.svc.Logout(r.Context(), token)

	if session != nil && globalAuditSvc != nil {
		ip := getClientIP(r)
		var bID *int64
		if session.BranchID > 0 {
			bID = &session.BranchID
		}
		globalAuditSvc.Log(r.Context(), session.Login, session.Role, bID, "LOGOUT", "session", nil, "Выход из системы", ip)
	}

	writeJSON(w, http.StatusOK, map[string]string{"message": "Вы успешно вышли из системы"})
}

// @Summary Список запросов на доступ (ожидающих)
// @Description Возвращает заявки со статусом pending. Доступно: Комплаенс, Администратор.
// @Tags AccessRequests
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {array} domain.AccessRequest
// @Failure 403 {object} map[string]string
// @Router /api/access-requests [get]
func (h *AuthHandler) GetAccessRequests(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.GetPendingRequests(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// @Summary История запросов на подтверждение роли и филиала
// @Description Возвращает историю обработанных заявок (только approved и rejected) с указанием кто подтвердил или отклонил. Доступно: Комплаенс, Администратор.
// @Tags AccessRequests
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {array} domain.AccessRequest
// @Failure 401 {object} CommonError
// @Failure 403 {object} CommonError
// @Router /api/access-requests/history [get]
func (h *AuthHandler) GetAccessRequestsHistory(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.GetAccessRequestsHistory(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// @Summary Одобрить запрос на доступ
// @Description Одобряет заявку и создаёт JWT access/refresh токены. Доступно: Комплаенс, Администратор.
// @Tags AccessRequests
// @Security ApiKeyAuth
// @Produce json
// @Param request_id path int true "ID заявки"
// @Success 200 {object} ports.LoginResult
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api/access-requests/{request_id}/approve [post]
func (h *AuthHandler) ApproveRequest(w http.ResponseWriter, r *http.Request) {
	requestID, ok := requireID(w, r, "request_id")
	if !ok {
		return
	}
	reviewer := GetLoginFromContext(r.Context())
	result, err := h.svc.ApproveRequest(r.Context(), requestID, reviewer)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}
	LogUserAction(r, "APPROVE_REQUEST", "access_request", &requestID, fmt.Sprintf("Одобрена заявка на доступ пользователем '%s'", reviewer))
	writeJSON(w, http.StatusOK, result)
}

// @Summary Отклонить запрос на доступ
// @Description Отклоняет заявку на доступ. Доступно: Комплаенс, Администратор.
// @Tags AccessRequests
// @Security ApiKeyAuth
// @Produce json
// @Param request_id path int true "ID заявки"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /api/access-requests/{request_id}/reject [post]
func (h *AuthHandler) RejectRequest(w http.ResponseWriter, r *http.Request) {
	requestID, ok := requireID(w, r, "request_id")
	if !ok {
		return
	}

	reviewer := GetLoginFromContext(r.Context())
	if err := h.svc.RejectRequest(r.Context(), requestID, reviewer); err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}
	LogUserAction(r, "REJECT_REQUEST", "access_request", &requestID, fmt.Sprintf("Отклонена заявка на доступ пользователем '%s'", reviewer))
	writeJSON(w, http.StatusOK, map[string]string{"message": "Запрос отклонён"})
}
