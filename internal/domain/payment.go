package domain

import (
	"time"
)

type Payment struct {
	ID                  int64      `gorm:"primaryKey" db:"id" json:"id"`
	ContractID          int64      `gorm:"not null;index" db:"contract_id" json:"contract_id"`
	PayerID             int64      `gorm:"not null;index" db:"payer_id" json:"payer_id"`
	ReceiverID          int64      `gorm:"not null;index" db:"receiver_id" json:"receiver_id"`
	PaymentNumber       string     `gorm:"not null" db:"payment_number" json:"payment_number"`
	PaymentDate         time.Time  `gorm:"type:date;not null" db:"payment_date" json:"payment_date"`
	CurrencyCode        string     `gorm:"type:char;not null" db:"currency_code" json:"currency_code"`
	Amount              float64    `gorm:"type:decimal(18,2);not null" db:"amount" json:"amount"`
	DeliveryDate        *time.Time `gorm:"type:date" db:"delivery_date" json:"delivery_date"`
	DeliveryConditions  string     `db:"delivery_conditions" json:"delivery_conditions"`
	RefundDate          *time.Time `gorm:"type:date" db:"refund_date" json:"refund_date"`
	SwiftDeadline       *time.Time `gorm:"type:date" db:"swift_deadline" json:"swift_deadline"`
	OverdueDays         int        `gorm:"not null;default:0" db:"overdue_days" json:"overdue_days"`
	ReceiverCountryCode string     `gorm:"type:varchar(2)" db:"receiver_country_code" json:"receiver_country_code"`
	CreatedAt           time.Time  `gorm:"not null;default:now()" db:"created_at" json:"created_at"`
	UpdatedAt           time.Time  `gorm:"not null;default:now()" db:"updated_at" json:"updated_at"`
}
