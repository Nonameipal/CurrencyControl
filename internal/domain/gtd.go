package domain

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

const DefaultHSCode = "Нет кода"

type GTD struct {
	ID                        int64          `gorm:"primaryKey" db:"id" json:"id"`
	ContractID                int64          `gorm:"not null;index" db:"contract_id" json:"contract_id"`
	InvoiceID                 int64          `gorm:"not null;index" db:"invoice_id" json:"invoice_id"`
	GTDNumber                 string         `gorm:"type:varchar(255);not null;column:gtd_number" db:"gtd_number" json:"gtd_number"`
	GTDDate                   time.Time      `gorm:"type:date;not null;column:gtd_date" db:"gtd_date" json:"gtd_date"`
	GTDAmount                 float64        `gorm:"type:decimal(18,2);not null;column:gtd_amount" db:"gtd_amount" json:"gtd_amount"`
	Currency                  string         `gorm:"type:varchar(10);not null;default:'USD';column:currency" db:"currency" json:"currency"`
	ClosesAmount              float64        `gorm:"type:decimal(18,2);not null;default:0" db:"closes_amount" json:"closes_amount"`
	InvoiceNumber             string         `gorm:"-" db:"invoice_number" json:"invoice_number,omitempty"`
	DocumentPath              *string        `db:"document_path" json:"document_path,omitempty"`
	SubmissionDate            *time.Time     `gorm:"type:date;column:submission_date" db:"submission_date" json:"submission_date,omitempty"`
	DeliveryDeadline          *time.Time     `gorm:"type:date;column:delivery_deadline" db:"delivery_deadline" json:"delivery_deadline,omitempty"`
	DaysDifference            int            `gorm:"column:days_difference;default:0" db:"days_difference" json:"days_difference"`
	DeliveryStatus            string         `gorm:"type:varchar(50);default:''" db:"delivery_status" json:"delivery_status,omitempty"`
	DeliveryNotice            string         `gorm:"type:text;default:''" db:"delivery_notice" json:"delivery_notice,omitempty"`
	CreatedBy                 string         `gorm:"type:varchar(255);not null" db:"created_by" json:"created_by"`
	ApprovalStatus            string         `gorm:"type:varchar(50);not null;default:'pending_currency_control';index" db:"approval_status" json:"approval_status"`
	CurrencyControlDecision   string         `gorm:"type:varchar(50);default:''" db:"currency_control_decision" json:"currency_control_decision,omitempty"`
	CurrencyControlComment    string         `gorm:"type:text;default:''" db:"currency_control_comment" json:"currency_control_comment,omitempty"`
	CurrencyControlReviewedBy string         `gorm:"type:varchar(255);default:''" db:"currency_control_reviewed_by" json:"currency_control_reviewed_by,omitempty"`
	CurrencyControlReviewedAt *time.Time     `gorm:"type:timestamptz" db:"currency_control_reviewed_at" json:"currency_control_reviewed_at,omitempty"`
	ComplianceDecision        string         `gorm:"type:varchar(50);default:''" db:"compliance_decision" json:"compliance_decision,omitempty"`
	ComplianceComment         string         `gorm:"type:text;default:''" db:"compliance_comment" json:"compliance_comment,omitempty"`
	ComplianceReviewedBy      string         `gorm:"type:varchar(255);default:''" db:"compliance_reviewed_by" json:"compliance_reviewed_by,omitempty"`
	ComplianceReviewedAt      *time.Time     `gorm:"type:timestamptz" db:"compliance_reviewed_at" json:"compliance_reviewed_at,omitempty"`
	RejectionReason           string         `gorm:"type:text;default:''" db:"rejection_reason" json:"rejection_reason,omitempty"`
	CreatedAt                 time.Time      `gorm:"not null;default:now()" db:"created_at" json:"created_at"`
	UpdatedAt                 time.Time      `gorm:"not null;default:now()" db:"updated_at" json:"updated_at"`
	DeletedAt                 gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
	UpdatedBy                 string         `gorm:"type:varchar(255);default:''" db:"updated_by" json:"updated_by,omitempty"`
	DeletedBy                 *string        `gorm:"type:varchar(255)" db:"deleted_by" json:"deleted_by,omitempty"`
	Creator                   *UserBrief     `gorm:"-" json:"creator,omitempty"`
	Updater                   *UserBrief     `gorm:"-" json:"updater,omitempty"`
	Deleter                   *UserBrief     `gorm:"-" json:"deleter,omitempty"`
	CurrencyControlReviewer   *UserBrief     `gorm:"-" json:"currency_control_reviewer,omitempty"`
	ComplianceReviewer        *UserBrief     `gorm:"-" json:"compliance_reviewer,omitempty"`
}

func (GTD) TableName() string {
	return "gtd"
}

func FormatRussianDays(n int) string {
	absN := n
	if absN < 0 {
		absN = -absN
	}
	mod100 := absN % 100
	mod10 := absN % 10
	if mod100 >= 11 && mod100 <= 19 {
		return fmt.Sprintf("%d дней", absN)
	}
	switch mod10 {
	case 1:
		return fmt.Sprintf("%d день", absN)
	case 2, 3, 4:
		return fmt.Sprintf("%d дня", absN)
	default:
		return fmt.Sprintf("%d дней", absN)
	}
}

func CalculateDeliveryComparison(actualDate time.Time, deadline *time.Time) (diffDays int, status string, notice string) {
	docName := "ГТД"
	verbPrefix := "предоставлена"
	termNameGenitive := "срока поставки"
	termNameAccusative := "срок поставки"

	if deadline == nil || deadline.IsZero() {
		return 0, "unknown", fmt.Sprintf("Плановый регламентированный срок по контракту не установлен. %s %s в систему %s.", docName, verbPrefix, actualDate.Format("02.01.2006"))
	}

	act := time.Date(actualDate.Year(), actualDate.Month(), actualDate.Day(), 0, 0, 0, 0, time.UTC)
	dl := time.Date(deadline.Year(), deadline.Month(), deadline.Day(), 0, 0, 0, 0, time.UTC)

	diffDays = int(act.Sub(dl).Hours() / 24)
	planStr := dl.Format("02.01.2006")
	factStr := act.Format("02.01.2006")

	switch {
	case diffDays < 0:
		status = "early"
		daysText := FormatRussianDays(-diffDays)
		notice = fmt.Sprintf("%s %s на %s раньше установленного %s (план: %s, факт: %s)",
			docName, verbPrefix, daysText, termNameGenitive, planStr, factStr)
	case diffDays == 0:
		status = "on_time"
		notice = fmt.Sprintf("%s %s точно в установленный %s (%s)",
			docName, verbPrefix, termNameAccusative, planStr)
	default:
		status = "overdue"
		daysText := FormatRussianDays(diffDays)
		notice = fmt.Sprintf("Внимание! %s %s на %s позже установленного %s (план: %s, факт: %s). Просрочка: %s",
			docName, verbPrefix, daysText, termNameGenitive, planStr, factStr, daysText)
	}

	return diffDays, status, notice
}
