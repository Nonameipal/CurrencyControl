package domain

import "time"

type GTD struct {
	ID        int64     `db:"id"`
	PaymentID int64     `db:"payment_id"`
	GTDNumber string    `db:"gtd_number"`
	GTDAmount float64   `db:"gtd_amount"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`
}
