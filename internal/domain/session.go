package domain

import (
	"time"
)
type Session struct {
	ID        int64     `gorm:"primaryKey"                   json:"id"`
	Token     string    `gorm:"uniqueIndex;not null"         json:"-"`
	Login     string    `gorm:"not null;index"               json:"login"`
	LastName  string    `gorm:"type:varchar(255);default:''" json:"last_name,omitempty"`
	FirstName string    `gorm:"type:varchar(255);default:''" json:"first_name,omitempty"`
	Email     string    `gorm:"type:varchar(255);default:''" json:"email,omitempty"`
	Role      string    `gorm:"not null"                     json:"role"`
	BranchID  int64     `gorm:"not null"                     json:"branch_id"`
	ExpiresAt time.Time `gorm:"not null"                     json:"expires_at"`
	CreatedAt time.Time `gorm:"not null;default:now()"       json:"created_at"`
}

