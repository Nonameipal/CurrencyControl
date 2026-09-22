package http

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"CurrencyControl/internal/errs"
	"CurrencyControl/internal/service/ports"

	"github.com/gorilla/mux"
)

type DocumentHandler struct {
	svc ports.DocumentService
}

func NewDocumentHandler(svc ports.DocumentService) *DocumentHandler {
	return &DocumentHandler{svc: svc}
}

// @Summary Просмотр или скачивание файла документа
// @Description маршрут для просмотра и скачивания загруженных файлов.
// @Description По умолчанию файл отдаётся для просмотра (inline) в браузере (например, PDF во встроенном просмотрщике).
// @Description Если указан параметр download=true (или download=1, mode=download), файл принудительно скачивается на устройство (attachment).
// @Tags Documents
// @Security ApiKeyAuth
// @Produce octet-stream
// @Produce application/pdf
// @Param entity_type path string true "Тип сущности (contract, invoice, gtd, additional_agreement, payment_order, gtd_extension)"
// @Param id path int true "ID сущности в базе данных"
// @Param download query bool false "Скачать файл (true) вместо просмотра (по умолчанию false)"
// @Success 200 {file} binary "Файл документа"
// @Failure 400 {object} CommonError "Некорректный ID или тип сущности"
// @Failure 401 {object} CommonError "Не авторизован"
// @Failure 404 {object} CommonError "Документ не найден или файл отсутствует"
// @Failure 500 {object} CommonError "Внутренняя ошибка сервера"
// @Router /api/documents/{entity_type}/{id}/file [get]
func (h *DocumentHandler) GetFile(w http.ResponseWriter, r *http.Request) {
	id, ok := requireID(w, r, "id")
	if !ok {
		return
	}

	entityType := strings.TrimSpace(mux.Vars(r)["entity_type"])
	if entityType == "" {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Параметр entity_type обязателен"})
		return
	}

	res, err := h.svc.GetDocumentFile(r.Context(), entityType, id)
	if err != nil {
		if errors.Is(err, errs.ErrNotFound) ||
			strings.Contains(err.Error(), "не найден") ||
			strings.Contains(err.Error(), "не прикреплен") ||
			strings.Contains(err.Error(), "отсутствует на сервере") {
			writeJSON(w, http.StatusNotFound, CommonError{Error: err.Error()})
			return
		}

		if strings.Contains(err.Error(), "неизвестный тип сущности") ||
			strings.Contains(err.Error(), "некорректный ID") ||
			strings.Contains(err.Error(), "недопустимый путь") ||
			strings.Contains(err.Error(), "каталог") {
			writeJSON(w, http.StatusBadRequest, CommonError{Error: err.Error()})
			return
		}

		writeJSON(w, http.StatusInternalServerError, CommonError{Error: err.Error()})
		return
	}

	isDownload := isDownloadRequest(r)

	dispositionType := "inline"
	if isDownload {
		dispositionType = "attachment"
	}

	w.Header().Set("Content-Type", res.ContentType)

	escapedFileName := url.PathEscape(res.FileName)
	w.Header().Set("Content-Disposition", fmt.Sprintf(
		`%s; filename="%s"; filename*=UTF-8''%s`,
		dispositionType,
		res.FileName,
		escapedFileName,
	))

	http.ServeFile(w, r, res.FilePath)
}

func isDownloadRequest(r *http.Request) bool {
	q := r.URL.Query()
	dl := strings.ToLower(strings.TrimSpace(q.Get("download")))
	if dl == "true" || dl == "1" || dl == "yes" {
		return true
	}
	mode := strings.ToLower(strings.TrimSpace(q.Get("mode")))
	return mode == "download"
}
