package domain

import "time"

type Payment struct {
	ID                  int64      `db:"id"`
	ContractID          int64      `db:"contract_id"`
	PayerID             int64      `db:"payer_id"`
	ReceiverID          int64      `db:"receiver_id"`
	PaymentNumber       string     `db:"payment_number"`
	PaymentDate         time.Time  `db:"payment_date"`
	CurrencyCode        string     `db:"currency_code"`
	Amount              float64    `db:"amount"`
	DeliveryDate        *time.Time `db:"delivery_date"`
	DeliveryConditions  string     `db:"delivery_conditions"`
	RefundDate          *time.Time `db:"refund_date"`
	SwiftDeadline       *time.Time `db:"swift_deadline"`
	OverdueDays         int        `db:"overdue_days"`
	ReceiverCountryCode string     `db:"receiver_country_code"`
	CreatedAt           time.Time  `db:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at"`
}
