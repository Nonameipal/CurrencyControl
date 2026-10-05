package http

import (
	"log"
	"net/http"
	"strings"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"
)

type AuthHandler struct {
	svc       ports.AuthService
	branchSvc ports.BranchService
}

func NewAuthHandler(svc ports.AuthService, branchSvc ports.BranchService) *AuthHandler {
	return &AuthHandler{svc: svc, branchSvc: branchSvc}
}

type RoleInfo struct {
	Role        string `json:"role"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type loginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token"`
}


type requestAccessBody struct {
	BranchID int64  `json:"branch_id" example:"5100"`
	Role     string `json:"role" enums:"operator,branch_head,currency_control,compliance,internal_audit" example:"operator"`
}

type AccessRequestResponse struct {
	domain.AccessRequest
	Message string `json:"message"`
}

// @Summary Вход в систему
// @Description Проверяет логин/пароль через AD.
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
		handleError(w, err)
		return
	}

	log.Printf("user %s authenticated", req.Login)
	if result.Status == "active" && result.Session != nil {
		ip := getClientIP(r)
		var bID *int64
		if result.Session.BranchID > 0 {
			bID = &result.Session.BranchID
		}
		if globalAuditSvc != nil {
			globalAuditSvc.Log(r.Context(), result.Session.Login, result.Session.Role, bID, "LOGIN", "session", nil, "Вход в систему через AD", ip)
		}
	}

	writeJSON(w, http.StatusOK, result)
}

// @Summary Обновить access token
// @Description Принимает refresh token и возвращает новый access token.
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body refreshRequest true "Refresh token"
// @Success 200 {object} ports.LoginResult
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Router /auth/refresh [post]
func (h *AuthHandler) Refresh(w http.ResponseWriter, r *http.Request) {
	var token string
	if r.Body != nil {
		var req refreshRequest
		if err := decodeJSON(r, &req); err == nil {
			token = strings.TrimSpace(req.RefreshToken)
		}
	}
	if token == "" {
		token = extractToken(r)
	}
	if token == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Refresh token не указан"})
		return
	}

	result, err := h.svc.Refresh(r.Context(), token)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, result)
}


