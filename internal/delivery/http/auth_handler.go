package http

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service"

	"github.com/gorilla/mux"
)

type AuthHandler struct {
	svc service.AuthService
}

func NewAuthHandler(svc service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type requestAccessBody struct {
	Login    string `json:"login"`
	BranchID int64  `json:"branch_id"`
	Role     string `json:"role"`
}

// @Summary Вход в систему
// @Description Проверяет логин/пароль через AD. Возвращает токен сессии если пользователь существует, или статус "no_role" если нужно выбрать филиал и роль.
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body loginRequest true "Логин и пароль"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 503 {object} map[string]string
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректное тело запроса"})
		return
	}
	if strings.TrimSpace(req.Login) == "" || strings.TrimSpace(req.Password) == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поля login и password обязательны"})
		return
	}

	result, err := h.svc.Login(r.Context(), req.Login, req.Password)
	if err != nil {
		log.Printf("authorization error for %s: %v", req.Login, err)
		status := http.StatusUnauthorized
		if strings.Contains(err.Error(), "недоступен") {
			status = http.StatusServiceUnavailable
		}
		writeJSON(w, status, CommonError{Error: err.Error()})
		return
	}

	log.Printf("user %s authenticated", req.Login)
	writeJSON(w, http.StatusOK, result)
}

// @Summary Запрос на доступ (для новых пользователей)
// @Description Новый пользователь, прошедший AD-проверку, указывает свой филиал и роль. Запрос уходит администратору.
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body requestAccessBody true "Логин, ID филиала, роль"
// @Success 202 {object} domain.AccessRequest
// @Failure 400 {object} map[string]string
// @Router /auth/request-access [post]
func (h *AuthHandler) RequestAccess(w http.ResponseWriter, r *http.Request) {
	var req requestAccessBody
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректное тело запроса"})
		return
	}
	if strings.TrimSpace(req.Login) == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле login обязательно"})
		return
	}
	if req.BranchID <= 0 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле branch_id обязательно"})
		return
	}
	if !isValidRole(req.Role) {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Недопустимая роль. Допустимые: operator, currency_controller, compliance"})
		return
	}

	accessReq, err := h.svc.RequestAccess(r.Context(), req.Login, req.BranchID, req.Role)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusAccepted, accessReq)
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
	requestID, err := strconv.ParseInt(mux.Vars(r)["request_id"], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID запроса"})
		return
	}

	req, err := h.svc.GetRequestStatus(r.Context(), requestID)
	if err != nil || req == nil {
		writeJSON(w, http.StatusNotFound, CommonError{Error: "Запрос не найден"})
		return
	}

	switch req.Status {
	case "approved":
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"status":  "approved",
			"token":   req.SessionToken,
			"message": "Доступ предоставлен",
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
// @Description Удаляет текущую сессию пользователя.
// @Tags Auth
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {object} map[string]string
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	token := extractToken(r)
	if token == "" {
		writeJSON(w, http.StatusUnauthorized, CommonError{Error: "Токен сессии не указан"})
		return
	}
	_ = h.svc.Logout(r.Context(), token)
	writeJSON(w, http.StatusOK, map[string]string{"message": "Вы успешно вышли из системы"})
}

// @Summary Список запросов на доступ (ожидающих)
// @Description Только для Администратора. Возвращает заявки со статусом pending.
// @Tags Admin
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {array} domain.AccessRequest
// @Failure 403 {object} map[string]string
// @Router /admin/access-requests [get]
func (h *AuthHandler) GetAccessRequests(w http.ResponseWriter, r *http.Request) {
	list, err := h.svc.GetPendingRequests(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// @Summary Одобрить запрос на доступ
// @Description Только для Администратора. Одобряет заявку, создаёт сессию.
// @Tags Admin
// @Security ApiKeyAuth
// @Produce json
// @Param request_id path int true "ID заявки"
// @Success 200 {object} domain.Session
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /admin/access-requests/{request_id}/approve [post]
func (h *AuthHandler) ApproveRequest(w http.ResponseWriter, r *http.Request) {
	requestID, err := strconv.ParseInt(mux.Vars(r)["request_id"], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID запроса"})
		return
	}

	sess, err := h.svc.ApproveRequest(r.Context(), requestID)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, sess)
}

// @Summary Отклонить запрос на доступ
// @Description Только для Администратора.
// @Tags Admin
// @Security ApiKeyAuth
// @Produce json
// @Param request_id path int true "ID заявки"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /admin/access-requests/{request_id}/reject [post]
func (h *AuthHandler) RejectRequest(w http.ResponseWriter, r *http.Request) {
	requestID, err := strconv.ParseInt(mux.Vars(r)["request_id"], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID запроса"})
		return
	}
	if err := h.svc.RejectRequest(r.Context(), requestID); err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Запрос отклонён"})
}

func isValidRole(role string) bool {
	switch role {
	case domain.RoleOperator, domain.RoleCurrencyController, domain.RoleCompliance:
		return true
	}
	return false
}
