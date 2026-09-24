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
			Select(`
				'contract' AS entity_type, c.id AS entity_id,
				c.contract_number AS document_number, c.document_path, c.contract_date AS document_date,
				COALESCE(c.subject,'') AS subject, c.total_amount AS amount, c.contract_currency AS currency,
				COALESCE(cp.llc,'') AS counterparty_name, COALESCE(b.name,'') AS branch_name,
				COALESCE(c.approval_status,'pending_currency_control') AS approval_status,
				COALESCE(c.currency_control_decision,'') AS currency_control_decision,
				COALESCE(c.currency_control_comment,'') AS currency_control_comment,
				COALESCE(c.currency_control_reviewed_by,'') AS currency_control_reviewed_by,
				c.currency_control_reviewed_at,
				COALESCE(c.compliance_decision,'') AS compliance_decision,
				COALESCE(c.compliance_comment,'') AS compliance_comment,
				COALESCE(c.compliance_reviewed_by,'') AS compliance_reviewed_by,
				c.compliance_reviewed_at,
				COALESCE(c.rejection_reason,'') AS rejection_reason, c.created_at,
				COALESCE(c.created_by, '') AS created_by`).
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
			Select(`
				'invoice' AS entity_type, i.id AS entity_id,
				i.invoice_number AS document_number, i.document_path, i.invoice_date AS document_date,
				COALESCE(i.hs_code,'') AS subject, i.amount AS amount, i.currency AS currency,
				COALESCE(cp.llc,'') AS counterparty_name, COALESCE(b.name,'') AS branch_name,
				COALESCE(i.approval_status,'pending_currency_control') AS approval_status,
				COALESCE(i.currency_control_decision,'') AS currency_control_decision,
				COALESCE(i.currency_control_comment,'') AS currency_control_comment,
				COALESCE(i.currency_control_reviewed_by,'') AS currency_control_reviewed_by,
				i.currency_control_reviewed_at,
				COALESCE(i.compliance_decision,'') AS compliance_decision,
				COALESCE(i.compliance_comment,'') AS compliance_comment,
				COALESCE(i.compliance_reviewed_by,'') AS compliance_reviewed_by,
				i.compliance_reviewed_at,
				COALESCE(i.rejection_reason,'') AS rejection_reason, i.created_at,
				COALESCE(i.created_by, '') AS created_by
			`).
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
			Select(`
				'gtd' AS entity_type, g.id AS entity_id,
				g.gtd_number AS document_number, g.document_path, COALESCE(g.gtd_date, g.created_at) AS document_date,
				COALESCE(g.hs_code,'') AS subject, g.gtd_amount AS amount, COALESCE(g.gtd_currency,'') AS currency,
				COALESCE(cp.llc,'') AS counterparty_name, COALESCE(b.name,'') AS branch_name,
				COALESCE(g.approval_status,'pending_currency_control') AS approval_status,
				COALESCE(g.currency_control_decision,'') AS currency_control_decision,
				COALESCE(g.currency_control_comment,'') AS currency_control_comment,
				COALESCE(g.currency_control_reviewed_by,'') AS currency_control_reviewed_by,
				g.currency_control_reviewed_at,
				COALESCE(g.compliance_decision,'') AS compliance_decision,
				COALESCE(g.compliance_comment,'') AS compliance_comment,
				COALESCE(g.compliance_reviewed_by,'') AS compliance_reviewed_by,
				g.compliance_reviewed_at,
				COALESCE(g.rejection_reason,'') AS rejection_reason, g.created_at,
				COALESCE(g.created_by, '') AS created_by
			`).
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
			Select(`
				'additional_agreement' AS entity_type, aa.id AS entity_id,
				COALESCE(aa.agreement_number,'') AS document_number, aa.document_path, COALESCE(aa.agreement_date, aa.created_at) AS document_date,
				COALESCE(aa.subject,'') AS subject, COALESCE(aa.foreign_amount,0) AS amount, COALESCE(aa.currency,'') AS currency,
				COALESCE(cp.llc,'') AS counterparty_name, COALESCE(b.name,'') AS branch_name,
				COALESCE(aa.approval_status,'pending_currency_control') AS approval_status,
				COALESCE(aa.currency_control_decision,'') AS currency_control_decision,
				COALESCE(aa.currency_control_comment,'') AS currency_control_comment,
				COALESCE(aa.currency_control_reviewed_by,'') AS currency_control_reviewed_by,
				aa.currency_control_reviewed_at,
				COALESCE(aa.compliance_decision,'') AS compliance_decision,
				COALESCE(aa.compliance_comment,'') AS compliance_comment,
				COALESCE(aa.compliance_reviewed_by,'') AS compliance_reviewed_by,
				aa.compliance_reviewed_at,
				COALESCE(aa.rejection_reason,'') AS rejection_reason, aa.created_at,
				COALESCE(aa.created_by, '') AS created_by
			`).
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
	if len(subQueries) == 1 {
		combinedQuery = subQueries[0]
	} else {
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
	}

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
	loginsMap := make(map[string]bool)
	for _, row := range rows {
		if row.CurrencyControlReviewedBy != "" {
			loginsMap[row.CurrencyControlReviewedBy] = true
		}
		if row.ComplianceReviewedBy != "" {
			loginsMap[row.ComplianceReviewedBy] = true
		}
		if row.CreatedBy != "" {
			loginsMap[row.CreatedBy] = true
		}
	}
	if len(loginsMap) == 0 {
		return nil
	}

	logins := make([]string, 0, len(loginsMap))
	for l := range loginsMap {
		logins = append(logins, l)
	}

	var users []domain.User
	if err := db.WithContext(ctx).Table("users").
		Select("login, first_name, last_name, email").
		Where("login IN ?", logins).
		Find(&users).Error; err != nil {
		return err
	}

	userMap := make(map[string]domain.UserBrief, len(users))
	for _, u := range users {
		userMap[u.Login] = domain.UserBrief{
			Login:     u.Login,
			FirstName: u.FirstName,
			LastName:  u.LastName,
			Email:     u.Email,
		}
	}

	for i, row := range rows {
		if row.CurrencyControlReviewedBy != "" {
			if b, ok := userMap[row.CurrencyControlReviewedBy]; ok {
				items[i].CurrencyControlReviewer = &b
			} else {
				items[i].CurrencyControlReviewer = &domain.UserBrief{Login: row.CurrencyControlReviewedBy}
			}
		}
		if row.ComplianceReviewedBy != "" {
			if b, ok := userMap[row.ComplianceReviewedBy]; ok {
				items[i].ComplianceReviewer = &b
			} else {
				items[i].ComplianceReviewer = &domain.UserBrief{Login: row.ComplianceReviewedBy}
			}
		}
		if row.CreatedBy != "" {
			if b, ok := userMap[row.CreatedBy]; ok {
				items[i].Creator = &b
			} else {
				items[i].Creator = &domain.UserBrief{Login: row.CreatedBy}
			}
		}
	}
	return nil
}
