package http
import (
	"fmt"
	"net/http"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/errs"
)
// @Summary Список всех пользователей системы
// @Description Возвращает список всех зарегистрированных пользователей с их ролью и филиалом. Доступно: Комплаенс, Администратор.
// @Tags Users
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {array} domain.User
// @Failure 401 {object} CommonError
// @Failure 403 {object} CommonError
// @Router /api/users [get]
func (h *AuthHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	users, err := h.svc.GetAllUsers(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, users)
}

// @Summary Добавить пользователя
// @Description Создает нового пользователя с указанной ролью и филиалом. Доступно: Комплаенс, Администратор.
// @Tags Users
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param body body domain.CreateUserRequest true "Данные пользователя"
// @Success 201 {object} domain.User
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 403 {object} CommonError
// @Router /api/users [post]
func (h *AuthHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var req domain.CreateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректное тело запроса"})
		return
	}

	created, err := h.svc.CreateUser(r.Context(), req)
	if err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "CREATE", "user", &created.ID, fmt.Sprintf("Создан пользователь: %s (роль: %s, филиал: %d)", created.Login, created.Role, created.BranchID))
	writeJSON(w, http.StatusCreated, created)
}

// @Summary Изменить роль и филиал пользователя
// @Description Обновляет роль и/или филиал пользователя. Доступно: Комплаенс, Администратор.
// @Tags Users
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param id path int true "ID пользователя"
// @Param body body domain.UpdateUserRequest true "Новая роль и/или филиал"
// @Success 200 {object} domain.User
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 404 {object} CommonError
// @Router /api/users/{id} [put]
func (h *AuthHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireID(w, r, "id")
	if !ok {
		return
	}

	var req domain.UpdateUserRequest
	if err := decodeJSON(r, &req); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	updated, err := h.svc.UpdateUser(r.Context(), userID, req)
	if err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "UPDATE", "user", &updated.ID, fmt.Sprintf("Обновлен пользователь: %s (роль: %s, филиал: %d)", updated.Login, updated.Role, updated.BranchID))
	writeJSON(w, http.StatusOK, updated)
}

// @Summary Удалить пользователя
// @Description Удаляет пользователя из системы и отзывает все его сессии. Доступно: Комплаенс, Администратор.
// @Tags Users
// @Security ApiKeyAuth
// @Produce json
// @Param id path int true "ID пользователя"
// @Success 200 {object} map[string]string
// @Failure 400 {object} CommonError
// @Failure 401 {object} CommonError
// @Failure 403 {object} CommonError
// @Failure 404 {object} CommonError
// @Router /api/users/{id} [delete]
func (h *AuthHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireID(w, r, "id")
	if !ok {
		return
	}

	if err := h.svc.DeleteUser(r.Context(), userID); err != nil {
		handleError(w, err)
		return
	}

	LogUserAction(r, "DELETE", "user", &userID, fmt.Sprintf("Удален пользователь ID %d", userID))
	writeJSON(w, http.StatusOK, map[string]string{"message": "Пользователь успешно удален"})
}

func isValidRole(role string) bool {
	return domain.IsAssignableRole(role)
}
