package domain

import "time"

const (
	ApprovalStatusPendingCurrencyControl  = "pending_currency_control"
	ApprovalStatusPendingCompliance       = "pending_compliance"
	ApprovalStatusRevisionRequired        = "revision_required"
	ApprovalStatusRejectedCurrencyControl = "rejected_currency_control"
	ApprovalStatusRejectedCompliance      = "rejected_compliance"
	ApprovalStatusApproved                = "approved"

	DecisionAccept   = "accept"
	DecisionRevision = "revision"
	DecisionReject   = "reject"
	DecisionApprove  = "approve"

	EntityTypeContract            = "contract"
	EntityTypeInvoice             = "invoice"
	EntityTypeGTD                 = "gtd"
	EntityTypeAdditionalAgreement = "additional_agreement"
)

type DocumentApprovalFields struct {
	ApprovalStatus            string     `gorm:"type:varchar(50);not null;default:'pending_currency_control'" db:"approval_status" json:"approval_status"`
	CurrencyControlDecision   string     `gorm:"type:varchar(50);default:''" db:"currency_control_decision" json:"currency_control_decision,omitempty"`
	CurrencyControlComment    string     `gorm:"type:text;default:''" db:"currency_control_comment" json:"currency_control_comment,omitempty"`
	CurrencyControlReviewedBy string     `gorm:"type:varchar(255);default:''" db:"currency_control_reviewed_by" json:"currency_control_reviewed_by,omitempty"`
	CurrencyControlReviewedAt *time.Time `gorm:"type:timestamptz" db:"currency_control_reviewed_at" json:"currency_control_reviewed_at,omitempty"`
	ComplianceDecision        string     `gorm:"type:varchar(50);default:''" db:"compliance_decision" json:"compliance_decision,omitempty"`
	ComplianceComment         string     `gorm:"type:text;default:''" db:"compliance_comment" json:"compliance_comment,omitempty"`
	ComplianceReviewedBy      string     `gorm:"type:varchar(255);default:''" db:"compliance_reviewed_by" json:"compliance_reviewed_by,omitempty"`
	ComplianceReviewedAt      *time.Time `gorm:"type:timestamptz" db:"compliance_reviewed_at" json:"compliance_reviewed_at,omitempty"`
	RejectionReason           string     `gorm:"type:text;default:''" db:"rejection_reason" json:"rejection_reason,omitempty"`
}
