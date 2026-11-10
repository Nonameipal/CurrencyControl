package domain

import (
	"time"

	"gorm.io/gorm"
)

type Act struct {
	ID           int64          `gorm:"primaryKey" db:"id" json:"id"`
	ContractID   int64          `gorm:"not null;index" db:"contract_id" json:"contract_id"`
	InvoiceID    int64          `gorm:"not null;index" db:"invoice_id" json:"invoice_id"`
	ActNumber    string         `gorm:"type:varchar(255);not null" db:"act_number" json:"act_number"`
	ActDate      time.Time      `gorm:"type:date;not null" db:"act_date" json:"act_date"`
	ActAmount    float64        `gorm:"type:decimal(18,2);not null" db:"act_amount" json:"act_amount"`
	Currency     string         `gorm:"type:varchar(10);not null;default:'USD'" db:"currency" json:"currency"`
	CreatedBy    string         `gorm:"type:varchar(255);not null" db:"created_by" json:"created_by"`
	DocumentPath *string        `db:"document_path" json:"document_path,omitempty"`
	CreatedAt    time.Time      `gorm:"not null;default:now()" db:"created_at" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"not null;default:now()" db:"updated_at" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	UpdatedBy    string         `gorm:"type:varchar(255);default:''" db:"updated_by" json:"updated_by,omitempty"`
	DeletedBy    *string        `gorm:"type:varchar(255)" db:"deleted_by" json:"deleted_by,omitempty"`
	Creator      *UserBrief     `gorm:"-" json:"creator,omitempty"`
	Updater      *UserBrief     `gorm:"-" json:"updater,omitempty"`
	Deleter      *UserBrief     `gorm:"-" json:"deleter,omitempty"`
}

func (Act) TableName() string {
	return "acts"
}
