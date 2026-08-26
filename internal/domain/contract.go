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
	ReceiverName        string     `db:"receiver_name"`
	ReceiverAccount     string     `db:"receiver_account"`
	ReceiverCountry     string     `db:"receiver_country"`
	AdditionalAgreement string     `db:"additional_agreement"`
	Subject             string     `db:"subject"`
	ContractEndDate     *time.Time `db:"contract_end_date"`
	CreatedAt           time.Time  `db:"created_at"`
	UpdatedAt           time.Time  `db:"updated_at"`
}
