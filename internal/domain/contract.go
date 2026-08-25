package domain

import "time"

type Contract struct {
	ID                  int64     `db:"id"`
	ContractNumber      string    `db:"contract_number"`       
	ContractDate        time.Time `db:"contract_date"`         
	AdditionalAgreement string    `db:"additional_agreement"`  
	Subject             string    `db:"subject"`            
	RemainingAmount     float64   `db:"remaining_amount"`     
	ContractCurrency    string    `db:"contract_currency"`    
	ContractEndDate     *time.Time `db:"contract_end_date"`   
	CreatedAt           time.Time `db:"created_at"`
	UpdatedAt           time.Time `db:"updated_at"`
}
