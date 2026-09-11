package domain

import (
	"time"
)

const (
	RoleAdmin              = "admin"
	RoleOperator           = "operator"
	RoleBranchHead         = "branch_head"
	RoleCurrencyControl    = "currency_control"
	RoleCurrencyController = "currency_controller" 
	RoleCompliance         = "compliance"
	RoleInternalAudit      = "internal_audit"
)

type User struct {
	ID        int64     `gorm:"primaryKey"          json:"id"`
	Login     string    `gorm:"uniqueIndex;not null" json:"login"`
	Role      string    `gorm:"not null"            json:"role"`
	BranchID  int64     `gorm:"not null;index"      json:"branch_id"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
}


type AccessRequest struct {
	ID           int64      `gorm:"primaryKey"                 json:"id"`
	Login        string     `gorm:"not null;index"             json:"login"`
	BranchID     int64      `gorm:"not null"                   json:"branch_id"`
	BranchName   string     `gorm:"-"                          json:"branch_name,omitempty"`
	Role         string     `gorm:"not null;default:'operator'" json:"role"`
	Status       string     `gorm:"not null;default:'pending'" json:"status"`
	SessionToken *string    `gorm:"uniqueIndex"                json:"session_token,omitempty"`
	CreatedAt    time.Time  `gorm:"not null;default:now()"     json:"created_at"`
	ReviewedAt   *time.Time `json:"reviewed_at"`
}


type Session struct {
	ID        int64     `gorm:"primaryKey"           json:"id"`
	Token     string    `gorm:"uniqueIndex;not null" json:"token"`
	Login     string    `gorm:"not null;index"       json:"login"`
	Role      string    `gorm:"not null"             json:"role"`
	BranchID  int64     `gorm:"not null"             json:"branch_id"`
	ExpiresAt time.Time `gorm:"not null"             json:"expires_at"`
	CreatedAt time.Time `gorm:"not null;default:now()" json:"created_at"`
}
