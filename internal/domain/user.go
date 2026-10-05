package domain

import (
	"context"
	"time"
)

type ContextKey string

const (
	CtxKeyLogin     ContextKey = "login"
	CtxKeyRole      ContextKey = "role"
	CtxKeyBranchID  ContextKey = "branch_id"
	CtxKeyLastName  ContextKey = "last_name"
	CtxKeyFirstName ContextKey = "first_name"
	CtxKeyEmail     ContextKey = "email"
)

func GetLoginFromCtx(ctx context.Context) string {
	if s, ok := ctx.Value(CtxKeyLogin).(string); ok {
		return s
	}
	return ""
}

func GetUserBriefFromCtx(ctx context.Context) UserBrief {
	var ub UserBrief
	if s, ok := ctx.Value(CtxKeyLogin).(string); ok {
		ub.Login = s
	}
	if s, ok := ctx.Value(CtxKeyFirstName).(string); ok {
		ub.FirstName = s
	}
	if s, ok := ctx.Value(CtxKeyLastName).(string); ok {
		ub.LastName = s
	}
	if s, ok := ctx.Value(CtxKeyEmail).(string); ok {
		ub.Email = s
	}
	return ub
}

const (
	RoleAdmin              = "admin"
	RoleOperator           = "operator"
	RoleBranchHead         = "branch_head"
	RoleCurrencyControl    = "currency_control"
	RoleCurrencyController = "currency_controller"
	RoleCompliance         = "compliance"
	RoleInternalAudit      = "internal_audit"
)

func IsValidRole(role string) bool {
	switch role {
	case RoleAdmin,
		RoleOperator,
		RoleBranchHead,
		RoleCurrencyControl,
		RoleCurrencyController,
		RoleCompliance,
		RoleInternalAudit:
		return true
	}
	return false
}

func IsAssignableRole(role string) bool {
	switch role {
	case RoleOperator,
		RoleBranchHead,
		RoleCurrencyControl,
		RoleCurrencyController,
		RoleCompliance,
		RoleInternalAudit:
		return true
	}
	return false
}

type User struct {
	ID         int64     `gorm:"primaryKey"                   json:"id"`
	Login      string    `gorm:"uniqueIndex;not null"         json:"login"`
	LastName   string    `gorm:"type:varchar(255);default:''" json:"last_name,omitempty"`
	FirstName  string    `gorm:"type:varchar(255);default:''" json:"first_name,omitempty"`
	Email      string    `gorm:"type:varchar(255);default:''" json:"email,omitempty"`
	Role       string    `gorm:"not null"                     json:"role"`
	BranchID   int64     `gorm:"not null;index"               json:"branch_id"`
	BranchName string    `gorm:"-"                            json:"branch_name,omitempty"`
	CreatedAt  time.Time `gorm:"not null;default:now()"       json:"created_at"`
}

type CreateUserRequest struct {
	Login     string `json:"login"`
	Role      string `json:"role"`
	BranchID  int64  `json:"branch_id"`
}

type UpdateUserRequest struct {
	Role     string `json:"role"`
	BranchID int64  `json:"branch_id"`
}

type UserBrief struct {
	Login     string `json:"login"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

