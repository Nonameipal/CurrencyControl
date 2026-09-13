package domain

import (
    "time"
    "gorm.io/gorm"
)

const (
	ContractStatusActive   = "active"
	ContractStatusArchived = "archived"
)

type Contract struct {
	ID                   int64          `gorm:"primaryKey" db:"id" json:"id"`
	ClientID             *int64         `gorm:"index" db:"client_id" json:"client_id"`
	BranchID             *int           `gorm:"index" db:"branch_id" json:"branch_id"`
	ContractNumber       string         `gorm:"not null" db:"contract_number" json:"contract_number"`
	ContractDate         time.Time      `gorm:"type:date;not null" db:"contract_date" json:"contract_date"`
	DeliveryDate         time.Time      `gorm:"type:date" db:"delivery_date" json:"delivery_date"`
	ReturnTermDays       int            `db:"return_term_days" json:"return_term_days"`
	ReturnDate           *time.Time     `gorm:"type:date" db:"return_date" json:"return_date,omitempty"`
	TotalAmount          float64        `gorm:"type:decimal(18,2);not null;default:0" db:"total_amount" json:"total_amount"`
	RemainingAmount      float64        `gorm:"type:decimal(18,2);not null;default:0" db:"remaining_amount" json:"remaining_amount"`
	ContractCurrency     string         `gorm:"type:char(3);not null" db:"contract_currency" json:"contract_currency"`
	Subject              string         `gorm:"not null" db:"subject" json:"subject"`
	ContractEndDate      *time.Time     `gorm:"type:date" db:"contract_end_date" json:"contract_end_date"`
	Status               string         `gorm:"type:varchar(20);not null;default:'active'" db:"status" json:"status"`
	ArchivedAt           *time.Time     `gorm:"type:timestamptz" db:"archived_at" json:"archived_at,omitempty"`
	ExtendDateTo         *time.Time     `gorm:"type:date;column:extend_date_to" db:"extend_date_to" json:"extend_date_to,omitempty"`
	DocumentPath         *string        `db:"document_path" json:"document_path,omitempty"`
	CreatedBy            string         `gorm:"type:varchar(255);default:''" db:"created_by" json:"created_by"`
	CreatedAt            time.Time      `gorm:"not null;default:now()" db:"created_at" json:"created_at"`
	UpdatedAt            time.Time      `gorm:"not null;default:now()" db:"updated_at" json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}
