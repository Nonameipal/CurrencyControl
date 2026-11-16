package domain

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

const (
	ContractStatusActive   = "active"
	ContractStatusArchived = "archived"
)

type ContractCurrencyBalance struct {
	Currency        string  `json:"currency"`
	TotalAmount     float64 `json:"total_amount"`
	RemainingAmount float64 `json:"remaining_amount"`
}

type Contract struct {
	ID                            int64                     `gorm:"primaryKey" db:"id" json:"id"`
	ClientID                      *int64                    `gorm:"index" db:"client_id" json:"client_id"`
	BranchID                      *int                      `gorm:"index" db:"branch_id" json:"branch_id"`
	ContractNumber                string                    `gorm:"type:varchar(255);not null;index" db:"contract_number" json:"contract_number"`
	ContractDate                  time.Time                 `gorm:"type:date;not null" db:"contract_date" json:"contract_date"`
	DeliveryDate                  time.Time                 `gorm:"type:date" db:"delivery_date" json:"delivery_date"`
	ReturnDays                    *int                      `gorm:"column:return_days" db:"return_days" json:"return_days,omitempty"`
	TotalAmount                   float64                   `gorm:"type:decimal(18,2);not null;default:0" db:"total_amount" json:"total_amount"`
	RemainingAmount               float64                   `gorm:"type:decimal(18,2);not null;default:0" db:"remaining_amount" json:"remaining_amount"`
	OriginalAmount                float64                   `gorm:"type:decimal(18,2);not null;default:0" db:"original_amount" json:"original_amount"`
	AdditionalCurrencies          []ContractCurrencyBalance `gorm:"-" json:"additional_currencies,omitempty"`
	AdditionalAgreementCurrency   *string                   `gorm:"-" json:"additional_agreement_currency,omitempty"`
	AdditionalAgreementAmount     *float64                  `gorm:"-" json:"additional_agreement_amount,omitempty"`
	AdditionalAgreementRemaining  *float64                  `gorm:"-" json:"additional_agreement_remaining,omitempty"`
	ContractCurrency              string                    `gorm:"type:char(3);not null" db:"contract_currency" json:"contract_currency"`
	Subject                   string         `gorm:"not null" db:"subject" json:"subject"`
	ContractEndDate           *time.Time     `gorm:"type:date" db:"contract_end_date" json:"contract_end_date"`
	Status                    string         `gorm:"type:varchar(20);not null;default:'active';index" db:"status" json:"status"`
	ArchivedAt                *time.Time     `gorm:"type:timestamptz" db:"archived_at" json:"archived_at,omitempty"`
	ExtendDateTo              *time.Time     `gorm:"type:date;column:extend_date_to" db:"extend_date_to" json:"extend_date_to,omitempty"`
	DocumentPath              *string        `db:"document_path" json:"document_path,omitempty"`
	CreatedBy                 string         `gorm:"type:varchar(255);default:''" db:"created_by" json:"created_by"`
	ReceiverName              string         `gorm:"type:varchar(255);default:''" db:"receiver_name" json:"receiver_name"`
	ReceiverBank              string         `gorm:"type:varchar(255);default:''" db:"receiver_bank" json:"receiver_bank"`
	ReceiverCountry           string         `gorm:"type:varchar(255);default:''" db:"receiver_country" json:"receiver_country"`
	SenderName                string         `gorm:"type:varchar(255);default:''" db:"sender_name" json:"sender_name"`
	CreatedAt                 time.Time      `gorm:"not null;default:now()" db:"created_at" json:"created_at"`
	UpdatedAt                 time.Time      `gorm:"not null;default:now()" db:"updated_at" json:"updated_at"`
	DeletedAt                 gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	UpdatedBy                 string         `gorm:"type:varchar(255);default:''" db:"updated_by" json:"updated_by,omitempty"`
	DeletedBy                 *string        `gorm:"type:varchar(255)" db:"deleted_by" json:"deleted_by,omitempty"`
	Creator                   *UserBrief     `gorm:"-" json:"creator,omitempty"`
	Updater                   *UserBrief     `gorm:"-" json:"updater,omitempty"`
	Deleter                   *UserBrief     `gorm:"-" json:"deleter,omitempty"`
}

func (c *Contract) ValidateDates() error {
	if c.ContractDate.IsZero() || c.DeliveryDate.IsZero() || c.ContractEndDate == nil || c.ContractEndDate.IsZero() {
		return nil 
	}
	dContract := time.Date(c.ContractDate.Year(), c.ContractDate.Month(), c.ContractDate.Day(), 0, 0, 0, 0, time.UTC)
	dDelivery := time.Date(c.DeliveryDate.Year(), c.DeliveryDate.Month(), c.DeliveryDate.Day(), 0, 0, 0, 0, time.UTC)
	dEnd := time.Date(c.ContractEndDate.Year(), c.ContractEndDate.Month(), c.ContractEndDate.Day(), 0, 0, 0, 0, time.UTC)

	if dDelivery.Before(dContract) {
		return fmt.Errorf("срок поставки товара не может быть раньше даты контракта")
	}
	if dEnd.Before(dContract) {
		return fmt.Errorf("дата окончания контракта не может быть раньше даты контракта")
	}
	if dDelivery.After(dEnd) {
		return fmt.Errorf("срок поставки товара не может быть позже даты окончания контракта")
	}
	if dEnd.Before(dDelivery) {
		return fmt.Errorf("дата окончания контракта не может быть раньше срока поставки товара")
	}
	return nil
}

func (c *Contract) ValidateDocumentDate(docName string, docDate time.Time) error {
	if docDate.IsZero() {
		return nil
	}
	dDoc := time.Date(docDate.Year(), docDate.Month(), docDate.Day(), 0, 0, 0, 0, time.UTC)

	if !c.ContractDate.IsZero() {
		dStart := time.Date(c.ContractDate.Year(), c.ContractDate.Month(), c.ContractDate.Day(), 0, 0, 0, 0, time.UTC)
		if dDoc.Before(dStart) {
			return fmt.Errorf("дата документа '%s' (%s) не может быть раньше даты контракта (%s)",
				docName, docDate.Format("02.01.2006"), c.ContractDate.Format("02.01.2006"))
		}
	}

	effEnd := c.GetEffectiveEndDate()
	if effEnd != nil && !effEnd.IsZero() {
		dEnd := time.Date(effEnd.Year(), effEnd.Month(), effEnd.Day(), 0, 0, 0, 0, time.UTC)
		if dDoc.After(dEnd) {
			return fmt.Errorf("дата документа '%s' (%s) не может превышать дату окончания контракта (%s)",
				docName, docDate.Format("02.01.2006"), effEnd.Format("02.01.2006"))
		}
	}

	return nil
}

func (c *Contract) GetEffectiveEndDate() *time.Time {
	if c.ExtendDateTo != nil && !c.ExtendDateTo.IsZero() {
		if c.ContractEndDate != nil && !c.ContractEndDate.IsZero() && c.ContractEndDate.After(*c.ExtendDateTo) {
			return c.ContractEndDate
		}
		return c.ExtendDateTo
	}
	if c.ContractEndDate != nil && !c.ContractEndDate.IsZero() {
		return c.ContractEndDate
	}
	return nil
}

func (c *Contract) IsExpired(targetDate time.Time) (bool, *time.Time) {
	exp := c.GetEffectiveEndDate()
	if exp == nil || exp.IsZero() {
		return false, nil
	}
	if targetDate.IsZero() {
		targetDate = time.Now()
	}
	dTarget := time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, time.UTC)
	dExp := time.Date(exp.Year(), exp.Month(), exp.Day(), 0, 0, 0, 0, time.UTC)
	if dTarget.After(dExp) {
		return true, exp
	}
	return false, exp
}
