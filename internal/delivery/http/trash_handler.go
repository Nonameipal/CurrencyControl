package http

import (
	"fmt"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/service/ports"

	"github.com/gorilla/mux"
)

type TrashHandler struct {
	svc ports.TrashService
}

func NewTrashHandler(svc ports.TrashService) *TrashHandler {
	return &TrashHandler{svc: svc}
}

// @Summary Просмотр корзины удаленных документов
// @Description Возвращает список всех удаленных документов (контракты, доп. соглашения, инвойсы, ГТД). Доступно только сотрудникам Комплаенс и Валютного контроля.
// @Tags Trash
// @Security ApiKeyAuth
// @Produce json
// @Param entity_type query string false "Фильтр по типу (contract, additional_agreement, invoice, gtd)"
// @Param branch_id query int false "Фильтр по ID филиала"
// @Param search query string false "Поиск по номеру документа или контрагенту"
// @Param page query int false "Номер страницы (по умолчанию 1)"
// @Param page_size query int false "Размер страницы (по умолчанию 20, макс 100)"
// @Success 200 {object} dto.TrashListResponse
// @Failure 401 {object} CommonError "Не авторизован"
// @Failure 403 {object} CommonError "Доступ запрещен"
// @Failure 500 {object} CommonError "Внутренняя ошибка сервера"
// @Router /api/trash [get]
func (h *TrashHandler) GetTrash(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	branchID, _ := strconv.Atoi(q.Get("branch_id"))
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	filter := dto.TrashFilter{
		EntityType: strings.TrimSpace(q.Get("entity_type")),
		BranchID:   branchID,
		Search:     strings.TrimSpace(q.Get("search")),
		Page:       page,
		PageSize:   pageSize,
	}

	items, total, err := h.svc.GetTrashList(r.Context(), filter)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, dto.TrashListResponse{
		Data:     items,
		Total:    total,
		Page:     page,
		PageSize: pageSize,
	})
}

// @Summary Детальная информация об удаленном документе
// @Description Возвращает подробную карточку удаленного документа из корзины. Доступно только Комплаенс и Валютному контролю.
// @Tags Trash
// @Security ApiKeyAuth
// @Produce json
// @Param entity_type path string true "Тип сущности (contract, additional_agreement, invoice, gtd)"
// @Param id path int true "ID записи"
// @Success 200 {object} dto.TrashItem
// @Failure 400 {object} CommonError "Некорректные параметры"
// @Failure 401 {object} CommonError "Не авторизован"
// @Failure 403 {object} CommonError "Доступ запрещен"
// @Failure 404 {object} CommonError "Документ не найден"
// @Router /api/trash/{entity_type}/{id} [get]
func (h *TrashHandler) GetTrashItem(w http.ResponseWriter, r *http.Request) {
	id, ok := requireID(w, r, "id")
	if !ok {
		return
	}
	entityType := strings.TrimSpace(mux.Vars(r)["entity_type"])

	item, err := h.svc.GetTrashItem(r.Context(), entityType, id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, CommonError{Error: err.Error()})
		return
	}

	writeJSON(w, http.StatusOK, item)
}

// @Summary Просмотр или скачивание удаленного файла из корзины
// @Description Отдает файл удаленного документа (PDF/Word) для просмотра. Доступно только Комплаенс и Валютному контролю.
// @Tags Trash
// @Security ApiKeyAuth
// @Produce octet-stream
// @Param entity_type path string true "Тип сущности (contract, additional_agreement, invoice, gtd)"
// @Param id path int true "ID записи"
// @Success 200 {file} binary "Файл документа"
// @Failure 400 {object} CommonError "Некорректные параметры"
// @Failure 401 {object} CommonError "Не авторизован"
// @Failure 403 {object} CommonError "Доступ запрещен"
// @Failure 404 {object} CommonError "Файл не найден"
// @Router /api/trash/{entity_type}/{id}/file [get]
func (h *TrashHandler) ViewFile(w http.ResponseWriter, r *http.Request) {
	id, ok := requireID(w, r, "id")
	if !ok {
		return
	}
	entityType := strings.TrimSpace(mux.Vars(r)["entity_type"])

	filePath, err := h.svc.GetFilePath(r.Context(), entityType, id)
	if err != nil {
		writeJSON(w, http.StatusNotFound, CommonError{Error: err.Error()})
		return
	}

	ext := strings.ToLower(filepath.Ext(filePath))
	switch ext {
	case ".pdf":
		w.Header().Set("Content-Type", "application/pdf")
	case ".doc":
		w.Header().Set("Content-Type", "application/msword")
	case ".docx":
		w.Header().Set("Content-Type", "application/vnd.openxmlformats-officedocument.wordprocessingml.document")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}

	fileName := filepath.Base(filePath)
	w.Header().Set("Content-Disposition", fmt.Sprintf(`inline; filename="%s"`, fileName))

	http.ServeFile(w, r, filePath)
}

// @Summary Восстановление документа из корзины
// @Description Восстанавливает удаленный документ обратно в систему. Доступно только Комплаенс и Валютному контролю.
// @Tags Trash
// @Security ApiKeyAuth
// @Produce json
// @Param entity_type path string true "Тип сущности (contract, additional_agreement, invoice, gtd)"
// @Param id path int true "ID записи"
// @Success 200 {object} map[string]string "Сообщение об успешном восстановлении"
// @Failure 400 {object} CommonError "Некорректные параметры"
// @Failure 401 {object} CommonError "Не авторизован"
// @Failure 403 {object} CommonError "Доступ запрещен"
// @Failure 500 {object} CommonError "Внутренняя ошибка сервера"
// @Router /api/trash/{entity_type}/{id}/restore [post]
func (h *TrashHandler) RestoreItem(w http.ResponseWriter, r *http.Request) {
	id, ok := requireID(w, r, "id")
	if !ok {
		return
	}
	entityType := strings.TrimSpace(mux.Vars(r)["entity_type"])

	if err := h.svc.RestoreItem(r.Context(), entityType, id); err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
		return
	}

	LogUserAction(r, "RESTORE", entityType, &id, fmt.Sprintf("Восстановление %s из корзины", entityType))

	writeJSON(w, http.StatusOK, map[string]string{
		"message": fmt.Sprintf("Документ (%s ID: %d) успешно восстановлен из корзины", entityType, id),
	})
}
