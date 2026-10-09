package http

import (
	"net/http"
	"strconv"
	"strings"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/service/ports"
)

type MyDocumentsHandler struct {
	svc ports.MyDocumentsService
}

func NewMyDocumentsHandler(svc ports.MyDocumentsService) *MyDocumentsHandler {
	return &MyDocumentsHandler{svc: svc}
}


//
// @Summary      Мои документы
// @Description  Список документов (контракт, инвойс, ГТД, доп.соглашение) созданных текущим пользователем. Включает все статусы: на проверке, на доработке, одобренные, отклонённые.
// @Tags         MyDocuments
// @Security     ApiKeyAuth
// @Produce      json
// @Param        scope       query  string  false  "Фильтр: mine (только мои, по умолчанию) или all (все документы)"
// @Param        status      query  string  false  "Фильтр по статусу: all, pending_currency_control, pending_compliance, revision_required, approved, rejected"
// @Param        entity_type query  string  false  "Тип документа: all, contract, invoice, gtd, additional_agreement"
// @Param        page        query  int     false  "Номер страницы (по умолчанию 1)"
// @Param        page_size   query  int     false  "Размер страницы (по умолчанию 20, макс 100)"
// @Success      200  {object}  dto.MyDocumentsResponse
// @Failure      401  {object}  CommonError
// @Failure      500  {object}  CommonError
// @Router       /api/my/documents [get]
func (h *MyDocumentsHandler) GetMyDocuments(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	pageSize, _ := strconv.Atoi(q.Get("page_size"))
	
	scope := strings.TrimSpace(q.Get("scope"))
	if scope == "" {
		scope = "all"
	}

	filter := dto.MyDocumentsFilter{
		Status:     strings.TrimSpace(q.Get("status")),
		EntityType: strings.TrimSpace(q.Get("entity_type")),
		Scope:      scope,
		Page:       page,
		PageSize:   pageSize,
	}
	res, err := h.svc.GetMyDocuments(r.Context(), login, filter)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, res)
}
