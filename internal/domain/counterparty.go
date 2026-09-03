package domain

import (
    "time"
    "gorm.io/gorm"
)

type Counterparty struct {
	ID           int64          `gorm:"primaryKey" db:"id" json:"id"`
	BranchID     int            `gorm:"index" db:"branch_id" json:"branch_id"`
	Name         string         `gorm:"not null" db:"name" json:"name"`
	INN          *string        `gorm:"type:varchar(22)" db:"inn" json:"inn"`
	Email        string         `db:"email" json:"email"`
	CreatedBy    string         `gorm:"type:varchar(255);default:''" db:"created_by" json:"created_by"`
	CreatedAt    time.Time      `gorm:"not null;default:now()" db:"created_at" json:"created_at"`
	UpdatedAt    time.Time      `gorm:"not null;default:now()" db:"updated_at" json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}
