package domain

import "time"

type Counterparty struct {
	ID           int64     `db:"id"`
	ClientNumber *string   `db:"client_number"` // Номер, который пишут пользователи самостоятельно
	Name         string    `db:"name"`
	INN          *string   `db:"inn"`
	Email        string    `db:"email"`
	IsThirdParty bool      `db:"is_third_party"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}
