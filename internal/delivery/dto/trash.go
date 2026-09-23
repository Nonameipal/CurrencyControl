package dto

import (
	"CurrencyControl/internal/domain"
	"time"
)

type TrashItem struct {
	ID             int64             `json:"id"`
	EntityType     string            `json:"entity_type"`
	EntityName     string            `json:"entity_name"`
	Number         string            `json:"number"`
	DocumentDate   *time.Time        `json:"document_date,omitempty"`
	Amount         float64           `json:"amount"`
	Currency       string            `json:"currency"`
	DocumentPath   string            `json:"document_path"`
	ClientID       *int64            `json:"client_id,omitempty"`
	ClientName     string            `json:"client_name,omitempty"`
	ContractID     *int64            `json:"contract_id,omitempty"`
	ContractNumber string            `json:"contract_number,omitempty"`
	CreatedBy      string            `json:"created_by"`
	DeletedBy      string            `json:"deleted_by,omitempty"`
	DeletedAt      time.Time         `json:"deleted_at"`
	Creator        *domain.UserBrief `json:"creator,omitempty"`
	Deleter        *domain.UserBrief `json:"deleter,omitempty"`
}

type TrashFilter struct {
	EntityType string
	BranchID   int
	Search     string
	Page       int
	PageSize   int
}

type TrashListResponse struct {
	Data     []TrashItem `json:"data"`
	Total    int         `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
}
