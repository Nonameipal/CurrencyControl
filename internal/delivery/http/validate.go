package http

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"CurrencyControl/internal/errs"
)

var contractRequiredFields = []string{
	"contract_number",
	"subject",
	"receiver_name",
	"receiver_bank",
	"sender_name",
	"sender_bank",
	"sender_country",
}

var invoiceRequiredFields = []string{
	"invoice_number",
	"currency",
	"sender_name",
	"sender_bank",
	"sender_country",
}

var gtdRequiredFields = []string{
	"gtd_number",
	"gtd_currency",
	"document_type",
	"sender_name",
	"sender_bank",
	"sender_country",
}

var paymentOrderRequiredFields = []string{
	"payment_order_number",
	"currency",
	"payer",
	"receiver_name",
	"receiver_bank",
	"payment_purpose",
	"sender_name",
	"sender_bank",
	"sender_country",
}

type FormValidator struct {
	r   *http.Request
	err error
}

func NewFormValidator(r *http.Request, maxMemory int64) *FormValidator {
	v := &FormValidator{r: r}
	if maxMemory > 0 {
		if err := r.ParseMultipartForm(maxMemory); err != nil {
			v.addError(errs.ErrInvalidRequestBody)
		}
	}
	return v
}

func (v *FormValidator) addError(err error) {
	if v.err == nil && err != nil {
		v.err = err
	}
}
func (v *FormValidator) RequirePositiveID(id int64, field string) {
	if id <= 0 {
		v.addError(fmt.Errorf("Поле %s обязательно", field))
	}
}

func (v *FormValidator) RequirePositiveInt(id int, field string) {
	if id <= 0 {
		v.addError(fmt.Errorf("Поле %s обязательно", field))
	}
}
func (v *FormValidator) RequireStrings(fields []string) {
	for _, field := range fields {
		if strings.TrimSpace(v.r.FormValue(field)) == "" {
			v.addError(fmt.Errorf("Поле %s обязательно", field))
			return
		}
	}
}

func (v *FormValidator) String(field string) string {
	val := strings.TrimSpace(v.r.FormValue(field))
	if val == "" {
		v.addError(fmt.Errorf("Поле %s обязательно", field))
	}
	return val
}
func (v *FormValidator) StringFallback(mainField string, fallbackFields ...string) string {
	val := getFormValueFallback(v.r, fallbackFields...)
	if val == "" {
		v.addError(fmt.Errorf("Поле %s обязательно", mainField))
	}
	return val
}
func (v *FormValidator) Date(field string) *time.Time {
	raw := strings.TrimSpace(v.r.FormValue(field))
	if raw == "" {
		v.addError(fmt.Errorf("Поле %s обязательно", field))
		return nil
	}
	d := parseDate(raw)
	if d == nil {
		v.addError(fmt.Errorf("Неверный формат %s. Ожидается дата (например 02.01.2006 или 2006-01-02)", field))
		return nil
	}
	return d
}
func (v *FormValidator) OptionalDate(field string) *time.Time {
	raw := strings.TrimSpace(v.r.FormValue(field))
	if raw == "" {
		return nil
	}
	d := parseDate(raw)
	if d == nil {
		v.addError(fmt.Errorf("Неверный формат %s. Ожидается дата (например 02.01.2006 или 2006-01-02)", field))
		return nil
	}
	return d
}

func (v *FormValidator) Float(field string) float64 {
	raw := strings.TrimSpace(v.r.FormValue(field))
	if raw == "" {
		v.addError(fmt.Errorf("Поле %s обязательно и должно быть больше 0", field))
		return 0
	}
	f, err := strconv.ParseFloat(raw, 64)
	if err != nil || f <= 0 {
		v.addError(fmt.Errorf("Поле %s обязательно и должно быть больше 0", field))
		return 0
	}
	return f
}
func (v *FormValidator) Int(field string) int {
	raw := strings.TrimSpace(v.r.FormValue(field))
	if raw == "" {
		v.addError(fmt.Errorf("Поле %s обязательно и должно быть больше 0", field))
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n <= 0 {
		v.addError(fmt.Errorf("Срок возврата денежных средств (%s) должен быть целым числом больше 0", field))
		return 0
	}
	return n
}

func (v *FormValidator) HasError() bool {
	return v.err != nil
}
func (v *FormValidator) Err() error {
	return v.err
}

func (v *FormValidator) AddError(err error) {
	v.addError(err)
}

func (v *FormValidator) Respond(w http.ResponseWriter) bool {
	if v.err != nil {
		if v.err == errs.ErrInvalidRequestBody {
			handleError(w, errs.ErrInvalidRequestBody)
			return true
		}
		writeJSON(w, http.StatusBadRequest, CommonError{Error: v.err.Error()})
		return true
	}
	return false
}
