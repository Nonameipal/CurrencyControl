package dto

import (
	"encoding/json"
	"strings"
	"time"

	"CurrencyControl/internal/domain"
)

type CreateCompanyRequest struct {
	INN string `json:"inn"`
}

type FlexibleStringSlice []string

func (f *FlexibleStringSlice) UnmarshalJSON(data []byte) error {
	var single string
	if err := json.Unmarshal(data, &single); err == nil {
		trimmed := strings.TrimSpace(single)
		if trimmed != "" {
			*f = []string{trimmed}
		} else {
			*f = []string{}
		}
		return nil
	}
	var slice []string
	if err := json.Unmarshal(data, &slice); err == nil {
		*f = slice
		return nil
	}
	return nil
}

type UpdateCompanyRequest struct {
	LLC        string              `json:"llc,omitempty"`
	INN        string              `json:"inn,omitempty"`
	ClientType string              `json:"client_type,omitempty"`
	Phone      string              `json:"phone,omitempty"`
	Phones     FlexibleStringSlice `json:"phones,omitempty"`
}

type AddCompanyPhonesRequest struct {
	Phones []string `json:"phones"`
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
