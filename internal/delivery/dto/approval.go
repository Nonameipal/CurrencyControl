package dto

type CurrencyControlDecisionRequest struct {
	Decision string `json:"decision" enums:"accepted,revision,rejected" example:"accepted"` // "accepted" (Принято), "revision" (На доработку), "rejected" (Отклонено)
	Comment  string `json:"comment" example:"Предмет контракта проверен"`
}

type ComplianceDecisionRequest struct {
	Decision string `json:"decision" enums:"approve,reject" example:"approve"` // "approve" (Одобрено), "reject" (Отказ)
	Comment  string `json:"comment" example:"Причина отказа обязательна при выборе reject"`
}

type PendingApprovalsFilter struct {
	Stage      string `json:"stage"`       // "currency_control", "compliance", "revision"
	EntityType string `json:"entity_type"` // "contract", "invoice", "gtd", "additional_agreement"
	BranchID   *int   `json:"branch_id"`
	Page       int    `json:"page"`
	PageSize   int    `json:"page_size"`
}

type ApprovalItemResponse struct {
	EntityType                 string   `json:"entity_type"`
	EntityID                   int64    `json:"entity_id"`
	BranchID                   *int     `json:"branch_id,omitempty"`
	BranchName                 string   `json:"branch_name,omitempty"`
	DocumentNumber             string   `json:"document_number"`
	DocumentDate               string   `json:"document_date"`
	Subject                    string   `json:"subject,omitempty"`
	Amount                     float64  `json:"amount"`
	Currency                   string   `json:"currency"`
	CounterpartyName           string   `json:"counterparty_name,omitempty"`
	ApprovalStatus             string   `json:"approval_status"`
	CurrencyControlDecision   string   `json:"currency_control_decision,omitempty"`
	CurrencyControlComment    string   `json:"currency_control_comment,omitempty"`
	CurrencyControlReviewedBy string   `json:"currency_control_reviewed_by,omitempty"`
	CurrencyControlReviewedAt *string  `json:"currency_control_reviewed_at,omitempty"`
	ComplianceDecision        string   `json:"compliance_decision,omitempty"`
	ComplianceComment         string   `json:"compliance_comment,omitempty"`
	ComplianceReviewedBy      string   `json:"compliance_reviewed_by,omitempty"`
	ComplianceReviewedAt      *string  `json:"compliance_reviewed_at,omitempty"`
	RejectionReason           string   `json:"rejection_reason,omitempty"`
	CreatedBy                  string   `json:"created_by"`
	CreatedAt                  string   `json:"created_at"`
}

type PendingApprovalsResponse struct {
	Items    []ApprovalItemResponse `json:"items"`
	Total    int                    `json:"total"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
}
