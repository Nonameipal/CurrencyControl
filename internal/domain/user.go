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
	LastName  string `json:"last_name"`
	FirstName string `json:"first_name"`
	Email     string `json:"email"`
	Role      string `json:"role"`
	BranchID  int64  `json:"branch_id"`
}

type UpdateUserRequest struct {
	Role     string `json:"role"`
	BranchID int64  `json:"branch_id"`
}

// UserBrief представляет информацию о пользователе для отображения кто создал/изменил/удалил/проверил
type UserBrief struct {
	Login     string `json:"login"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Email     string `json:"email"`
}

type AccessRequest struct {
	ID           int64      `gorm:"primaryKey"                   json:"id"`
	Login        string     `gorm:"not null;index"               json:"login"`
	LastName     string     `gorm:"type:varchar(255);default:''" json:"last_name,omitempty"`
	FirstName    string     `gorm:"type:varchar(255);default:''" json:"first_name,omitempty"`
	Email        string     `gorm:"type:varchar(255);default:''" json:"email,omitempty"`
	BranchID     int64      `gorm:"not null"                     json:"branch_id"`
	BranchName   string     `gorm:"-"                            json:"branch_name,omitempty"`
	Role         string     `gorm:"not null;default:'operator'"  json:"role"`
	Status       string     `gorm:"not null;default:'pending'"   json:"status"`
	SessionToken *string    `gorm:"uniqueIndex"                  json:"session_token,omitempty"`
	CreatedAt    time.Time  `gorm:"not null;default:now()"       json:"created_at"`
	ReviewedAt   *time.Time `json:"reviewed_at,omitempty"`
	ReviewedBy   string     `gorm:"type:varchar(255);default:''" json:"reviewed_by,omitempty"`
	Applicant    *UserBrief `gorm:"-"                            json:"applicant,omitempty"`
	Reviewer     *UserBrief `gorm:"-"                            json:"reviewer,omitempty"`
}

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

type CurrencyControlPermission struct {
	Login     string     `gorm:"primaryKey" db:"login" json:"login"`
	CanEdit   bool       `gorm:"not null;default:true" db:"can_edit" json:"can_edit"`
	CanDelete bool       `gorm:"not null;default:true" db:"can_delete" json:"can_delete"`
	GrantedBy string     `gorm:"not null" db:"granted_by" json:"granted_by"`
	GrantedAt time.Time  `gorm:"not null;default:now()" db:"granted_at" json:"granted_at"`
	User      *UserBrief `gorm:"-" json:"user,omitempty"`
	Granter   *UserBrief `gorm:"-" json:"granter,omitempty"`
}
