package domain

import "time"

type Contract struct {
	ID                  int64      `db:"id"`
	ClientID            *int64     `db:"client_id"`
	BranchID            *int       `db:"branch_id"`
	ContractNumber      string     `db:"contract_number"`
	ContractName        string     `db:"contract_name"`
	ContractDate        time.Time  `db:"contract_date"`
	DeliveryDate        time.Time  `db:"delivery_date"`
	DeliveryConditions  string     `db:"delivery_conditions"`
	DeliveryTermDays    int        `db:"delivery_term_days"`
	ReturnTermDays      int        `db:"return_term_days"`
	TotalAmount         float64    `db:"total_amount"`
	RemainingAmount     float64    `db:"remaining_amount"`
	ContractCurrency    string     `db:"contract_currency"`
	SenderAccount       string     `db:"sender_account"`
	ReceiverName        string     `db:"receiver_name"`
	ReceiverAccount     string     `db:"receiver_account"`
	ReceiverCountry     string     `db:"receiver_country"`
	AdditionalAgreement string     `db:"additional_agreement"`
	Subject             string     `db:"subject"`
	ContractEndDate     *time.Time `db:"contract_end_date"`
	DocumentPath        *string    `db:"document_path" json:"document_path,omitempty"`
	OriginalDocumentName *string    `db:"original_document_name" json:"original_document_name,omitempty"`
	CreatedBy           string     `db:"created_by" json:"created_by"`
	CreatedAt           time.Time  `db:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at"`
}
