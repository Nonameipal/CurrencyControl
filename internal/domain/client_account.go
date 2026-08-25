package domain

import "time"

type ClientAccount struct {
	ID             int64     `db:"id"`
	CounterpartyID int64     `db:"counterparty_id"`
	AccountNumber  string    `db:"account_number"`
	BankName       string    `db:"bank_name"`
	BankSWIFT      string    `db:"bank_swift"`
	CurrencyID     int       `db:"currency_id"`
	CountryID      int       `db:"country_id"`
	IsActive       bool      `db:"is_active"`
	CreatedAt      time.Time `db:"created_at"`
	UpdatedAt      time.Time `db:"updated_at"`
}
