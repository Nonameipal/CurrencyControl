package dto

import (
	"time"

	"CurrencyControl/internal/domain"
)

type CreateCompanyRequest struct {
	INN string `json:"inn"`
}

type UpdateCompanyRequest struct {
	LLC        string   `json:"llc,omitempty"`
	INN        string   `json:"inn,omitempty"`
	ClientType string   `json:"client_type,omitempty"`
	Phones     []string `json:"phones,omitempty"`
}

type CompanyResponse struct {
	ID         int64             `json:"id"`
	LLC        string            `json:"llc,omitempty"`
	INN        string            `json:"inn"`
	ClientType string            `json:"client_type"`
	Phones     []string          `json:"phones"`
	BranchID   int               `json:"branch_id"`
	CreatedBy  string            `json:"created_by"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	Creator    *domain.UserBrief `json:"creator,omitempty"`
}

