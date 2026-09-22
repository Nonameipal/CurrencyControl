package dto

import "time"

type DashboardSearchRequest struct {
	CompanyName string  `json:"company_name"`
	Amount      float64 `json:"amount"`
	INN         string  `json:"inn"`
	BranchID    int     `json:"branch_id"`
}

type DashboardSearchResult struct {
	ID         int64     `json:"id"`
	Number     string    `json:"number"`
	Name       string    `json:"name,omitempty"`     
	LLC        string    `json:"llc,omitempty"`    
	INN        string    `json:"inn"`                 
	ClientType string    `json:"client_type"`        
	Phones     []string  `json:"phones"`              
	CreatedBy  string    `json:"created_by"`            
	CreatedAt  time.Time `json:"created_at"`            
	UpdatedAt  time.Time `json:"updated_at"`            
}
