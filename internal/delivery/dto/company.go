package dto

import "time"


type CreateCompanyRequest struct {
	LLC string `json:"llc"`
	INN string `json:"inn" `           
}

type UpdateCompanyRequest struct {
	Name       string   `json:"name,omitempty"`
	LLC        string   `json:"llc,omitempty"`
	INN        string   `json:"inn,omitempty"`
	ClientType string   `json:"client_type,omitempty"`
	Phones     []string `json:"phones,omitempty"`
	Accounts   []string `json:"accounts,omitempty"`
}

type CompanyResponse struct {
	ID         int64     `json:"id"`
	Name       string    `json:"name"`        
	LLC        string    `json:"llc"`         
	INN        string    `json:"inn"`
	ClientType string    `json:"client_type"` 
	Phones     []string  `json:"phones"`     
	Accounts   []string  `json:"accounts"`    
	BranchID   int       `json:"branch_id"`
	CreatedBy  string    `json:"created_by"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}
