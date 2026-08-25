package domain

import "time"

type Counterparty struct {
	ID           int64     `db:"id"`
	Name         string    `db:"name"`
	Email        string    `db:"email"`
	IsThirdParty bool      `db:"is_third_party"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}
