package dto

import (
	"time"

	"CurrencyControl/internal/domain"
)

type DashboardSearchRequest struct {
	Query      string `json:"query,omitempty"`
	SearchType string `json:"search_type,omitempty"`
	BranchID   int    `json:"branch_id"`
}

type DashboardSearchResult struct {
	ID         int64             `json:"id"`
	Number     string            `json:"number"`
	LLC        string            `json:"llc,omitempty"`
	INN        string            `json:"inn"`
	ClientType string            `json:"client_type"`
	Phones     []string          `json:"phones"`
	CreatedBy  string            `json:"created_by"`
	CreatedAt  time.Time         `json:"created_at"`
	UpdatedAt  time.Time         `json:"updated_at"`
	Creator    *domain.UserBrief `json:"creator,omitempty"`
}

