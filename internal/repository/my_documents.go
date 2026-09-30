package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"CurrencyControl/internal/delivery/dto"
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
	EntityType                string
	EntityID                  int64
	DocumentNumber            string
	DocumentPath              *string
	DocumentDate              time.Time
	Subject                   string
	Amount                    float64
	Currency                  string
	CounterpartyName          string
	BranchName                string
	ApprovalStatus            string
	CurrencyControlDecision   string
	CurrencyControlComment    string
	CurrencyControlReviewedBy string
	CurrencyControlReviewedAt *time.Time
	ComplianceDecision        string
	ComplianceComment         string
	ComplianceReviewedBy      string
	ComplianceReviewedAt      *time.Time
	RejectionReason           string
	CreatedAt                 time.Time
	CreatedBy                 string
}


func buildDeletedFilter(alias, status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
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

	includeContract := normEntity == "" || normEntity == "all" || normEntity == "contract"
	includeInvoice := normEntity == "" || normEntity == "all" || normEntity == "invoice"
	includeGTD := normEntity == "" || normEntity == "all" || normEntity == "gtd"
	includeAA := normEntity == "" || normEntity == "all" || normEntity == "additional_agreement"

	var subQueries []*gorm.DB



	if includeContract {
		deletedCond := buildDeletedFilter("c", normStatus)
		q := r.db.WithContext(ctx).Table("contracts c").
			Select(BuildMyDocumentsSelect("c", "contract", "c.contract_number", "c.contract_date", "c.subject", "c.total_amount", "c.contract_currency")).
			Joins("LEFT JOIN branches b ON b.id = c.branch_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where(deletedCond)
		if filter.Scope == "mine" {
			q = q.Where("c.created_by = ?", login)
		}
		if normStatus != "" && normStatus != "all" && normStatus != "rejected" {
			q = q.Where("c.approval_status = ?", normStatus)
		}
		subQueries = append(subQueries, q)
	}

	if includeInvoice {
		deletedCond := buildDeletedFilter("i", normStatus)
		q := r.db.WithContext(ctx).Table("invoices i").
			Select(BuildMyDocumentsSelect("i", "invoice", "i.invoice_number", "i.invoice_date", "i.hs_code", "i.amount", "i.currency")).
			Joins("JOIN contracts c ON c.id = i.contract_id").
			Joins("LEFT JOIN branches b ON b.id = c.branch_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where(deletedCond)
		if filter.Scope == "mine" {
			q = q.Where("i.created_by = ?", login)
		}
		if normStatus != "" && normStatus != "all" && normStatus != "rejected" {
			q = q.Where("i.approval_status = ?", normStatus)
		}
		subQueries = append(subQueries, q)
	}

	if includeGTD {
		deletedCond := buildDeletedFilter("g", normStatus)
		q := r.db.WithContext(ctx).Table("gtd g").
			Select(BuildMyDocumentsSelect("g", "gtd", "g.gtd_number", "COALESCE(g.gtd_date, g.created_at)", "g.hs_code", "g.gtd_amount", "COALESCE(g.gtd_currency, '')")).
			Joins("JOIN contracts c ON c.id = g.contract_id").
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

	if includeAA {
		deletedCond := buildDeletedFilter("aa", normStatus)
		q := r.db.WithContext(ctx).Table("additional_agreements aa").
			Select(BuildMyDocumentsSelect("aa", "additional_agreement", "COALESCE(aa.agreement_number, '')", "COALESCE(aa.agreement_date, aa.created_at)", "aa.subject", "COALESCE(aa.foreign_amount, 0)", "COALESCE(aa.currency, '')")).
			Joins("JOIN contracts c ON c.id = aa.contract_id").
			Joins("LEFT JOIN branches b ON b.id = c.branch_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where(deletedCond)
		if filter.Scope == "mine" {
			q = q.Where("aa.created_by = ?", login)
		}
		if normStatus != "" && normStatus != "all" && normStatus != "rejected" {
			q = q.Where("aa.approval_status = ?", normStatus)
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
		items[i] = dto.MyDocumentItem{
			EntityType:              row.EntityType,
			EntityID:                row.EntityID,
			DocumentNumber:          row.DocumentNumber,
			DocumentPath:            row.DocumentPath,
			DocumentDate:            row.DocumentDate.Format("02.01.2006"),
			Subject:                 row.Subject,
			Amount:                  row.Amount,
			Currency:                row.Currency,
			CounterpartyName:        row.CounterpartyName,
			BranchName:              row.BranchName,
			ApprovalStatus:          row.ApprovalStatus,
			CurrencyControlDecision: row.CurrencyControlDecision,
			CurrencyControlComment:  row.CurrencyControlComment,
			ComplianceDecision:      row.ComplianceDecision,
			ComplianceComment:       row.ComplianceComment,
			RejectionReason:         row.RejectionReason,
			CreatedAt:               row.CreatedAt.Format(time.RFC3339),
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
