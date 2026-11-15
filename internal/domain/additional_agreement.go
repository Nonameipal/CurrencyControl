package domain

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

const (
	AgreementStatusActive   = "active"
	AgreementStatusArchived = "archived"
)

type AdditionalAgreement struct {
	ID                        int64          `gorm:"primaryKey" db:"id" json:"id"`
	ContractID                int64          `gorm:"not null;index" db:"contract_id" json:"contract_id"`
	AgreementNumber           *string        `gorm:"type:varchar(255);index" db:"agreement_number" json:"agreement_number"`
	AgreementDate             *time.Time     `gorm:"type:date" db:"agreement_date" json:"agreement_date"`
	Subject                   *string        `db:"subject" json:"subject,omitempty"`
	DeliveryDate              *time.Time     `gorm:"type:date;column:delivery_date" db:"delivery_date" json:"delivery_date,omitempty"`
	ReturnDays                *int           `gorm:"column:return_days" db:"return_days" json:"return_days,omitempty"`
	ExtendDateTo              *time.Time     `gorm:"type:date;column:extend_date_to" db:"extend_date_to" json:"extend_date_to,omitempty"`
	AgreementEndDate          *time.Time     `gorm:"-" json:"agreement_end_date,omitempty"`
	Amount                    *float64       `gorm:"-" json:"amount,omitempty"`
	Currency                  *string        `gorm:"-" json:"currency,omitempty"`
	ForeignAmount             *float64       `gorm:"type:decimal(18,2)" db:"foreign_amount" json:"foreign_amount,omitempty"`
	ForeignCurrency           *string        `gorm:"column:currency" db:"currency" json:"foreign_currency,omitempty"`
	RemainingAmount           float64        `gorm:"type:decimal(18,2);not null;default:0" db:"remaining_amount" json:"remaining_amount"`
	Status                    string         `gorm:"type:varchar(20);not null;default:'active';index" db:"status" json:"status"`
	ArchivedAt                *time.Time     `gorm:"type:timestamptz" db:"archived_at" json:"archived_at,omitempty"`
	DocumentPath              string         `db:"document_path" json:"document_path"`
	CreatedBy                 string         `gorm:"type:varchar(255);default:''" db:"created_by" json:"created_by"`
	CreatedAt                 time.Time      `gorm:"not null;default:now()" db:"created_at" json:"created_at"`
	UpdatedAt                 time.Time      `gorm:"not null;default:now()" db:"updated_at" json:"updated_at"`
	DeletedAt                 gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	UpdatedBy                 string         `gorm:"type:varchar(255);default:''" db:"updated_by" json:"updated_by,omitempty"`
	DeletedBy                 *string        `gorm:"type:varchar(255)" db:"deleted_by" json:"deleted_by,omitempty"`
	Creator                   *UserBrief     `gorm:"-" json:"creator,omitempty"`
	Updater                   *UserBrief     `gorm:"-" json:"updater,omitempty"`
	Deleter                   *UserBrief     `gorm:"-" json:"deleter,omitempty"`
}

func (ag *AdditionalAgreement) Normalize() {
	if ag.Amount == nil {
		ag.Amount = ag.ForeignAmount
	} else if ag.ForeignAmount == nil {
		ag.ForeignAmount = ag.Amount
	}

	if ag.Currency == nil {
		ag.Currency = ag.ForeignCurrency
	} else if ag.ForeignCurrency == nil {
		ag.ForeignCurrency = ag.Currency
	}

	if ag.AgreementEndDate == nil {
		ag.AgreementEndDate = ag.ExtendDateTo
	} else if ag.ExtendDateTo == nil {
		ag.ExtendDateTo = ag.AgreementEndDate
	}

	if ag.RemainingAmount == 0 && ag.Amount != nil {
		ag.RemainingAmount = *ag.Amount
	}
}

func (a *AdditionalAgreement) ValidateDates() error {
	if a.AgreementDate == nil || a.AgreementDate.IsZero() || a.DeliveryDate == nil || a.DeliveryDate.IsZero() || a.AgreementEndDate == nil || a.AgreementEndDate.IsZero() {
		return nil
	}
	dContract := time.Date(a.AgreementDate.Year(), a.AgreementDate.Month(), a.AgreementDate.Day(), 0, 0, 0, 0, time.UTC)
	dDelivery := time.Date(a.DeliveryDate.Year(), a.DeliveryDate.Month(), a.DeliveryDate.Day(), 0, 0, 0, 0, time.UTC)
	dEnd := time.Date(a.AgreementEndDate.Year(), a.AgreementEndDate.Month(), a.AgreementEndDate.Day(), 0, 0, 0, 0, time.UTC)

	if dDelivery.Before(dContract) {
		return fmt.Errorf("срок поставки товара не может быть раньше даты доп. соглашения")
	}
	if dEnd.Before(dContract) {
		return fmt.Errorf("дата окончания не может быть раньше даты доп. соглашения")
	}
	if dDelivery.After(dEnd) {
		return fmt.Errorf("срок поставки товара не может быть позже даты окончания")
	}
	if dEnd.Before(dDelivery) {
		return fmt.Errorf("дата окончания не может быть раньше срока поставки товара")
	}
	return nil
}
