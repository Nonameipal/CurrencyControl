package domain

import "time"

type InvoiceSpec struct {
	ID         int64     `db:"id"`
	PaymentID  int64     `db:"payment_id"`
	SpecName   string    `db:"spec_name"`
	SpecDate   time.Time `db:"spec_date"`
	SpecAmount float64   `db:"spec_amount"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}
