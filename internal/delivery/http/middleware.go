package http

import (
	"context"
	"net/http"
	"strings"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"
)

type contextKey = domain.ContextKey

const (
	LoginContextKey     = domain.CtxKeyLogin
	RoleContextKey      = domain.CtxKeyRole
	BranchIDContextKey  = domain.CtxKeyBranchID
	LastNameContextKey  = domain.CtxKeyLastName
	FirstNameContextKey = domain.CtxKeyFirstName
	EmailContextKey     = domain.CtxKeyEmail
)


var (
	globalAuthSvc  ports.AuthService
	globalAuditSvc ports.AuditLogService
	globalPermSvc  ports.PermissionService
)

func SetAuthService(svc ports.AuthService) {
	globalAuthSvc = svc
}

func SetAuditService(svc ports.AuditLogService) {
	globalAuditSvc = svc
}

func SetPermissionService(svc ports.PermissionService) {
	globalPermSvc = svc
}

func RequireDocumentEditAccess() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := GetRoleFromContext(r.Context())
			login := GetLoginFromContext(r.Context())
			if role == "" || login == "" {
				writeJSON(w, http.StatusUnauthorized, CommonError{Error: "Неавторизованный доступ"})
				return
			}
			if role == domain.RoleAdmin || role == domain.RoleCompliance {
				next.ServeHTTP(w, r)
				return
			}
			if role == domain.RoleCurrencyControl || role == domain.RoleCurrencyController {
				if globalPermSvc != nil && globalPermSvc.CanEditFiles(r.Context(), role, login) {
					next.ServeHTTP(w, r)
					return
				}
				writeJSON(w, http.StatusForbidden, CommonError{Error: "Недостаточно прав. Требуется разрешение от сотрудника Комплаенса на редактирование файлов"})
				return
			}
			writeJSON(w, http.StatusForbidden, CommonError{Error: "Недостаточно прав для выполнения данной операции"})
		})
	}
}

func RequireDocumentDeleteAccess() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := GetRoleFromContext(r.Context())
			login := GetLoginFromContext(r.Context())
			if role == "" || login == "" {
				writeJSON(w, http.StatusUnauthorized, CommonError{Error: "Неавторизованный доступ"})
				return
			}
			if role == domain.RoleAdmin || role == domain.RoleCompliance || role == domain.RoleCurrencyControl || role == domain.RoleCurrencyController {
				next.ServeHTTP(w, r)
				return
			}
			writeJSON(w, http.StatusForbidden, CommonError{Error: "Недостаточно прав для выполнения данной операции"})
		})
	}
}

func RequireRoles(allowedRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role := GetRoleFromContext(r.Context())
			if role == "" {
				writeJSON(w, http.StatusUnauthorized, CommonError{Error: "Неавторизованный доступ"})
				return
			}
			if role == domain.RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}
			for _, allowed := range allowedRoles {
				if role == allowed {
					next.ServeHTTP(w, r)
					return
				}
			}
			writeJSON(w, http.StatusForbidden, CommonError{Error: "Недостаточно прав для выполнения данной операции"})
		})
	}
}

func LogUserAction(r *http.Request, action, entity string, entityID *int64, details string) {
	if globalAuditSvc == nil {
		return
	}
	userBrief := GetUserBriefFromContext(r.Context())
	role := GetRoleFromContext(r.Context())
	branchIDVal := GetBranchIDFromContext(r.Context())
	var branchID *int64
	if branchIDVal > 0 {
		branchID = &branchIDVal
	}
	ip := getClientIP(r)
	globalAuditSvc.Log(r.Context(), userBrief.Login, role, branchID, action, entity, entityID, details, ip, userBrief)
}

func extractToken(r *http.Request) string {
	if token := r.Header.Get("Session-Token"); token != "" {
		return token
	}
	if auth := r.Header.Get("Authorization"); auth != "" {
		if strings.HasPrefix(strings.ToLower(auth), "bearer ") {
			return strings.TrimSpace(auth[7:])
		}
		return strings.TrimSpace(auth)
	}
	return ""
}

func AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			writeJSON(w, http.StatusUnauthorized, CommonError{Error: "Необходима авторизация. Укажите access token"})
			return
		}
		sess, err := globalAuthSvc.ValidateSession(r.Context(), token)
		if err != nil || sess == nil {
			writeJSON(w, http.StatusUnauthorized, CommonError{Error: "Сессия недействительна или истекла"})
			return
		}
		if sess.Role == "pre_auth" {
			writeJSON(w, http.StatusForbidden, CommonError{Error: "Доступ не подтвержден администратором. Ожидайте одобрения заявки"})
			return
		}
		ctx := context.WithValue(r.Context(), LoginContextKey, sess.Login)
		ctx = context.WithValue(ctx, RoleContextKey, sess.Role)
		ctx = context.WithValue(ctx, BranchIDContextKey, sess.BranchID)
		ctx = context.WithValue(ctx, LastNameContextKey, sess.LastName)
		ctx = context.WithValue(ctx, FirstNameContextKey, sess.FirstName)
		ctx = context.WithValue(ctx, EmailContextKey, sess.Email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func AdminMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := extractToken(r)
		if token == "" {
			writeJSON(w, http.StatusUnauthorized, CommonError{Error: "Необходима авторизация. Укажите access token"})
			return
		}
		sess, err := globalAuthSvc.ValidateSession(r.Context(), token)
		if err != nil || sess == nil {
			writeJSON(w, http.StatusUnauthorized, CommonError{Error: "Сессия недействительна или истекла"})
			return
		}
		if sess.Role != "admin" {
			writeJSON(w, http.StatusForbidden, CommonError{Error: "Доступ запрещён. Требуются права администратора"})
			return
		}
		ctx := context.WithValue(r.Context(), LoginContextKey, sess.Login)
		ctx = context.WithValue(ctx, RoleContextKey, sess.Role)
		ctx = context.WithValue(ctx, BranchIDContextKey, sess.BranchID)
		ctx = context.WithValue(ctx, LastNameContextKey, sess.LastName)
		ctx = context.WithValue(ctx, FirstNameContextKey, sess.FirstName)
		ctx = context.WithValue(ctx, EmailContextKey, sess.Email)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func GetLoginFromContext(ctx context.Context) string {
	if login, ok := ctx.Value(LoginContextKey).(string); ok {
		return login
	}
	return ""
}

func GetRoleFromContext(ctx context.Context) string {
	if role, ok := ctx.Value(RoleContextKey).(string); ok {
		return role
	}
	return ""
}

func GetBranchIDFromContext(ctx context.Context) int64 {
	if id, ok := ctx.Value(BranchIDContextKey).(int64); ok {
		return id
	}
	return 0
}

func GetLastNameFromContext(ctx context.Context) string {
	if s, ok := ctx.Value(LastNameContextKey).(string); ok {
		return s
	}
	return ""
}

func GetFirstNameFromContext(ctx context.Context) string {
	if s, ok := ctx.Value(FirstNameContextKey).(string); ok {
		return s
	}
	return ""
}

func GetEmailFromContext(ctx context.Context) string {
	if s, ok := ctx.Value(EmailContextKey).(string); ok {
		return s
	}
	return ""
}

func GetUserBriefFromContext(ctx context.Context) domain.UserBrief {
	return domain.UserBrief{
		Login:     GetLoginFromContext(ctx),
		FirstName: GetFirstNameFromContext(ctx),
		LastName:  GetLastNameFromContext(ctx),
		Email:     GetEmailFromContext(ctx),
	}
}
func CORSMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		} else {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		}

		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD")

		defaultHeaders := "Accept, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, Session-Token, Login, Origin, Cache-Control, X-Requested-With"
		reqHeaders := r.Header.Get("Access-Control-Request-Headers")
		if reqHeaders != "" {
			w.Header().Set("Access-Control-Allow-Headers", defaultHeaders+", "+reqHeaders)
		} else {
			w.Header().Set("Access-Control-Allow-Headers", defaultHeaders)
		}

		w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Disposition, Session-Token, Authorization")
		w.Header().Set("Access-Control-Max-Age", "86400")
		w.Header().Add("Vary", "Origin, Access-Control-Request-Method, Access-Control-Request-Headers")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
