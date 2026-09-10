package domain

import (
	"time"
)

type AuditLog struct {
	ID        int64     `gorm:"primaryKey" json:"id"`
	UserLogin string    `gorm:"type:varchar(255);not null;index" json:"user_login"`
	Role      string    `gorm:"type:varchar(50);not null" json:"role"`
	BranchID  *int64    `gorm:"index" json:"branch_id,omitempty"`
	Action    string    `gorm:"type:varchar(100);not null;index" json:"action"`
	Entity    string    `gorm:"type:varchar(100);not null" json:"entity"`
	EntityID  *int64    `json:"entity_id,omitempty"`
	Details   string    `gorm:"type:text" json:"details,omitempty"`
	IPAddress string    `gorm:"type:varchar(50)" json:"ip_address,omitempty"`
	CreatedAt time.Time `gorm:"not null;default:now();index" json:"created_at"`
}
