package http
import(
		"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/errs"

	"github.com/gorilla/mux"
)

// @Summary Список доп. соглашений контракта
// @Description Возвращает список дополнительных соглашений по контракту.
// @Tags AdditionalAgreements
// @Produce json
// @Param Login header string true "Логин пользователя"
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Success 200 {array} domain.AdditionalAgreement
// @Failure 401 {object} map[string]string
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements [get]
func (h *InvoiceHandler) GetAdditionalAgreements(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}
	contractID, err := strconv.ParseInt(mux.Vars(r)["contract_id"], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID контракта"})
		return
	}
	list, err := h.addlSvc.GetByContractID(r.Context(), contractID)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, list)
}

// @Summary Создать доп. соглашение
// @Description Создает доп. соглашение к контракту.
// @Tags AdditionalAgreements
// @Accept multipart/form-data
// @Produce json
// @Param Login header string true "Логин пользователя"
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param document formData file true "PDF файл доп. соглашения"
// @Param agreement_number formData string false "Номер доп. соглашения"
// @Param agreement_date formData string false "Дата доп. соглашения (YYYY-MM-DD)"
// @Param delivery_conditions formData string false "Новые условия доставки"
// @Param delivery_term_days formData int false "Новый срок доставки (дней)"
// @Param return_term_days formData int false "Новый срок возврата (дней)"
// @Param subject formData string false "Предмет соглашения"
// @Param extend_date_to formData string false "Продлить срок контракта до (YYYY-MM-DD)"
// @Param foreign_amount formData number false "Сумма платежа (в валюте платежа)"
// @Param foreign_currency formData string false "Валюта платежа (например USD, EUR)"
// @Param amount_in_contract_currency formData number false "Сумма в валюте контракта (прибавляется к лимиту)"
// @Success 201 {object} domain.AdditionalAgreement
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements [post]
func (h *InvoiceHandler) CreateAdditionalAgreement(w http.ResponseWriter, r *http.Request) {
	login := GetLoginFromContext(r.Context())
	if login == "" {
		handleError(w, errs.ErrUnauthorized)
		return
	}
	contractID, err := strconv.ParseInt(mux.Vars(r)["contract_id"], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID контракта"})
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	file, handler, err := r.FormFile("document")
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "PDF файл доп. соглашения обязателен"})
		return
	}
	defer file.Close()

	ag := domain.AdditionalAgreement{
		ContractID: contractID,
		CreatedBy:  login,
	}

	if v := strings.TrimSpace(r.FormValue("agreement_number")); v != "" {
		ag.AgreementNumber = &v
	}
	if v := r.FormValue("agreement_date"); v != "" {
		if d := parseDate(v); d != nil {
			ag.AgreementDate = d
		}
	}
	if v := strings.TrimSpace(r.FormValue("delivery_conditions")); v != "" {
		ag.DeliveryConditions = &v
	}
	if v := r.FormValue("delivery_term_days"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			ag.DeliveryTermDays = &i
		}
	}
	if v := r.FormValue("return_term_days"); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			ag.ReturnTermDays = &i
		}
	}
	if v := strings.TrimSpace(r.FormValue("subject")); v != "" {
		ag.Subject = &v
	}
	if v := r.FormValue("extend_date_to"); v != "" {
		if d := parseDate(v); d != nil {
			ag.ExtendDateTo = d
		}
	}

	if v := r.FormValue("foreign_amount"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			ag.ForeignAmount = &f
		}
	}
	if v := r.FormValue("foreign_currency"); v != "" {
		upper := strings.ToUpper(v)
		ag.ForeignCurrency = &upper
	}
	if v := r.FormValue("amount_in_contract_currency"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			ag.AmountInContractCurrency = f
		}
	}

	os.MkdirAll("uploads/additional_agreements", os.ModePerm)
	uniqueFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
	filePath := filepath.Join("uploads/additional_agreements", uniqueFileName)
	dst, err := os.Create(filePath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, CommonError{Error: "Ошибка при сохранении файла"})
		return
	}
	defer dst.Close()
	io.Copy(dst, file)

	ag.DocumentPath = filePath
	ag.OriginalDocumentName = handler.Filename

	created, err := h.addlSvc.Create(r.Context(), ag)
	if err != nil {
		handleError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, created)
}

// @Summary Редактирование доп. соглашения
// @Description Позволяет администратору обновить данные доп. соглашения
// @Tags Admin
// @Accept multipart/form-data
// @Produce json
// @Param Login header string true "Логин администратора (admin)"
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param agreement_id path int true "ID доп. соглашения"
// @Param delivery_conditions formData string false "Условия поставки"
// @Param delivery_term_days formData integer false "Срок поставки"
// @Param return_term_days formData integer false "Срок возврата"
// @Param subject formData string false "Предмет соглашения"
// @Param extend_date_to formData string false "Продлить до (YYYY-MM-DD)"
// @Param foreign_amount formData number false "Сумма платежа (в валюте платежа)"
// @Param foreign_currency formData string false "Валюта платежа (например USD, EUR)"
// @Param amount_in_contract_currency formData number false "Сумма в валюте контракта (прибавляется к лимиту)"
// @Param document formData file false "Новый PDF файл"
// @Success 200 {object} domain.AdditionalAgreement
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string "Доступ запрещен"
// @Failure 500 {object} map[string]string
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id} [put]
func (h *InvoiceHandler) UpdateAdditionalAgreement(w http.ResponseWriter, r *http.Request) {
	agreementID, err := strconv.ParseInt(mux.Vars(r)["agreement_id"], 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Неверный ID соглашения"})
		return
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		handleError(w, errs.ErrInvalidRequestBody)
		return
	}

	existing, err := h.addlSvc.GetByID(r.Context(), agreementID)
	if err != nil {
		handleError(w, err)
		return
	}

	if v := strings.TrimSpace(r.FormValue("delivery_conditions")); v != "" { existing.DeliveryConditions = &v }
	if v := r.FormValue("delivery_term_days"); v != "" {
		if i, err := strconv.Atoi(v); err == nil { existing.DeliveryTermDays = &i }
	}
	if v := r.FormValue("return_term_days"); v != "" {
		if i, err := strconv.Atoi(v); err == nil { existing.ReturnTermDays = &i }
	}
	if v := strings.TrimSpace(r.FormValue("subject")); v != "" { existing.Subject = &v }
	if v := r.FormValue("extend_date_to"); v != "" {
		if d, err := time.Parse(time.DateOnly, v); err == nil { existing.ExtendDateTo = &d }
	}
	if v := r.FormValue("foreign_amount"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil { existing.ForeignAmount = &f }
	}
	if v := r.FormValue("foreign_currency"); v != "" {
		upper := strings.ToUpper(v)
		existing.ForeignCurrency = &upper
	}
	if v := r.FormValue("amount_in_contract_currency"); v != "" {
		if f, err := strconv.ParseFloat(v, 64); err == nil { existing.AmountInContractCurrency = f }
	}

	file, handler, err := r.FormFile("document")
	if err == nil {
		defer file.Close()
		os.MkdirAll("uploads/additional_agreements", os.ModePerm)
		uniqueFileName := fmt.Sprintf("%d_%s", time.Now().UnixNano(), handler.Filename)
		filePath := filepath.Join("uploads/additional_agreements", uniqueFileName)
		dst, err := os.Create(filePath)
		if err == nil {
			io.Copy(dst, file)
			dst.Close()
			existing.DocumentPath = filePath
			existing.OriginalDocumentName = handler.Filename
		}
	}

	updated, err := h.addlSvc.Update(r.Context(), existing.ID, existing)
	if err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, updated)
}
// @Summary Удаление доп. соглашения
// @Description Позволяет администратору удалить доп. соглашение 
// @Tags Admin
// @Accept json
// @Produce json
// @Param Login header string true "Логин администратора (admin)"
// @Param id path int true "ID филиала"
// @Param company_id path int true "ID компании"
// @Param contract_id path int true "ID контракта"
// @Param agreement_id path int true "ID доп. соглашения"
// @Success 200 {object} map[string]string "Сообщение об успешном удалении"
// @Failure 400 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 500 {object} map[string]string
// @Router /api/branches/{id}/dashboard/companies/{company_id}/contracts/{contract_id}/additional-agreements/{agreement_id} [delete]
func (h *InvoiceHandler) DeleteAdditionalAgreement(w http.ResponseWriter, r *http.Request) {
	agreementIDStr := mux.Vars(r)["agreement_id"]
	agreementID, err := strconv.ParseInt(agreementIDStr, 10, 64)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, CommonError{Error: "Некорректный ID доп. соглашения"})
		return
	}

	if err := h.addlSvc.SoftDelete(r.Context(), agreementID); err != nil {
		handleError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"message": "Доп. соглашение успешно удалено"})
}
