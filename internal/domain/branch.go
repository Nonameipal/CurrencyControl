package domain

import (
	"time"

	"gorm.io/gorm"
)

type Branch struct {
	ID        int            `gorm:"primaryKey" db:"id" json:"id"`
	Name      string         `gorm:"unique;not null" db:"name" json:"name"`
	CreatedBy string         `gorm:"type:varchar(255);default:''" db:"created_by" json:"created_by"`
	CreatedAt time.Time      `gorm:"not null;default:now()" db:"created_at" json:"created_at"`
	UpdatedAt time.Time      `gorm:"not null;default:now()" db:"updated_at" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
}
