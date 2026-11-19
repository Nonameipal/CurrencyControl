package dto

import "CurrencyControl/internal/domain"

type MyDocumentsFilter struct {
	Status     string 
	EntityType string 
	Scope      string
	Page       int
	PageSize   int
}

type MyDocumentItem struct {
	EntityType                string            `json:"entity_type"`
	EntityID                  int64             `json:"entity_id"`
	DocumentNumber            string            `json:"document_number"`
	DocumentPath              *string           `json:"document_path,omitempty"`
	DocumentDate              string            `json:"document_date"`
	Subject                   string            `json:"subject,omitempty"`
	Amount                    float64           `json:"amount"`
	Currency                  string            `json:"currency"`
	CounterpartyName          string            `json:"counterparty_name,omitempty"`
	BranchName                string            `json:"branch_name,omitempty"`
	ContractID                *int64            `json:"contract_id,omitempty"`
	ContractNumber            string            `json:"contract_number,omitempty"`
	InvoiceID                 *int64            `json:"invoice_id,omitempty"`
	InvoiceNumber             string            `json:"invoice_number,omitempty"`
	SenderName                string            `json:"sender_name,omitempty"`
	ReceiverName              string            `json:"receiver_name,omitempty"`
	ReceiverBank              string            `json:"receiver_bank,omitempty"`
	ReceiverCountry           string            `json:"receiver_country,omitempty"`
	PaymentPurpose            string            `json:"payment_purpose,omitempty"`
	ValueDate                 string            `json:"value_date,omitempty"`
	ClosesAmount              *float64          `json:"closes_amount,omitempty"`
	SubmissionDate            string            `json:"submission_date,omitempty"`
	DeliveryDeadline          string            `json:"delivery_deadline,omitempty"`
	DaysDifference            *int              `json:"days_difference,omitempty"`
	DeliveryStatus            string            `json:"delivery_status,omitempty"`
	DeliveryNotice            string            `json:"delivery_notice,omitempty"`
	ApprovalStatus            string            `json:"approval_status"`
	CurrencyControlDecision   string            `json:"currency_control_decision,omitempty"`
	CurrencyControlComment    string            `json:"currency_control_comment,omitempty"`
	CurrencyControlReviewedAt *string           `json:"currency_control_reviewed_at,omitempty"`
	ComplianceDecision        string            `json:"compliance_decision,omitempty"`
	ComplianceComment         string            `json:"compliance_comment,omitempty"`
	ComplianceReviewedAt      *string           `json:"compliance_reviewed_at,omitempty"`
	RejectionReason           string            `json:"rejection_reason,omitempty"`
	CreatedAt                 string            `json:"created_at"`
	CurrencyControlReviewer   *domain.UserBrief `json:"currency_control_reviewer,omitempty"`
	ComplianceReviewer        *domain.UserBrief `json:"compliance_reviewer,omitempty"`
	Creator                   *domain.UserBrief `json:"creator,omitempty"`
}

type MyDocumentsResponse struct {
	Items    []MyDocumentItem `json:"items"`
	Total    int              `json:"total"`
	Page     int              `json:"page"`
	PageSize int              `json:"page_size"`
}
