package response

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"CurrencyControl/internal/errs"
)

var ContractRequiredFields = []string{
	"contract_number",
	"subject",
	"receiver_name",
	"receiver_bank",
	"sender_name",
}

var InvoiceRequiredFields = []string{
	"invoice_number",
	"currency",
}

var GTDRequiredFields = []string{
	"gtd_number",
}

var ActRequiredFields = []string{
	"act_number",
}

var AdditionalAgreementRequiredFields = []string{
	"agreement_number",
	"currency",
}

var PaymentOrderRequiredFields = []string{
	"currency",
	"sender_name",
	"receiver_name",
	"receiver_bank",
	"receiver_country",
	"payment_purpose",
}

var (
	contractRequiredFields            = ContractRequiredFields
	invoiceRequiredFields             = InvoiceRequiredFields
	gtdRequiredFields                 = GTDRequiredFields
	actRequiredFields                 = ActRequiredFields
	additionalAgreementRequiredFields = AdditionalAgreementRequiredFields
	paymentOrderRequiredFields        = PaymentOrderRequiredFields
)

type FormValidator struct {
	r   *http.Request
	err error
}

func NewFormValidator(r *http.Request, maxMemory int64) *FormValidator {
	v := &FormValidator{r: r}
	if err := r.ParseMultipartForm(maxMemory); err != nil {
		v.err = errs.ErrInvalidRequestBody
	}
	return v
}

func (v *FormValidator) RequireStrings(fields []string) {
	if v.err != nil {
		return
	}
	for _, f := range fields {
		if strings.TrimSpace(v.r.FormValue(f)) == "" {
			v.err = fmt.Errorf("обязательное поле '%s' не заполнено", f)
			return
		}
	}
}

func (v *FormValidator) RequirePositiveID(id int64, name string) {
	if v.err != nil {
		return
	}
	if id <= 0 {
		v.err = fmt.Errorf("некорректный %s", name)
	}
}

func (v *FormValidator) RequirePositiveInt(val int, name string) {
	if v.err != nil {
		return
	}
	if val <= 0 {
		v.err = fmt.Errorf("некорректный %s", name)
	}
}

func (v *FormValidator) Date(field string) *time.Time {
	if v.err != nil {
		return nil
	}
	raw := strings.TrimSpace(v.r.FormValue(field))
	if raw == "" {
		v.err = fmt.Errorf("поле '%s' обязательно", field)
		return nil
	}
	d := ParseDate(raw)
	if d == nil {
		v.err = fmt.Errorf("неверный формат даты для поля '%s'", field)
		return nil
	}
	return d
}

func (v *FormValidator) OptionalDate(field string) *time.Time {
	if v.err != nil {
		return nil
	}
	raw := strings.TrimSpace(v.r.FormValue(field))
	if raw == "" {
		return nil
	}
	d := ParseDate(raw)
	if d == nil {
		v.err = fmt.Errorf("неверный формат даты для поля '%s'", field)
		return nil
	}
	return d
}

func (v *FormValidator) Float(field string) float64 {
	if v.err != nil {
		return 0
	}
	raw := strings.TrimSpace(v.r.FormValue(field))
	if raw == "" {
		v.err = fmt.Errorf("поле '%s' обязательно", field)
		return 0
	}
	clean := strings.ReplaceAll(raw, " ", "")
	clean = strings.ReplaceAll(clean, ",", ".")
	f, err := strconv.ParseFloat(clean, 64)
	if err != nil || f <= 0 {
		v.err = fmt.Errorf("поле '%s' должно быть положительным числом", field)
		return 0
	}
	return f
}

func (v *FormValidator) OptionalFloat(field string) *float64 {
	if v.err != nil {
		return nil
	}
	raw := strings.TrimSpace(v.r.FormValue(field))
	if raw == "" {
		return nil
	}
	clean := strings.ReplaceAll(raw, " ", "")
	clean = strings.ReplaceAll(clean, ",", ".")
	f, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		v.err = fmt.Errorf("некорректное числовое значение для поля '%s'", field)
		return nil
	}
	return &f
}

func (v *FormValidator) Int(field string) int {
	if v.err != nil {
		return 0
	}
	raw := strings.TrimSpace(v.r.FormValue(field))
	if raw == "" {
		v.err = fmt.Errorf("поле '%s' обязательно", field)
		return 0
	}
	clean := strings.ReplaceAll(raw, " ", "")
	n, err := strconv.Atoi(clean)
	if err != nil || n <= 0 {
		v.err = fmt.Errorf("поле '%s' должно быть целым положительным числом", field)
		return 0
	}
	return n
}

func (v *FormValidator) String(field string) string {
	return strings.TrimSpace(v.r.FormValue(field))
}

func (v *FormValidator) Respond(w http.ResponseWriter) bool {
	if v.err != nil {
		HandleError(w, v.err)
		return true
	}
	return false
}
