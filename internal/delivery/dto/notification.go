package dto

import "time"

type NotificationResponse struct {
	CompanyID        int64     `json:"company_id"`
	CompanyName      string    `json:"company_name"`
	ContractID       int64     `json:"contract_id"`
	ContractNumber   string    `json:"contract_number"`
	EffectiveEndDate time.Time `json:"effective_end_date"`
	DaysLeft         int       `json:"days_left"`
}