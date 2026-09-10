package dto

import "time"

type ContractReportItem struct {
	ID               int64      `json:"id"`
	BranchID         *int       `json:"branch_id"`
	BranchName       string     `json:"branch_name"`
	ContractNumber   string     `json:"contract_number"`
	ContractName     string     `json:"contract_name"`
	ContractDate     time.Time  `json:"contract_date"`
	DeliveryDate     *time.Time `json:"delivery_date,omitempty"`
	ContractEndDate  *time.Time `json:"contract_end_date,omitempty"`
	TotalAmount      float64    `json:"total_amount"`
	RemainingAmount  float64    `json:"remaining_amount"`
	ContractCurrency string     `json:"contract_currency"`
	ReceiverName     string     `json:"receiver_name"`
	ReceiverCountry  string     `json:"receiver_country"`
	Subject          string     `json:"subject"`
	InvoicesCount    int        `json:"invoices_count"`
	InvoicesAmount   float64    `json:"invoices_amount"`
	IsOverdue        bool       `json:"is_overdue"`
	CreatedBy        string     `json:"created_by"`
}

type ContractsReportResponse struct {
	TotalContracts        int                  `json:"total_contracts"`
	TotalAmountByCurrency map[string]float64   `json:"total_amount_by_currency"`
	TotalInvoicesCount    int                  `json:"total_invoices_count"`
	OverdueCount          int                  `json:"overdue_count"`
	Contracts             []ContractReportItem `json:"contracts"`
}
