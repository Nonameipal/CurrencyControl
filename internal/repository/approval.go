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

type approvalRepo struct {
	db *gorm.DB
}

func NewApprovalRepository(db *gorm.DB) ports.ApprovalRepository {
	return &approvalRepo{db: db}
}

func getApprovalTableName(entityType string) (string, error) {
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

type rawApprovalRow struct {
	EntityType                string
	EntityID                  int64
	BranchID                  *int
	BranchName                string
	DocumentNumber            string
	DocumentDate              time.Time
	Subject                   string
	Amount                    float64
	Currency                  string
	CounterpartyName          string
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
	CreatedBy                 string
	CreatedAt                 time.Time
}

func (row *rawApprovalRow) toDTO(entityType string) dto.ApprovalItemResponse {
	if entityType == "" {
		entityType = row.EntityType
	}
	return dto.ApprovalItemResponse{
		EntityType:                entityType,
		EntityID:                  row.EntityID,
		BranchID:                  row.BranchID,
		BranchName:                row.BranchName,
		DocumentNumber:            row.DocumentNumber,
		DocumentDate:              row.DocumentDate.Format("02.01.2006"),
		Subject:                   row.Subject,
		Amount:                    row.Amount,
		Currency:                  row.Currency,
		CounterpartyName:          row.CounterpartyName,
		ApprovalStatus:            row.ApprovalStatus,
		CurrencyControlDecision:   row.CurrencyControlDecision,
		CurrencyControlComment:    row.CurrencyControlComment,
		CurrencyControlReviewedBy: row.CurrencyControlReviewedBy,
		CurrencyControlReviewedAt: formatTimePtr(row.CurrencyControlReviewedAt),
		ComplianceDecision:        row.ComplianceDecision,
		ComplianceComment:         row.ComplianceComment,
		ComplianceReviewedBy:      row.ComplianceReviewedBy,
		ComplianceReviewedAt:      formatTimePtr(row.ComplianceReviewedAt),
		RejectionReason:           row.RejectionReason,
		CreatedBy:                 row.CreatedBy,
		CreatedAt:                 row.CreatedAt.Format(time.RFC3339),
	}
}

func (r *approvalRepo) SetCurrencyControlDecision(ctx context.Context, entityType string, id int64, decision, comment, reviewer string) (*dto.ApprovalItemResponse, error) {
	tbl, err := getApprovalTableName(entityType)
	if err != nil {
		return nil, err
	}

	var currentStatus string
	if err := r.db.WithContext(ctx).Table(tbl).
		Select("approval_status").
		Where("id = ? AND deleted_at IS NULL", id).
		Scan(&currentStatus).Error; err != nil {
		return nil, fmt.Errorf("документ не найден: %w", err)
	}
	if currentStatus == "" {
		return nil, fmt.Errorf("документ не найден")
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

	now := time.Now()
	updates := map[string]interface{}{
		"approval_status":              newStatus,
		"currency_control_decision":    normDecision,
		"currency_control_comment":     comment,
		"currency_control_reviewed_by": reviewer,
		"currency_control_reviewed_at": &now,
		"rejection_reason":             rejectionReason,
		"updated_at":                   &now,
	}

	res := r.db.WithContext(ctx).Table(tbl).Where("id = ? AND deleted_at IS NULL", id).Updates(updates)
	if res.Error != nil {
		return nil, fmt.Errorf("ошибка обновления статуса документа: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, fmt.Errorf("документ не найден")
	}

	return r.GetApprovalDetail(ctx, entityType, id)
}

func (r *approvalRepo) SetComplianceDecision(ctx context.Context, entityType string, id int64, decision, comment, reviewer string) (*dto.ApprovalItemResponse, error) {
	tbl, err := getApprovalTableName(entityType)
	if err != nil {
		return nil, err
	}

	var currentStatus string
	if err := r.db.WithContext(ctx).Table(tbl).
		Select("approval_status").
		Where("id = ? AND deleted_at IS NULL", id).
		Scan(&currentStatus).Error; err != nil {
		return nil, fmt.Errorf("документ не найден: %w", err)
	}
	if currentStatus == "" {
		return nil, fmt.Errorf("документ не найден")
	}

	if currentStatus != domain.ApprovalStatusPendingCompliance {
		return nil, fmt.Errorf("документ находится в статусе '%s'. Комплаенс-контроль может рассматривать только документы, прошедшие проверку валютным контролем (pending_compliance)", currentStatus)
	}

	var newStatus string
	normDecision := strings.ToLower(strings.TrimSpace(decision))
	switch normDecision {
	case "approve":
		newStatus = domain.ApprovalStatusApproved
	case "reject":
		newStatus = domain.ApprovalStatusRejectedCompliance
	default:
		return nil, fmt.Errorf("недопустимое решение комплаенс-контроля: %s (допустимы: approve, reject)", decision)
	}

	rejectionReason := ""
	if newStatus == domain.ApprovalStatusRejectedCompliance {
		rejectionReason = comment
	}

	now := time.Now()
	updates := map[string]interface{}{
		"approval_status":        newStatus,
		"compliance_decision":    normDecision,
		"compliance_comment":     comment,
		"compliance_reviewed_by": reviewer,
		"compliance_reviewed_at": &now,
		"rejection_reason":       rejectionReason,
		"updated_at":             &now,
	}

	res := r.db.WithContext(ctx).Table(tbl).Where("id = ? AND deleted_at IS NULL", id).Updates(updates)
	if res.Error != nil {
		return nil, fmt.Errorf("ошибка обновления статуса документа: %w", res.Error)
	}
	if res.RowsAffected == 0 {
		return nil, fmt.Errorf("документ не найден")
	}

	if newStatus == domain.ApprovalStatusApproved {
		if tbl == "contracts" || tbl == "additional_agreements" {
			r.db.WithContext(ctx).Table(tbl).
				Where("id = ? AND (status IS NULL OR status != 'archived')", id).
				Update("status", "active")
		}
	}

	return r.GetApprovalDetail(ctx, entityType, id)
}

func (r *approvalRepo) ResetToPendingCurrencyControl(ctx context.Context, entityType string, id int64) error {
	tbl, err := getApprovalTableName(entityType)
	if err != nil {
		return err
	}

	now := time.Now()
	updates := map[string]interface{}{
		"approval_status":  domain.ApprovalStatusPendingCurrencyControl,
		"rejection_reason": "",
		"updated_at":       &now,
	}

	return r.db.WithContext(ctx).Table(tbl).Where("id = ? AND deleted_at IS NULL", id).Updates(updates).Error
}

func (r *approvalRepo) GetApprovalDetail(ctx context.Context, entityType string, id int64) (*dto.ApprovalItemResponse, error) {
	norm := strings.ToLower(strings.TrimSpace(entityType))
	var res *dto.ApprovalItemResponse
	var err error
	switch norm {
	case "contract", "contracts":
		res, err = r.getContractApproval(ctx, id)
	case "invoice", "invoices":
		res, err = r.getInvoiceApproval(ctx, id)
	case "gtd":
		res, err = r.getGTDApproval(ctx, id)
	case "additional_agreement", "additional-agreement", "additional_agreements", "additional-agreements":
		res, err = r.getAAApproval(ctx, id)
	default:
		return nil, fmt.Errorf("неизвестный тип документа: %s", entityType)
	}
	if err != nil {
		return nil, err
	}
	if res != nil {
		items := []dto.ApprovalItemResponse{*res}
		_ = r.enrichApprovalItems(ctx, items)
		*res = items[0]
	}
	return res, nil
}

func (r *approvalRepo) enrichApprovalItems(ctx context.Context, items []dto.ApprovalItemResponse) error {
	loginsMap := make(map[string]bool)
	for _, it := range items {
		if it.CreatedBy != "" {
			loginsMap[it.CreatedBy] = true
		}
		if it.CurrencyControlReviewedBy != "" {
			loginsMap[it.CurrencyControlReviewedBy] = true
		}
		if it.ComplianceReviewedBy != "" {
			loginsMap[it.ComplianceReviewedBy] = true
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
	if err := r.db.WithContext(ctx).Table("users").
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

	for i := range items {
		if b, ok := userMap[items[i].CreatedBy]; ok {
			items[i].Creator = &b
		} else if items[i].CreatedBy != "" {
			items[i].Creator = &domain.UserBrief{Login: items[i].CreatedBy}
		}
		if b, ok := userMap[items[i].CurrencyControlReviewedBy]; ok {
			items[i].CurrencyControlReviewer = &b
		} else if items[i].CurrencyControlReviewedBy != "" {
			items[i].CurrencyControlReviewer = &domain.UserBrief{Login: items[i].CurrencyControlReviewedBy}
		}
		if b, ok := userMap[items[i].ComplianceReviewedBy]; ok {
			items[i].ComplianceReviewer = &b
		} else if items[i].ComplianceReviewedBy != "" {
			items[i].ComplianceReviewer = &domain.UserBrief{Login: items[i].ComplianceReviewedBy}
		}
	}
	return nil
}

func (r *approvalRepo) getContractApproval(ctx context.Context, id int64) (*dto.ApprovalItemResponse, error) {
	var row rawApprovalRow
	err := r.db.WithContext(ctx).Table("contracts c").
		Select(`
			c.id AS entity_id, c.branch_id, COALESCE(b.name, '') AS branch_name, c.contract_number AS document_number, c.contract_date AS document_date,
			COALESCE(c.subject, '') AS subject, c.total_amount AS amount, c.contract_currency AS currency,
			COALESCE(cp.llc, '') AS counterparty_name, COALESCE(c.approval_status, 'pending_currency_control') AS approval_status,
			COALESCE(c.currency_control_decision, '') AS currency_control_decision, COALESCE(c.currency_control_comment, '') AS currency_control_comment,
			COALESCE(c.currency_control_reviewed_by, '') AS currency_control_reviewed_by, c.currency_control_reviewed_at,
			COALESCE(c.compliance_decision, '') AS compliance_decision, COALESCE(c.compliance_comment, '') AS compliance_comment,
			COALESCE(c.compliance_reviewed_by, '') AS compliance_reviewed_by, c.compliance_reviewed_at,
			COALESCE(c.rejection_reason, '') AS rejection_reason, COALESCE(c.created_by, '') AS created_by, c.created_at
		`).
		Joins("LEFT JOIN branches b ON b.id = c.branch_id").
		Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
		Where("c.id = ? AND c.deleted_at IS NULL", id).
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	res := row.toDTO(domain.EntityTypeContract)
	return &res, nil
}

func (r *approvalRepo) getInvoiceApproval(ctx context.Context, id int64) (*dto.ApprovalItemResponse, error) {
	var row rawApprovalRow
	err := r.db.WithContext(ctx).Table("invoices i").
		Select(`
			i.id AS entity_id, c.branch_id, COALESCE(b.name, '') AS branch_name, i.invoice_number AS document_number, i.invoice_date AS document_date,
			COALESCE(i.hs_code, '') AS subject, i.amount AS amount, i.currency AS currency,
			COALESCE(cp.llc, '') AS counterparty_name, COALESCE(i.approval_status, 'pending_currency_control') AS approval_status,
			COALESCE(i.currency_control_decision, '') AS currency_control_decision, COALESCE(i.currency_control_comment, '') AS currency_control_comment,
			COALESCE(i.currency_control_reviewed_by, '') AS currency_control_reviewed_by, i.currency_control_reviewed_at,
			COALESCE(i.compliance_decision, '') AS compliance_decision, COALESCE(i.compliance_comment, '') AS compliance_comment,
			COALESCE(i.compliance_reviewed_by, '') AS compliance_reviewed_by, i.compliance_reviewed_at,
			COALESCE(i.rejection_reason, '') AS rejection_reason, COALESCE(i.created_by, '') AS created_by, i.created_at
		`).
		Joins("JOIN contracts c ON c.id = i.contract_id").
		Joins("LEFT JOIN branches b ON b.id = c.branch_id").
		Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
		Where("i.id = ? AND i.deleted_at IS NULL", id).
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	res := row.toDTO(domain.EntityTypeInvoice)
	return &res, nil
}

func (r *approvalRepo) getGTDApproval(ctx context.Context, id int64) (*dto.ApprovalItemResponse, error) {
	var row rawApprovalRow
	err := r.db.WithContext(ctx).Table("gtd g").
		Select(`
			g.id AS entity_id, c.branch_id, COALESCE(b.name, '') AS branch_name, g.gtd_number AS document_number, COALESCE(g.gtd_date, g.created_at) AS document_date,
			COALESCE(g.hs_code, '') AS subject, g.gtd_amount AS amount, COALESCE(g.gtd_currency, '') AS currency,
			COALESCE(cp.llc, '') AS counterparty_name, COALESCE(g.approval_status, 'pending_currency_control') AS approval_status,
			COALESCE(g.currency_control_decision, '') AS currency_control_decision, COALESCE(g.currency_control_comment, '') AS currency_control_comment,
			COALESCE(g.currency_control_reviewed_by, '') AS currency_control_reviewed_by, g.currency_control_reviewed_at,
			COALESCE(g.compliance_decision, '') AS compliance_decision, COALESCE(g.compliance_comment, '') AS compliance_comment,
			COALESCE(g.compliance_reviewed_by, '') AS compliance_reviewed_by, g.compliance_reviewed_at,
			COALESCE(g.rejection_reason, '') AS rejection_reason, COALESCE(g.created_by, '') AS created_by, g.created_at
		`).
		Joins("JOIN contracts c ON c.id = g.contract_id").
		Joins("LEFT JOIN branches b ON b.id = c.branch_id").
		Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
		Where("g.id = ? AND g.deleted_at IS NULL", id).
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	res := row.toDTO(domain.EntityTypeGTD)
	return &res, nil
}

func (r *approvalRepo) getAAApproval(ctx context.Context, id int64) (*dto.ApprovalItemResponse, error) {
	var row rawApprovalRow
	err := r.db.WithContext(ctx).Table("additional_agreements aa").
		Select(`
			aa.id AS entity_id, c.branch_id, COALESCE(b.name, '') AS branch_name, COALESCE(aa.agreement_number, '') AS document_number, COALESCE(aa.agreement_date, aa.created_at) AS document_date,
			COALESCE(aa.subject, '') AS subject, COALESCE(aa.foreign_amount, 0) AS amount, COALESCE(aa.currency, '') AS currency,
			COALESCE(cp.llc, '') AS counterparty_name, COALESCE(aa.approval_status, 'pending_currency_control') AS approval_status,
			COALESCE(aa.currency_control_decision, '') AS currency_control_decision, COALESCE(aa.currency_control_comment, '') AS currency_control_comment,
			COALESCE(aa.currency_control_reviewed_by, '') AS currency_control_reviewed_by, aa.currency_control_reviewed_at,
			COALESCE(aa.compliance_decision, '') AS compliance_decision, COALESCE(aa.compliance_comment, '') AS compliance_comment,
			COALESCE(aa.compliance_reviewed_by, '') AS compliance_reviewed_by, aa.compliance_reviewed_at,
			COALESCE(aa.rejection_reason, '') AS rejection_reason, COALESCE(aa.created_by, '') AS created_by, aa.created_at
		`).
		Joins("JOIN contracts c ON c.id = aa.contract_id").
		Joins("LEFT JOIN branches b ON b.id = c.branch_id").
		Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
		Where("aa.id = ? AND aa.deleted_at IS NULL", id).
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	res := row.toDTO(domain.EntityTypeAdditionalAgreement)
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

	normEntity := strings.ToLower(strings.TrimSpace(filter.EntityType))
	includeContract := normEntity == "" || normEntity == "contract" || normEntity == "contracts"
	includeInvoice := normEntity == "" || normEntity == "invoice" || normEntity == "invoices"
	includeGTD := normEntity == "" || normEntity == "gtd"
	includeAA := normEntity == "" || normEntity == "additional_agreement" || normEntity == "additional_agreements"

	var subQueries []*gorm.DB

	if includeContract {
		q := r.db.WithContext(ctx).Table("contracts c").
			Select(`
				'contract' AS entity_type, c.id AS entity_id, c.branch_id, COALESCE(b.name, '') AS branch_name,
				c.contract_number AS document_number, c.contract_date AS document_date, COALESCE(c.subject, '') AS subject,
				c.total_amount AS amount, c.contract_currency AS currency, COALESCE(cp.llc, '') AS counterparty_name,
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
			`).
			Joins("LEFT JOIN branches b ON b.id = c.branch_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("c.deleted_at IS NULL AND c.approval_status IN ?", targetStatuses)
		if filter.BranchID != nil && *filter.BranchID > 0 {
			q = q.Where("c.branch_id = ?", *filter.BranchID)
		}
		subQueries = append(subQueries, q)
	}

	if includeInvoice {
		q := r.db.WithContext(ctx).Table("invoices i").
			Select(`
				'invoice' AS entity_type, i.id AS entity_id, c.branch_id, COALESCE(b.name, '') AS branch_name,
				i.invoice_number AS document_number, i.invoice_date AS document_date, COALESCE(i.hs_code, '') AS subject,
				i.amount AS amount, i.currency AS currency, COALESCE(cp.llc, '') AS counterparty_name,
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
			`).
			Joins("JOIN contracts c ON c.id = i.contract_id").
			Joins("LEFT JOIN branches b ON b.id = c.branch_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("i.deleted_at IS NULL AND i.approval_status IN ?", targetStatuses)
		if filter.BranchID != nil && *filter.BranchID > 0 {
			q = q.Where("c.branch_id = ?", *filter.BranchID)
		}
		subQueries = append(subQueries, q)
	}

	if includeGTD {
		q := r.db.WithContext(ctx).Table("gtd g").
			Select(`
				'gtd' AS entity_type, g.id AS entity_id, c.branch_id, COALESCE(b.name, '') AS branch_name,
				g.gtd_number AS document_number, COALESCE(g.gtd_date, g.created_at) AS document_date, COALESCE(g.hs_code, '') AS subject,
				g.gtd_amount AS amount, COALESCE(g.gtd_currency, '') AS currency, COALESCE(cp.llc, '') AS counterparty_name,
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
			`).
			Joins("JOIN contracts c ON c.id = g.contract_id").
			Joins("LEFT JOIN branches b ON b.id = c.branch_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("g.deleted_at IS NULL AND g.approval_status IN ?", targetStatuses)
		if filter.BranchID != nil && *filter.BranchID > 0 {
			q = q.Where("c.branch_id = ?", *filter.BranchID)
		}
		subQueries = append(subQueries, q)
	}

	if includeAA {
		q := r.db.WithContext(ctx).Table("additional_agreements aa").
			Select(`
				'additional_agreement' AS entity_type, aa.id AS entity_id, c.branch_id, COALESCE(b.name, '') AS branch_name,
				COALESCE(aa.agreement_number, '') AS document_number, COALESCE(aa.agreement_date, aa.created_at) AS document_date, COALESCE(aa.subject, '') AS subject,
				COALESCE(aa.foreign_amount, 0) AS amount, COALESCE(aa.currency, '') AS currency, COALESCE(cp.llc, '') AS counterparty_name,
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
			`).
			Joins("JOIN contracts c ON c.id = aa.contract_id").
			Joins("LEFT JOIN branches b ON b.id = c.branch_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("aa.deleted_at IS NULL AND aa.approval_status IN ?", targetStatuses)
		if filter.BranchID != nil && *filter.BranchID > 0 {
			q = q.Where("c.branch_id = ?", *filter.BranchID)
		}
		subQueries = append(subQueries, q)
	}

	if len(subQueries) == 0 {
		return []dto.ApprovalItemResponse{}, 0, nil
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
		combinedQuery = r.db.WithContext(ctx).Table(fmt.Sprintf("(%s) AS total_tbl", unionSQL), args...)
	}

	var total int64
	if err := combinedQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []rawApprovalRow
	if err := combinedQuery.Order("created_at DESC").Limit(pageSize).Offset(offset).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	items := make([]dto.ApprovalItemResponse, len(rows))
	for i := range rows {
		items[i] = rows[i].toDTO("")
	}
	_ = r.enrichApprovalItems(ctx, items)

	return items, int(total), nil
}
