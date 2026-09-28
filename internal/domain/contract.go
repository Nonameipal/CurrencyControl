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

type Contract struct {
	ID                        int64          `gorm:"primaryKey" db:"id" json:"id"`
	ClientID                  *int64         `gorm:"index" db:"client_id" json:"client_id"`
	BranchID                  *int           `gorm:"index" db:"branch_id" json:"branch_id"`
	ContractNumber            string         `gorm:"not null" db:"contract_number" json:"contract_number"`
	ContractDate              time.Time      `gorm:"type:date;not null" db:"contract_date" json:"contract_date"`
	DeliveryDate              time.Time      `gorm:"type:date" db:"delivery_date" json:"delivery_date"`
	ReturnDays                *int           `gorm:"column:return_days" db:"return_days" json:"return_days,omitempty"`
	TotalAmount               float64        `gorm:"type:decimal(18,2);not null;default:0" db:"total_amount" json:"total_amount"`
	RemainingAmount           float64        `gorm:"type:decimal(18,2);not null;default:0" db:"remaining_amount" json:"remaining_amount"`
	ContractCurrency          string         `gorm:"type:char(3);not null" db:"contract_currency" json:"contract_currency"`
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
	SenderBank                string         `gorm:"type:varchar(255);default:''" db:"sender_bank" json:"sender_bank"`
	SenderCountry             string         `gorm:"type:varchar(255);default:''" db:"sender_country" json:"sender_country"`
	ApprovalStatus            string         `gorm:"type:varchar(50);not null;default:'pending_currency_control';index" db:"approval_status" json:"approval_status"`
	CurrencyControlDecision   string         `gorm:"type:varchar(50);default:''" db:"currency_control_decision" json:"currency_control_decision,omitempty"`
	CurrencyControlComment    string         `gorm:"type:text;default:''" db:"currency_control_comment" json:"currency_control_comment,omitempty"`
	CurrencyControlReviewedBy string         `gorm:"type:varchar(255);default:''" db:"currency_control_reviewed_by" json:"currency_control_reviewed_by,omitempty"`
	CurrencyControlReviewedAt *time.Time     `gorm:"type:timestamptz" db:"currency_control_reviewed_at" json:"currency_control_reviewed_at,omitempty"`
	ComplianceDecision        string         `gorm:"type:varchar(50);default:''" db:"compliance_decision" json:"compliance_decision,omitempty"`
	ComplianceComment         string         `gorm:"type:text;default:''" db:"compliance_comment" json:"compliance_comment,omitempty"`
	ComplianceReviewedBy      string         `gorm:"type:varchar(255);default:''" db:"compliance_reviewed_by" json:"compliance_reviewed_by,omitempty"`
	ComplianceReviewedAt      *time.Time     `gorm:"type:timestamptz" db:"compliance_reviewed_at" json:"compliance_reviewed_at,omitempty"`
	RejectionReason           string         `gorm:"type:text;default:''" db:"rejection_reason" json:"rejection_reason,omitempty"`
	CreatedAt                 time.Time      `gorm:"not null;default:now()" db:"created_at" json:"created_at"`
	UpdatedAt                 time.Time      `gorm:"not null;default:now()" db:"updated_at" json:"updated_at"`
	DeletedAt                 gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	UpdatedBy                 string         `gorm:"type:varchar(255);default:''" db:"updated_by" json:"updated_by,omitempty"`
	DeletedBy                 *string        `gorm:"type:varchar(255)" db:"deleted_by" json:"deleted_by,omitempty"`
	Creator                   *UserBrief     `gorm:"-" json:"creator,omitempty"`
	Updater                   *UserBrief     `gorm:"-" json:"updater,omitempty"`
	Deleter                   *UserBrief     `gorm:"-" json:"deleter,omitempty"`
	CurrencyControlReviewer   *UserBrief     `gorm:"-" json:"currency_control_reviewer,omitempty"`
	ComplianceReviewer        *UserBrief     `gorm:"-" json:"compliance_reviewer,omitempty"`
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
