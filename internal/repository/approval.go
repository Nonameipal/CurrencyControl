package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"

	"github.com/jackc/pgx/v5/pgxpool"
)

type approvalRepo struct {
	db *pgxpool.Pool
}

func NewApprovalRepository(db *pgxpool.Pool) ports.ApprovalRepository {
	return &approvalRepo{db: db}
}

func getTableName(entityType string) (string, error) {
	norm := strings.ToLower(strings.TrimSpace(entityType))
	switch norm {
	case "contract", "contracts":
		return "contracts", nil
	case "invoice", "invoices":
		return "invoices", nil
	case "gtd":
		return "gtd", nil
	case "additional_agreement", "additional-agreement", "additional_agreements", "additional-agreements":
		return "additional_agreements", nil
	default:
		return "", fmt.Errorf("неизвестный тип документа: %s", entityType)
	}
}

func formatTimePtr(t *time.Time) *string {
	if t == nil || t.IsZero() {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}

func (r *approvalRepo) SetCurrencyControlDecision(ctx context.Context, entityType string, id int64, decision, comment, reviewer string) (*dto.ApprovalItemResponse, error) {
	tbl, err := getTableName(entityType)
	if err != nil {
		return nil, err
	}

	var currentStatus string
	err = r.db.QueryRow(ctx, fmt.Sprintf(`SELECT approval_status FROM %s WHERE id = $1 AND deleted_at IS NULL`, tbl), id).Scan(&currentStatus)
	if err != nil {
		return nil, fmt.Errorf("документ не найден: %w", err)
	}

	if currentStatus != domain.ApprovalStatusPendingCurrencyControl && currentStatus != domain.ApprovalStatusRevisionRequired {
		return nil, fmt.Errorf("документ находится в статусе '%s' и не ожидает проверки валютным контролем", currentStatus)
	}

	var newStatus string
	normDecision := strings.ToLower(strings.TrimSpace(decision))
	switch normDecision {
	case "accepted":
		newStatus = domain.ApprovalStatusPendingCompliance
	case "revision":
		newStatus = domain.ApprovalStatusRevisionRequired
	case "rejected":
		newStatus = domain.ApprovalStatusRejectedCurrencyControl
	default:
		return nil, fmt.Errorf("недопустимое решение валютного контроля: %s (допустимы: accepted, revision, rejected)", decision)
	}

	rejectionReason := ""
	if newStatus == domain.ApprovalStatusRevisionRequired || newStatus == domain.ApprovalStatusRejectedCurrencyControl {
		rejectionReason = comment
	}

	query := fmt.Sprintf(`
		UPDATE %s
		SET approval_status = $1,
		    currency_control_decision = $2,
		    currency_control_comment = $3,
		    currency_control_reviewed_by = $4,
		    currency_control_reviewed_at = NOW(),
		    rejection_reason = $5,
		    updated_at = NOW()
		WHERE id = $6 AND deleted_at IS NULL`, tbl)

	_, err = r.db.Exec(ctx, query, newStatus, normDecision, comment, reviewer, rejectionReason, id)
	if err != nil {
		return nil, fmt.Errorf("ошибка обновления статуса документа: %w", err)
	}

	return r.GetApprovalDetail(ctx, entityType, id)
}

func (r *approvalRepo) SetComplianceDecision(ctx context.Context, entityType string, id int64, decision, comment, reviewer string) (*dto.ApprovalItemResponse, error) {
	tbl, err := getTableName(entityType)
	if err != nil {
		return nil, err
	}

	var currentStatus string
	err = r.db.QueryRow(ctx, fmt.Sprintf(`SELECT approval_status FROM %s WHERE id = $1 AND deleted_at IS NULL`, tbl), id).Scan(&currentStatus)
	if err != nil {
		return nil, fmt.Errorf("документ не найден: %w", err)
	}

	if currentStatus != domain.ApprovalStatusPendingCompliance {
		return nil, fmt.Errorf("документ находится в статусе '%s'. Комплаенс-контроль может рассматривать только документы, прошедшие проверку валютным контролем (pending_compliance)", currentStatus)
	}

	var newStatus string
	normDecision := strings.ToLower(strings.TrimSpace(decision))
	switch normDecision {
	case "approve":
		newStatus = domain.ApprovalStatusApproved
		normDecision = "approve"
	case "reject":
		newStatus = domain.ApprovalStatusRejectedCompliance
		normDecision = "reject"
	default:
		return nil, fmt.Errorf("недопустимое решение комплаенс-контроля: %s (допустимы: approve, reject)", decision)
	}

	rejectionReason := ""
	if newStatus == domain.ApprovalStatusRejectedCompliance {
		rejectionReason = comment
	}

	query := fmt.Sprintf(`
		UPDATE %s
		SET approval_status = $1,
		    compliance_decision = $2,
		    compliance_comment = $3,
		    compliance_reviewed_by = $4,
		    compliance_reviewed_at = NOW(),
		    rejection_reason = $5,
		    updated_at = NOW()
		WHERE id = $6 AND deleted_at IS NULL`, tbl)

	_, err = r.db.Exec(ctx, query, newStatus, normDecision, comment, reviewer, rejectionReason, id)
	if err != nil {
		return nil, fmt.Errorf("ошибка обновления статуса документа: %w", err)
	}

	if newStatus == domain.ApprovalStatusApproved {
		if tbl == "contracts" || tbl == "additional_agreements" {
			_, _ = r.db.Exec(ctx, fmt.Sprintf(`UPDATE %s SET status = 'active' WHERE id = $1 AND (status IS NULL OR status != 'archived')`, tbl), id)
		}
	}

	return r.GetApprovalDetail(ctx, entityType, id)
}

func (r *approvalRepo) ResetToPendingCurrencyControl(ctx context.Context, entityType string, id int64) error {
	tbl, err := getTableName(entityType)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`
		UPDATE %s
		SET approval_status = $1,
		    rejection_reason = '',
		    updated_at = NOW()
		WHERE id = $2 AND deleted_at IS NULL`, tbl)

	_, err = r.db.Exec(ctx, query, domain.ApprovalStatusPendingCurrencyControl, id)
	return err
}

func (r *approvalRepo) GetApprovalDetail(ctx context.Context, entityType string, id int64) (*dto.ApprovalItemResponse, error) {
	norm := strings.ToLower(strings.TrimSpace(entityType))
	switch norm {
	case "contract", "contracts":
		return r.getContractApproval(ctx, id)
	case "invoice", "invoices":
		return r.getInvoiceApproval(ctx, id)
	case "gtd":
		return r.getGTDApproval(ctx, id)
	case "additional_agreement", "additional-agreement", "additional_agreements", "additional-agreements":
		return r.getAAApproval(ctx, id)
	default:
		return nil, fmt.Errorf("неизвестный тип документа: %s", entityType)
	}
}

func (r *approvalRepo) getContractApproval(ctx context.Context, id int64) (*dto.ApprovalItemResponse, error) {
	query := `
		SELECT 
			c.id, c.branch_id, COALESCE(b.name, ''), c.contract_number, c.contract_date,
			COALESCE(c.subject, ''), c.total_amount, c.contract_currency,
			COALESCE(cp.name, ''), COALESCE(c.approval_status, 'pending_currency_control'),
			COALESCE(c.currency_control_decision, ''), COALESCE(c.currency_control_comment, ''),
			COALESCE(c.currency_control_reviewed_by, ''), c.currency_control_reviewed_at,
			COALESCE(c.compliance_decision, ''), COALESCE(c.compliance_comment, ''),
			COALESCE(c.compliance_reviewed_by, ''), c.compliance_reviewed_at,
			COALESCE(c.rejection_reason, ''), COALESCE(c.created_by, ''), c.created_at
		FROM contracts c
		LEFT JOIN branches b ON b.id = c.branch_id
		LEFT JOIN counterparties cp ON cp.id = c.client_id
		WHERE c.id = $1 AND c.deleted_at IS NULL`

	var res dto.ApprovalItemResponse
	res.EntityType = domain.EntityTypeContract
	var cDate, createdAt time.Time
	var ccAt, compAt *time.Time

	err := r.db.QueryRow(ctx, query, id).Scan(
		&res.EntityID, &res.BranchID, &res.BranchName, &res.DocumentNumber, &cDate,
		&res.Subject, &res.Amount, &res.Currency,
		&res.CounterpartyName, &res.ApprovalStatus,
		&res.CurrencyControlDecision, &res.CurrencyControlComment,
		&res.CurrencyControlReviewedBy, &ccAt,
		&res.ComplianceDecision, &res.ComplianceComment,
		&res.ComplianceReviewedBy, &compAt,
		&res.RejectionReason, &res.CreatedBy, &createdAt,
	)
	if err != nil {
		return nil, err
	}
	res.DocumentDate = cDate.Format("02.01.2006")
	res.CreatedAt = createdAt.Format(time.RFC3339)
	res.CurrencyControlReviewedAt = formatTimePtr(ccAt)
	res.ComplianceReviewedAt = formatTimePtr(compAt)
	return &res, nil
}

func (r *approvalRepo) getInvoiceApproval(ctx context.Context, id int64) (*dto.ApprovalItemResponse, error) {
	query := `
		SELECT 
			i.id, c.branch_id, COALESCE(b.name, ''), i.invoice_number, i.invoice_date,
			COALESCE(i.hs_code, ''), i.amount, i.currency,
			COALESCE(cp.name, ''), COALESCE(i.approval_status, 'pending_currency_control'),
			COALESCE(i.currency_control_decision, ''), COALESCE(i.currency_control_comment, ''),
			COALESCE(i.currency_control_reviewed_by, ''), i.currency_control_reviewed_at,
			COALESCE(i.compliance_decision, ''), COALESCE(i.compliance_comment, ''),
			COALESCE(i.compliance_reviewed_by, ''), i.compliance_reviewed_at,
			COALESCE(i.rejection_reason, ''), COALESCE(i.created_by, ''), i.created_at
		FROM invoices i
		JOIN contracts c ON c.id = i.contract_id
		LEFT JOIN branches b ON b.id = c.branch_id
		LEFT JOIN counterparties cp ON cp.id = c.client_id
		WHERE i.id = $1 AND i.deleted_at IS NULL`

	var res dto.ApprovalItemResponse
	res.EntityType = domain.EntityTypeInvoice
	var iDate, createdAt time.Time
	var ccAt, compAt *time.Time

	err := r.db.QueryRow(ctx, query, id).Scan(
		&res.EntityID, &res.BranchID, &res.BranchName, &res.DocumentNumber, &iDate,
		&res.Subject, &res.Amount, &res.Currency,
		&res.CounterpartyName, &res.ApprovalStatus,
		&res.CurrencyControlDecision, &res.CurrencyControlComment,
		&res.CurrencyControlReviewedBy, &ccAt,
		&res.ComplianceDecision, &res.ComplianceComment,
		&res.ComplianceReviewedBy, &compAt,
		&res.RejectionReason, &res.CreatedBy, &createdAt,
	)
	if err != nil {
		return nil, err
	}
	res.DocumentDate = iDate.Format("02.01.2006")
	res.CreatedAt = createdAt.Format(time.RFC3339)
	res.CurrencyControlReviewedAt = formatTimePtr(ccAt)
	res.ComplianceReviewedAt = formatTimePtr(compAt)
	return &res, nil
}

func (r *approvalRepo) getGTDApproval(ctx context.Context, id int64) (*dto.ApprovalItemResponse, error) {
	query := `
		SELECT 
			g.id, c.branch_id, COALESCE(b.name, ''), g.gtd_number, COALESCE(g.gtd_date, g.created_at),
			COALESCE(g.hs_code, ''), g.gtd_amount, COALESCE(g.gtd_currency, ''),
			COALESCE(cp.name, ''), COALESCE(g.approval_status, 'pending_currency_control'),
			COALESCE(g.currency_control_decision, ''), COALESCE(g.currency_control_comment, ''),
			COALESCE(g.currency_control_reviewed_by, ''), g.currency_control_reviewed_at,
			COALESCE(g.compliance_decision, ''), COALESCE(g.compliance_comment, ''),
			COALESCE(g.compliance_reviewed_by, ''), g.compliance_reviewed_at,
			COALESCE(g.rejection_reason, ''), COALESCE(g.created_by, ''), g.created_at
		FROM gtd g
		JOIN contracts c ON c.id = g.contract_id
		LEFT JOIN branches b ON b.id = c.branch_id
		LEFT JOIN counterparties cp ON cp.id = c.client_id
		WHERE g.id = $1 AND g.deleted_at IS NULL`

	var res dto.ApprovalItemResponse
	res.EntityType = domain.EntityTypeGTD
	var gDate, createdAt time.Time
	var ccAt, compAt *time.Time

	err := r.db.QueryRow(ctx, query, id).Scan(
		&res.EntityID, &res.BranchID, &res.BranchName, &res.DocumentNumber, &gDate,
		&res.Subject, &res.Amount, &res.Currency,
		&res.CounterpartyName, &res.ApprovalStatus,
		&res.CurrencyControlDecision, &res.CurrencyControlComment,
		&res.CurrencyControlReviewedBy, &ccAt,
		&res.ComplianceDecision, &res.ComplianceComment,
		&res.ComplianceReviewedBy, &compAt,
		&res.RejectionReason, &res.CreatedBy, &createdAt,
	)
	if err != nil {
		return nil, err
	}
	res.DocumentDate = gDate.Format("02.01.2006")
	res.CreatedAt = createdAt.Format(time.RFC3339)
	res.CurrencyControlReviewedAt = formatTimePtr(ccAt)
	res.ComplianceReviewedAt = formatTimePtr(compAt)
	return &res, nil
}

func (r *approvalRepo) getAAApproval(ctx context.Context, id int64) (*dto.ApprovalItemResponse, error) {
	query := `
		SELECT 
			aa.id, c.branch_id, COALESCE(b.name, ''), COALESCE(aa.agreement_number, ''), COALESCE(aa.agreement_date, aa.created_at),
			COALESCE(aa.subject, ''), COALESCE(aa.foreign_amount, 0), COALESCE(aa.currency, ''),
			COALESCE(cp.name, ''), COALESCE(aa.approval_status, 'pending_currency_control'),
			COALESCE(aa.currency_control_decision, ''), COALESCE(aa.currency_control_comment, ''),
			COALESCE(aa.currency_control_reviewed_by, ''), aa.currency_control_reviewed_at,
			COALESCE(aa.compliance_decision, ''), COALESCE(aa.compliance_comment, ''),
			COALESCE(aa.compliance_reviewed_by, ''), aa.compliance_reviewed_at,
			COALESCE(aa.rejection_reason, ''), COALESCE(aa.created_by, ''), aa.created_at
		FROM additional_agreements aa
		JOIN contracts c ON c.id = aa.contract_id
		LEFT JOIN branches b ON b.id = c.branch_id
		LEFT JOIN counterparties cp ON cp.id = c.client_id
		WHERE aa.id = $1 AND aa.deleted_at IS NULL`

	var res dto.ApprovalItemResponse
	res.EntityType = domain.EntityTypeAdditionalAgreement
	var aDate, createdAt time.Time
	var ccAt, compAt *time.Time

	err := r.db.QueryRow(ctx, query, id).Scan(
		&res.EntityID, &res.BranchID, &res.BranchName, &res.DocumentNumber, &aDate,
		&res.Subject, &res.Amount, &res.Currency,
		&res.CounterpartyName, &res.ApprovalStatus,
		&res.CurrencyControlDecision, &res.CurrencyControlComment,
		&res.CurrencyControlReviewedBy, &ccAt,
		&res.ComplianceDecision, &res.ComplianceComment,
		&res.ComplianceReviewedBy, &compAt,
		&res.RejectionReason, &res.CreatedBy, &createdAt,
	)
	if err != nil {
		return nil, err
	}
	res.DocumentDate = aDate.Format("02.01.2006")
	res.CreatedAt = createdAt.Format(time.RFC3339)
	res.CurrencyControlReviewedAt = formatTimePtr(ccAt)
	res.ComplianceReviewedAt = formatTimePtr(compAt)
	return &res, nil
}

func (r *approvalRepo) GetPendingApprovals(ctx context.Context, filter dto.PendingApprovalsFilter) ([]dto.ApprovalItemResponse, int, error) {
	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var targetStatuses []string
	switch strings.ToLower(strings.TrimSpace(filter.Stage)) {
	case "currency_control", "cc", "вк":
		targetStatuses = []string{domain.ApprovalStatusPendingCurrencyControl}
	case "compliance", "комплаенс":
		targetStatuses = []string{domain.ApprovalStatusPendingCompliance}
	case "revision", "revision_required", "доработка":
		targetStatuses = []string{domain.ApprovalStatusRevisionRequired}
	default:
		targetStatuses = []string{
			domain.ApprovalStatusPendingCurrencyControl,
			domain.ApprovalStatusPendingCompliance,
			domain.ApprovalStatusRevisionRequired,
		}
	}

	var statusIn []string
	for _, s := range targetStatuses {
		statusIn = append(statusIn, fmt.Sprintf("'%s'", s))
	}
	statusClause := strings.Join(statusIn, ", ")

	var queries []string

	normEntity := strings.ToLower(strings.TrimSpace(filter.EntityType))
	includeContract := normEntity == "" || normEntity == "contract" || normEntity == "contracts"
	includeInvoice := normEntity == "" || normEntity == "invoice" || normEntity == "invoices"
	includeGTD := normEntity == "" || normEntity == "gtd"
	includeAA := normEntity == "" || normEntity == "additional_agreement" || normEntity == "additional_agreements"

	if includeContract {
		q := fmt.Sprintf(`
			SELECT 
				'contract' AS entity_type, c.id AS entity_id, c.branch_id, COALESCE(b.name, '') AS branch_name,
				c.contract_number AS document_number, c.contract_date AS document_date, COALESCE(c.subject, '') AS subject,
				c.total_amount AS amount, c.contract_currency AS currency, COALESCE(cp.name, '') AS counterparty_name,
				COALESCE(c.approval_status, 'pending_currency_control') AS approval_status,
				COALESCE(c.currency_control_decision, '') AS currency_control_decision,
				COALESCE(c.currency_control_comment, '') AS currency_control_comment,
				COALESCE(c.currency_control_reviewed_by, '') AS currency_control_reviewed_by,
				c.currency_control_reviewed_at,
				COALESCE(c.compliance_decision, '') AS compliance_decision,
				COALESCE(c.compliance_comment, '') AS compliance_comment,
				COALESCE(c.compliance_reviewed_by, '') AS compliance_reviewed_by,
				c.compliance_reviewed_at,
				COALESCE(c.rejection_reason, '') AS rejection_reason,
				COALESCE(c.created_by, '') AS created_by, c.created_at
			FROM contracts c
			LEFT JOIN branches b ON b.id = c.branch_id
			LEFT JOIN counterparties cp ON cp.id = c.client_id
			WHERE c.deleted_at IS NULL AND c.approval_status IN (%s)`, statusClause)
		if filter.BranchID != nil && *filter.BranchID > 0 {
			q += fmt.Sprintf(" AND c.branch_id = %d", *filter.BranchID)
		}
		queries = append(queries, q)
	}

	if includeInvoice {
		q := fmt.Sprintf(`
			SELECT 
				'invoice' AS entity_type, i.id AS entity_id, c.branch_id, COALESCE(b.name, '') AS branch_name,
				i.invoice_number AS document_number, i.invoice_date AS document_date, COALESCE(i.hs_code, '') AS subject,
				i.amount AS amount, i.currency AS currency, COALESCE(cp.name, '') AS counterparty_name,
				COALESCE(i.approval_status, 'pending_currency_control') AS approval_status,
				COALESCE(i.currency_control_decision, '') AS currency_control_decision,
				COALESCE(i.currency_control_comment, '') AS currency_control_comment,
				COALESCE(i.currency_control_reviewed_by, '') AS currency_control_reviewed_by,
				i.currency_control_reviewed_at,
				COALESCE(i.compliance_decision, '') AS compliance_decision,
				COALESCE(i.compliance_comment, '') AS compliance_comment,
				COALESCE(i.compliance_reviewed_by, '') AS compliance_reviewed_by,
				i.compliance_reviewed_at,
				COALESCE(i.rejection_reason, '') AS rejection_reason,
				COALESCE(i.created_by, '') AS created_by, i.created_at
			FROM invoices i
			JOIN contracts c ON c.id = i.contract_id
			LEFT JOIN branches b ON b.id = c.branch_id
			LEFT JOIN counterparties cp ON cp.id = c.client_id
			WHERE i.deleted_at IS NULL AND i.approval_status IN (%s)`, statusClause)
		if filter.BranchID != nil && *filter.BranchID > 0 {
			q += fmt.Sprintf(" AND c.branch_id = %d", *filter.BranchID)
		}
		queries = append(queries, q)
	}

	if includeGTD {
		q := fmt.Sprintf(`
			SELECT 
				'gtd' AS entity_type, g.id AS entity_id, c.branch_id, COALESCE(b.name, '') AS branch_name,
				g.gtd_number AS document_number, COALESCE(g.gtd_date, g.created_at) AS document_date, COALESCE(g.hs_code, '') AS subject,
				g.gtd_amount AS amount, COALESCE(g.gtd_currency, '') AS currency, COALESCE(cp.name, '') AS counterparty_name,
				COALESCE(g.approval_status, 'pending_currency_control') AS approval_status,
				COALESCE(g.currency_control_decision, '') AS currency_control_decision,
				COALESCE(g.currency_control_comment, '') AS currency_control_comment,
				COALESCE(g.currency_control_reviewed_by, '') AS currency_control_reviewed_by,
				g.currency_control_reviewed_at,
				COALESCE(g.compliance_decision, '') AS compliance_decision,
				COALESCE(g.compliance_comment, '') AS compliance_comment,
				COALESCE(g.compliance_reviewed_by, '') AS compliance_reviewed_by,
				g.compliance_reviewed_at,
				COALESCE(g.rejection_reason, '') AS rejection_reason,
				COALESCE(g.created_by, '') AS created_by, g.created_at
			FROM gtd g
			JOIN contracts c ON c.id = g.contract_id
			LEFT JOIN branches b ON b.id = c.branch_id
			LEFT JOIN counterparties cp ON cp.id = c.client_id
			WHERE g.deleted_at IS NULL AND g.approval_status IN (%s)`, statusClause)
		if filter.BranchID != nil && *filter.BranchID > 0 {
			q += fmt.Sprintf(" AND c.branch_id = %d", *filter.BranchID)
		}
		queries = append(queries, q)
	}

	if includeAA {
		q := fmt.Sprintf(`
			SELECT 
				'additional_agreement' AS entity_type, aa.id AS entity_id, c.branch_id, COALESCE(b.name, '') AS branch_name,
				COALESCE(aa.agreement_number, '') AS document_number, COALESCE(aa.agreement_date, aa.created_at) AS document_date, COALESCE(aa.subject, '') AS subject,
				COALESCE(aa.foreign_amount, 0) AS amount, COALESCE(aa.currency, '') AS currency, COALESCE(cp.name, '') AS counterparty_name,
				COALESCE(aa.approval_status, 'pending_currency_control') AS approval_status,
				COALESCE(aa.currency_control_decision, '') AS currency_control_decision,
				COALESCE(aa.currency_control_comment, '') AS currency_control_comment,
				COALESCE(aa.currency_control_reviewed_by, '') AS currency_control_reviewed_by,
				aa.currency_control_reviewed_at,
				COALESCE(aa.compliance_decision, '') AS compliance_decision,
				COALESCE(aa.compliance_comment, '') AS compliance_comment,
				COALESCE(aa.compliance_reviewed_by, '') AS compliance_reviewed_by,
				aa.compliance_reviewed_at,
				COALESCE(aa.rejection_reason, '') AS rejection_reason,
				COALESCE(aa.created_by, '') AS created_by, aa.created_at
			FROM additional_agreements aa
			JOIN contracts c ON c.id = aa.contract_id
			LEFT JOIN branches b ON b.id = c.branch_id
			LEFT JOIN counterparties cp ON cp.id = c.client_id
			WHERE aa.deleted_at IS NULL AND aa.approval_status IN (%s)`, statusClause)
		if filter.BranchID != nil && *filter.BranchID > 0 {
			q += fmt.Sprintf(" AND c.branch_id = %d", *filter.BranchID)
		}
		queries = append(queries, q)
	}

	if len(queries) == 0 {
		return []dto.ApprovalItemResponse{}, 0, nil
	}

	unionQuery := strings.Join(queries, " UNION ALL ")
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM (%s) AS total_tbl", unionQuery)

	var total int
	err := r.db.QueryRow(ctx, countQuery).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	pagedQuery := fmt.Sprintf("%s ORDER BY created_at DESC LIMIT %d OFFSET %d", unionQuery, pageSize, offset)
	rows, err := r.db.Query(ctx, pagedQuery)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	var items []dto.ApprovalItemResponse
	for rows.Next() {
		var it dto.ApprovalItemResponse
		var docDate, createdAt time.Time
		var ccAt, compAt *time.Time

		err := rows.Scan(
			&it.EntityType, &it.EntityID, &it.BranchID, &it.BranchName,
			&it.DocumentNumber, &docDate, &it.Subject,
			&it.Amount, &it.Currency, &it.CounterpartyName,
			&it.ApprovalStatus,
			&it.CurrencyControlDecision, &it.CurrencyControlComment,
			&it.CurrencyControlReviewedBy, &ccAt,
			&it.ComplianceDecision, &it.ComplianceComment,
			&it.ComplianceReviewedBy, &compAt,
			&it.RejectionReason, &it.CreatedBy, &createdAt,
		)
		if err != nil {
			return nil, 0, err
		}
		it.DocumentDate = docDate.Format("02.01.2006")
		it.CreatedAt = createdAt.Format(time.RFC3339)
		it.CurrencyControlReviewedAt = formatTimePtr(ccAt)
		it.ComplianceReviewedAt = formatTimePtr(compAt)
		items = append(items, it)
	}
	if items == nil {
		items = []dto.ApprovalItemResponse{}
	}

	return items, total, nil
}
