package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"

	"gorm.io/gorm"
)

type myDocumentsRepo struct {
	db *gorm.DB
}

func NewMyDocumentsRepository(db *gorm.DB) ports.MyDocumentsRepository {
	return &myDocumentsRepo{db: db}
}

type rawMyDocRow struct {
	EntityType                string     `gorm:"column:entity_type"`
	EntityID                  int64      `gorm:"column:entity_id"`
	DocumentNumber            string     `gorm:"column:document_number"`
	DocumentPath              *string    `gorm:"column:document_path"`
	DocumentDate              time.Time  `gorm:"column:document_date"`
	Subject                   string     `gorm:"column:subject"`
	Amount                    float64    `gorm:"column:amount"`
	Currency                  string     `gorm:"column:currency"`
	CounterpartyName          string     `gorm:"column:counterparty_name"`
	BranchName                string     `gorm:"column:branch_name"`
	ContractID                *int64     `gorm:"column:contract_id"`
	ContractNumber            string     `gorm:"column:contract_number"`
	InvoiceID                 *int64     `gorm:"column:invoice_id"`
	InvoiceNumber             string     `gorm:"column:invoice_number"`
	SenderName                string     `gorm:"column:sender_name"`
	ReceiverName              string     `gorm:"column:receiver_name"`
	ReceiverBank              string     `gorm:"column:receiver_bank"`
	ReceiverCountry           string     `gorm:"column:receiver_country"`
	PaymentPurpose            string     `gorm:"column:payment_purpose"`
	ValueDate                 *time.Time `gorm:"column:value_date"`
	ClosesAmount              float64    `gorm:"column:closes_amount"`
	SubmissionDate            *time.Time `gorm:"column:submission_date"`
	DeliveryDeadline          *time.Time `gorm:"column:delivery_deadline"`
	DaysDifference            int        `gorm:"column:days_difference"`
	DeliveryStatus            string     `gorm:"column:delivery_status"`
	DeliveryNotice            string     `gorm:"column:delivery_notice"`
	ApprovalStatus            string     `gorm:"column:approval_status"`
	CurrencyControlDecision   string     `gorm:"column:currency_control_decision"`
	CurrencyControlComment    string     `gorm:"column:currency_control_comment"`
	CurrencyControlReviewedBy string     `gorm:"column:currency_control_reviewed_by"`
	CurrencyControlReviewedAt *time.Time `gorm:"column:currency_control_reviewed_at"`
	ComplianceDecision        string     `gorm:"column:compliance_decision"`
	ComplianceComment         string     `gorm:"column:compliance_comment"`
	ComplianceReviewedBy      string     `gorm:"column:compliance_reviewed_by"`
	ComplianceReviewedAt      *time.Time `gorm:"column:compliance_reviewed_at"`
	RejectionReason           string     `gorm:"column:rejection_reason"`
	CreatedAt                 time.Time  `gorm:"column:created_at"`
	CreatedBy                 string     `gorm:"column:created_by"`
}


func buildDeletedFilter(alias, status string) string {
	norm := strings.ToLower(strings.TrimSpace(status))
	switch norm {
	case "rejected":
		return fmt.Sprintf(
			"%s.deleted_at IS NOT NULL AND %s.approval_status IN ('rejected_currency_control','rejected_compliance')",
			alias, alias,
		)
	case "all", "":
		return fmt.Sprintf(
			"(%s.deleted_at IS NULL OR (%s.deleted_at IS NOT NULL AND %s.approval_status IN ('rejected_currency_control','rejected_compliance')))",
			alias, alias, alias,
		)
	default:
		return fmt.Sprintf("%s.deleted_at IS NULL", alias)
	}
}

func (r *myDocumentsRepo) GetMyDocuments(ctx context.Context, login string, filter dto.MyDocumentsFilter) ([]dto.MyDocumentItem, int, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	normEntity := strings.ToLower(strings.TrimSpace(filter.EntityType))
	normStatus := strings.ToLower(strings.TrimSpace(filter.Status))

	includeGTD := normEntity == "" || normEntity == "all" || normEntity == domain.EntityTypeGTD
	includePO := normEntity == "" || normEntity == "all" || normEntity == domain.EntityTypePaymentOrder

	if !includeGTD && !includePO {
		return []dto.MyDocumentItem{}, 0, nil
	}

	var subQueries []*gorm.DB

	if includeGTD {
		deletedCond := buildDeletedFilter("g", normStatus)
		q := r.db.WithContext(ctx).Table("gtd g").
			Select(BuildMyDocumentsSelect("g", "gtd", "g.gtd_number", "g.gtd_date", "''", "g.gtd_amount", "g.currency")).
			Joins("JOIN contracts c ON c.id = g.contract_id").
			Joins("LEFT JOIN invoices i ON i.id = g.invoice_id").
			Joins("LEFT JOIN branches b ON b.id = c.branch_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where(deletedCond)
		if filter.Scope == "mine" {
			q = q.Where("g.created_by = ?", login)
		}
		if normStatus != "" && normStatus != "all" && normStatus != "rejected" {
			q = q.Where("g.approval_status = ?", normStatus)
		}
		subQueries = append(subQueries, q)
	}

	if includePO {
		deletedCond := buildDeletedFilter("po", normStatus)
		q := r.db.WithContext(ctx).Table("payment_orders po").
			Select(BuildMyDocumentsSelect("po", "payment_order", "po.id::text", "po.value_date", "po.payment_purpose", "po.amount", "po.currency")).
			Joins("JOIN contracts c ON c.id = po.contract_id").
			Joins("LEFT JOIN invoices i ON i.id = po.invoice_id").
			Joins("LEFT JOIN branches b ON b.id = c.branch_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where(deletedCond)
		if filter.Scope == "mine" {
			q = q.Where("po.created_by = ?", login)
		}
		if normStatus != "" && normStatus != "all" && normStatus != "rejected" {
			q = q.Where("po.approval_status = ?", normStatus)
		}
		subQueries = append(subQueries, q)
	}

	if len(subQueries) == 0 {
		return []dto.MyDocumentItem{}, 0, nil
	}
	var combinedQuery *gorm.DB
	unionClauses := make([]string, len(subQueries))
	for i := range subQueries {
		unionClauses[i] = "(?)"
	}
	unionSQL := strings.Join(unionClauses, " UNION ALL ")
	args := make([]interface{}, len(subQueries))
	for i, sq := range subQueries {
		args[i] = sq
	}
	combinedQuery = r.db.WithContext(ctx).Table(fmt.Sprintf("(%s) AS my_docs_tbl", unionSQL), args...)

	var total int64
	if err := combinedQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []rawMyDocRow
	if err := combinedQuery.
		Order("created_at DESC").
		Limit(pageSize).Offset(offset).
		Scan(&rows).Error; err != nil {
		return nil, 0, err
	}
	items := make([]dto.MyDocumentItem, len(rows))
	for i, row := range rows {
		subject := row.Subject
		if row.EntityType == domain.EntityTypeGTD && strings.TrimSpace(subject) == "" {
			subject = domain.DefaultHSCode
		}
		var valDateStr string
		if row.ValueDate != nil && !row.ValueDate.IsZero() {
			valDateStr = row.ValueDate.Format("02.01.2006")
		}
		var submissionDateStr string
		if row.SubmissionDate != nil && !row.SubmissionDate.IsZero() {
			submissionDateStr = row.SubmissionDate.Format("02.01.2006")
		}
		var deliveryDeadlineStr string
		if row.DeliveryDeadline != nil && !row.DeliveryDeadline.IsZero() {
			deliveryDeadlineStr = row.DeliveryDeadline.Format("02.01.2006")
		}
		var closesAmountPtr *float64
		var daysDiffPtr *int
		if row.EntityType == domain.EntityTypeGTD {
			closesAmountPtr = &row.ClosesAmount
			daysDiffPtr = &row.DaysDifference
		}
		var contractIDPtr *int64
		if row.ContractID != nil && *row.ContractID > 0 {
			contractIDPtr = row.ContractID
		}
		var invoiceIDPtr *int64
		if row.InvoiceID != nil && *row.InvoiceID > 0 {
			invoiceIDPtr = row.InvoiceID
		}
		items[i] = dto.MyDocumentItem{
			EntityType:                row.EntityType,
			EntityID:                  row.EntityID,
			DocumentNumber:            row.DocumentNumber,
			DocumentPath:              row.DocumentPath,
			DocumentDate:              row.DocumentDate.Format("02.01.2006"),
			Subject:                   subject,
			Amount:                    row.Amount,
			Currency:                  row.Currency,
			CounterpartyName:          row.CounterpartyName,
			BranchName:                row.BranchName,
			ContractID:                contractIDPtr,
			ContractNumber:            row.ContractNumber,
			InvoiceID:                 invoiceIDPtr,
			InvoiceNumber:             row.InvoiceNumber,
			SenderName:                row.SenderName,
			ReceiverName:              row.ReceiverName,
			ReceiverBank:              row.ReceiverBank,
			ReceiverCountry:           row.ReceiverCountry,
			PaymentPurpose:            row.PaymentPurpose,
			ValueDate:                 valDateStr,
			ClosesAmount:              closesAmountPtr,
			SubmissionDate:            submissionDateStr,
			DeliveryDeadline:          deliveryDeadlineStr,
			DaysDifference:            daysDiffPtr,
			DeliveryStatus:            row.DeliveryStatus,
			DeliveryNotice:            row.DeliveryNotice,
			ApprovalStatus:            row.ApprovalStatus,
			CurrencyControlDecision:   row.CurrencyControlDecision,
			CurrencyControlComment:    row.CurrencyControlComment,
			ComplianceDecision:        row.ComplianceDecision,
			ComplianceComment:         row.ComplianceComment,
			RejectionReason:           row.RejectionReason,
			CreatedAt:                 row.CreatedAt.Format(time.RFC3339),
			CurrencyControlReviewedAt: formatTimePtr(row.CurrencyControlReviewedAt),
			ComplianceReviewedAt:      formatTimePtr(row.ComplianceReviewedAt),
		}
	}
	_ = enrichMyDocItems(ctx, r.db, items, rows)

	return items, int(total), nil
}

func enrichMyDocItems(ctx context.Context, db *gorm.DB, items []dto.MyDocumentItem, rows []rawMyDocRow) error {
	logins := make([]string, 0, len(rows)*3)
	for _, row := range rows {
		logins = append(logins, row.CurrencyControlReviewedBy, row.ComplianceReviewedBy, row.CreatedBy)
	}
	userMap := fetchUserBriefs(ctx, db, logins)

	for i, row := range rows {
		if row.CurrencyControlReviewedBy != "" {
			b := userMap[row.CurrencyControlReviewedBy]
			items[i].CurrencyControlReviewer = &b
		}
		if row.ComplianceReviewedBy != "" {
			b := userMap[row.ComplianceReviewedBy]
			items[i].ComplianceReviewer = &b
		}
		if row.CreatedBy != "" {
			b := userMap[row.CreatedBy]
			items[i].Creator = &b
		}
	}
	return nil
}
