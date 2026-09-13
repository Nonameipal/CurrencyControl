package domain

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

const (
	DocumentTypeGTD = "gtd" 
	DocumentTypeAct = "act"
)

type GTD struct {
	ID                   int64          `gorm:"primaryKey" db:"id" json:"id"`
	ContractID            int64          `gorm:"not null;index" db:"contract_id" json:"contract_id"`
	AdditionalAgreementID *int64         `gorm:"index" db:"additional_agreement_id" json:"additional_agreement_id,omitempty"`
	InvoiceID             int64          `gorm:"index" db:"invoice_id" json:"invoice_id"`
	DocumentType         string         `gorm:"column:document_type;type:varchar(50);default:'gtd'" db:"document_type" json:"document_type"` // gtd или act
	GTDNumber            string         `gorm:"not null;column:gtd_number" db:"gtd_number" json:"gtd_number"`
	GTDCurrency          *string        `gorm:"not null;column:gtd_currency" db:"gtd_currency" json:"gtd_currency"`
	GTDDate              *time.Time     `gorm:"type:date;column:gtd_date" db:"gtd_date" json:"gtd_date"`
	GTDAmount            float64        `gorm:"type:decimal(18,2);not null;column:gtd_amount" db:"gtd_amount" json:"gtd_amount"`
	ClosesAmount         float64        `gorm:"type:decimal(18,2);not null;default:0" db:"closes_amount" json:"closes_amount"`
	HSCode               string         `gorm:"type:varchar(50);default:''" db:"hs_code" json:"hs_code"`
	DestinationCountry   string         `gorm:"type:varchar(255);default:''" db:"destination_country" json:"destination_country"`
	InvoiceNumber        string         `gorm:"-" db:"invoice_number" json:"invoice_number,omitempty"`
	DocumentPath         *string        `db:"document_path" json:"document_path,omitempty"`
	SubmissionDate       *time.Time     `gorm:"type:date;column:submission_date" db:"submission_date" json:"submission_date,omitempty"`
	DeliveryDeadline     *time.Time     `gorm:"type:date;column:delivery_deadline" db:"delivery_deadline" json:"delivery_deadline,omitempty"`
	DaysDifference       int            `gorm:"column:days_difference;default:0" db:"days_difference" json:"days_difference"`
	DeliveryStatus       string         `gorm:"type:varchar(50);default:''" db:"delivery_status" json:"delivery_status,omitempty"` // early, on_time, overdue, unknown
	DeliveryNotice       string         `gorm:"type:text;default:''" db:"delivery_notice" json:"delivery_notice,omitempty"`
	CreatedBy            string         `gorm:"type:varchar(255);default:''" db:"created_by" json:"created_by"`
	CreatedAt            time.Time      `gorm:"not null;default:now()" db:"created_at" json:"created_at"`
	UpdatedAt            time.Time      `gorm:"not null;default:now()" db:"updated_at" json:"updated_at"`
	DeletedAt            gorm.DeletedAt `gorm:"index" json:"deleted_at,omitempty"`
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

func CalculateDeliveryComparison(docType string, actualDate time.Time, deadline *time.Time) (diffDays int, status string, notice string) {
	docName := "ГТД"
	verbPrefix := "предоставлена"
	termNameGenitive := "срока поставки"
	termNameAccusative := "срок поставки"
	if docType == DocumentTypeAct {
		docName = "Акт выполненных работ"
		verbPrefix = "предоставлен"
		termNameGenitive = "срока предоставления услуг"
		termNameAccusative = "срок предоставления услуг"
	}

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
