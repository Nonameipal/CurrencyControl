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
	DocumentPath              *string
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
		DocumentPath:              row.DocumentPath,
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

	if newStatus == domain.ApprovalStatusRejectedCurrencyControl {
		now2 := time.Now()
		r.db.WithContext(ctx).Table(tbl).Where("id = ? AND deleted_at IS NULL", id).Updates(map[string]interface{}{
			"deleted_at": &now2,
			"deleted_by": reviewer,
		})
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
	if newStatus == domain.ApprovalStatusRejectedCompliance {
		now2 := time.Now()
		r.db.WithContext(ctx).Table(tbl).Where("id = ? AND deleted_at IS NULL", id).Updates(map[string]interface{}{
			"deleted_at": &now2,
			"deleted_by": reviewer,
		})
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
	logins := make([]string, 0, len(items)*3)
	for _, it := range items {
		logins = append(logins, it.CreatedBy, it.CurrencyControlReviewedBy, it.ComplianceReviewedBy)
	}
	userMap := fetchUserBriefs(ctx, r.db, logins)

	for i := range items {
		if items[i].CreatedBy != "" {
			b := userMap[items[i].CreatedBy]
			items[i].Creator = &b
		}
		if items[i].CurrencyControlReviewedBy != "" {
			b := userMap[items[i].CurrencyControlReviewedBy]
			items[i].CurrencyControlReviewer = &b
		}
		if items[i].ComplianceReviewedBy != "" {
			b := userMap[items[i].ComplianceReviewedBy]
			items[i].ComplianceReviewer = &b
		}
	}
	return nil
}

func (r *approvalRepo) getContractApproval(ctx context.Context, id int64) (*dto.ApprovalItemResponse, error) {
	var row rawApprovalRow
	err := r.db.WithContext(ctx).Table("contracts c").
		Select(BuildApprovalSelect("c", "contract", "c.contract_number", "c.contract_date", "c.subject", "c.total_amount", "c.contract_currency")).
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
		Select(BuildApprovalSelect("i", "invoice", "i.invoice_number", "i.invoice_date", "i.hs_code", "i.amount", "i.currency")).
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
		Select(BuildApprovalSelect("g", "gtd", "g.gtd_number", "COALESCE(g.gtd_date, g.created_at)", "g.hs_code", "g.gtd_amount", "COALESCE(g.gtd_currency, '')")).
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
		Select(BuildApprovalSelect("aa", "additional_agreement", "COALESCE(aa.agreement_number, '')", "COALESCE(aa.agreement_date, aa.created_at)", "aa.subject", "COALESCE(aa.foreign_amount, 0)", "COALESCE(aa.currency, '')")).
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
			Select(BuildApprovalSelect("c", "contract", "c.contract_number", "c.contract_date", "c.subject", "c.total_amount", "c.contract_currency")).
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
			Select(BuildApprovalSelect("i", "invoice", "i.invoice_number", "i.invoice_date", "i.hs_code", "i.amount", "i.currency")).
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
			Select(BuildApprovalSelect("g", "gtd", "g.gtd_number", "COALESCE(g.gtd_date, g.created_at)", "g.hs_code", "g.gtd_amount", "COALESCE(g.gtd_currency, '')")).
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
			Select(BuildApprovalSelect("aa", "additional_agreement", "COALESCE(aa.agreement_number, '')", "COALESCE(aa.agreement_date, aa.created_at)", "aa.subject", "COALESCE(aa.foreign_amount, 0)", "COALESCE(aa.currency, '')")).
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
