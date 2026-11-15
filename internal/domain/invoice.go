package domain

import (
	"gorm.io/gorm"
	"time"
)

type Invoice struct {
	ID                        int64          `gorm:"primaryKey" db:"id" json:"id"`
	ContractID                int64          `gorm:"not null;index" db:"contract_id" json:"contract_id"`
	InvoiceNumber             string         `gorm:"type:varchar(255);not null;index" db:"invoice_number" json:"invoice_number"`
	InvoiceDate               time.Time      `gorm:"type:date;not null" db:"invoice_date" json:"invoice_date"`
	Amount                    float64        `gorm:"type:decimal(18,2);not null" db:"amount" json:"amount"`
	Currency                  string         `gorm:"not null" db:"currency" json:"currency"`
	DocumentPath              *string        `db:"document_path" json:"document_path,omitempty"`
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

type InvoiceWithDetails struct {
	Invoice
	GTD                    *GTD           `json:"gtd,omitempty"`
	Acts                   []Act          `json:"acts,omitempty"`
	PaymentOrders          []PaymentOrder `json:"payment_orders,omitempty"`
	PaidAmount             float64        `json:"paid_amount"`
	RemainingPaymentAmount float64        `json:"remaining_payment_amount"`
}
