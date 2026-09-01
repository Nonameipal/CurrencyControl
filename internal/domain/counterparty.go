package domain

import "time"

type Counterparty struct {
	ID           int64     `db:"id"`
	BranchID     int       `db:"branch_id"`
	Name         string    `db:"name"`
	INN          *string   `db:"inn"`
	Email        string    `db:"email"`
	CreatedBy    string    `db:"created_by"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}
