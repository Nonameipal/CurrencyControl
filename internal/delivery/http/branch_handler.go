package http

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/service/ports"

	"github.com/gorilla/mux"
)

type BranchHandler struct {
	svc ports.BranchService
}

func NewBranchHandler(svc ports.BranchService) *BranchHandler {
	return &BranchHandler{svc: svc}
}

// @Summary Создать филиал
// @Description Создает новый филиал с указанным кодом (ID) и названием. Доступно: Комплаенс, Администратор.
// @Tags Branches
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param body body dto.CreateBranchRequest true "Код и название филиала"
// @Success 201 {object} dto.BranchResponse
// @Failure 400 {object} CommonError "Некорректные данные запроса"
// @Failure 401 {object} CommonError "Не авторизован"
// @Failure 403 {object} CommonError "Доступ запрещен"
// @Failure 500 {object} CommonError "Внутренняя ошибка сервера"
// @Router /api/branches [post]
func (h *BranchHandler) Create(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	if login == "" {
		writeJSON(w, http.StatusUnauthorized, CommonError{Error: "Неавторизованный доступ"})
		return
	}

	var req dto.CreateBranchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный JSON запроса"})
		return
	}

	if req.ID <= 0 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле id обязательно и должно быть положительным числом"})
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле name обязательно"})
		return
	}

	created, err := h.svc.Create(r.Context(), login, req)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}

	res := dto.BranchResponse{
		ID:        created.ID,
		Name:      created.Name,
		CreatedBy: created.CreatedBy,
		CreatedAt: created.CreatedAt,
		UpdatedAt: created.UpdatedAt,
	}

	bID64 := int64(created.ID)
	LogUserAction(r, "CREATE", "branch", &bID64, "Создание филиала: "+created.Name)

	writeJSON(w, http.StatusCreated, res)
}

// @Summary Список всех филиалов
// @Description Возвращает полный список филиалов со всеми метаданными (создатель, даты). Доступно: всем авторизованным пользователям.
// @Tags Branches
// @Security ApiKeyAuth
// @Produce json
// @Success 200 {array} dto.BranchResponse
// @Failure 401 {object} CommonError "Не авторизован"
// @Failure 403 {object} CommonError "Доступ запрещен"
// @Failure 500 {object} CommonError "Внутренняя ошибка сервера"
// @Router /api/branches/all [get]
func (h *BranchHandler) GetAll(w http.ResponseWriter, r *http.Request) {
	branches, err := h.svc.GetAll(r.Context())
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: err.Error()})
		return
	}

	res := make([]dto.BranchResponse, 0, len(branches))
	for _, b := range branches {
		res = append(res, dto.BranchResponse{
			ID:        b.ID,
			Name:      b.Name,
			CreatedBy: b.CreatedBy,
			CreatedAt: b.CreatedAt,
			UpdatedAt: b.UpdatedAt,
		})
	}

	writeJSON(w, http.StatusOK, res)
}

// @Summary Получить филиал по ID
// @Description Возвращает филиал по его коду (ID). 
// @Tags Branches
// @Security ApiKeyAuth
// @Produce json
// @Param branch_id path int true "ID филиала"
// @Success 200 {object} dto.BranchResponse
// @Failure 400 {object} CommonError "Некорректный ID"
// @Failure 401 {object} CommonError "Не авторизован"
// @Failure 403 {object} CommonError "Доступ запрещен"
// @Failure 404 {object} CommonError "Филиал не найден"
// @Router /api/branches/{branch_id} [get]
func (h *BranchHandler) GetByID(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["branch_id"]
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID филиала"})
		return
	}

	b, err := h.svc.GetByID(r.Context(), id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, CommonError{Error: err.Error()})
		return
	}

	res := dto.BranchResponse{
		ID:        b.ID,
		Name:      b.Name,
		CreatedBy: b.CreatedBy,
		CreatedAt: b.CreatedAt,
		UpdatedAt: b.UpdatedAt,
	}

	writeJSON(w, http.StatusOK, res)
}

// @Summary Редактировать филиал
// @Description Обновляет название филиала по его ID. Доступно: Комплаенс, Администратор.
// @Tags Branches
// @Security ApiKeyAuth
// @Accept json
// @Produce json
// @Param branch_id path int true "ID филиала"
// @Param body body dto.UpdateBranchRequest true "Новое название филиала"
// @Success 200 {object} dto.BranchResponse
// @Failure 400 {object} CommonError "Некорректные данные"
// @Failure 401 {object} CommonError "Не авторизован"
// @Failure 403 {object} CommonError "Доступ запрещен"
// @Failure 404 {object} CommonError "Филиал не найден"
// @Failure 500 {object} CommonError "Внутренняя ошибка сервера"
// @Router /api/branches/{branch_id} [put]
func (h *BranchHandler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["branch_id"]
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID филиала"})
		return
	}

	var req dto.UpdateBranchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный JSON запроса"})
		return
	}

	if strings.TrimSpace(req.Name) == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Поле name обязательно"})
		return
	}

	updated, err := h.svc.Update(r.Context(), id, req)
	if err != nil {
		handleError(w, err)
		return
	}

	res := dto.BranchResponse{
		ID:        updated.ID,
		Name:      updated.Name,
		CreatedBy: updated.CreatedBy,
		CreatedAt: updated.CreatedAt,
		UpdatedAt: updated.UpdatedAt,
	}

	bID64 := int64(updated.ID)
	LogUserAction(r, "UPDATE", "branch", &bID64, "Обновление филиала: "+updated.Name)

	writeJSON(w, http.StatusOK, res)
}

// @Summary Удалить филиал
// @Description удаление филиала. Если к филиалу привязаны пользователи, компании или заявки, удаление отклоняется. Доступно: Комплаенс, Администратор.
// @Tags Branches
// @Security ApiKeyAuth
// @Produce json
// @Param branch_id path int true "ID филиала"
// @Success 200 {object} map[string]string "Сообщение об успешном удалении"
// @Failure 400 {object} CommonError "Некорректный ID или филиал используется"
// @Failure 401 {object} CommonError "Не авторизован"
// @Failure 403 {object} CommonError "Доступ запрещен"
// @Failure 404 {object} CommonError "Филиал не найден"
// @Failure 500 {object} CommonError "Внутренняя ошибка сервера"
// @Router /api/branches/{branch_id} [delete]
func (h *BranchHandler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := mux.Vars(r)["branch_id"]
	id, err := strconv.Atoi(idStr)
	if err != nil || id <= 0 {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID филиала"})
		return
	}

	if err := h.svc.Delete(r.Context(), id); err != nil {
		handleError(w, err)
		return
	}

	bID64 := int64(id)
	LogUserAction(r, "DELETE", "branch", &bID64, "Удаление филиала")

	writeJSON(w, http.StatusOK, map[string]string{"message": "Филиал успешно удален"})
}
