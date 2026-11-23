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
	case domain.EntityTypeGTD:
		return "gtd", nil
	case domain.EntityTypePaymentOrder:
		return "payment_orders", nil
	default:
		return "", fmt.Errorf("неизвестный тип документа для согласования: %s (допустимы: gtd, payment_order)", entityType)
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
	EntityType                string     `gorm:"column:entity_type"`
	EntityID                  int64      `gorm:"column:entity_id"`
	BranchID                  *int       `gorm:"column:branch_id"`
	BranchName                string     `gorm:"column:branch_name"`
	DocumentNumber            string     `gorm:"column:document_number"`
	DocumentPath              *string    `gorm:"column:document_path"`
	DocumentDate              time.Time  `gorm:"column:document_date"`
	Subject                   string     `gorm:"column:subject"`
	Amount                    float64    `gorm:"column:amount"`
	Currency                  string     `gorm:"column:currency"`
	CounterpartyName          string     `gorm:"column:counterparty_name"`
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
	CreatedBy                 string     `gorm:"column:created_by"`
	CreatedAt                 time.Time  `gorm:"column:created_at"`
}

func (row *rawApprovalRow) toDTO(entityType string) dto.ApprovalItemResponse {
	if entityType == "" {
		entityType = row.EntityType
	}
	subject := row.Subject
	if entityType == domain.EntityTypeGTD && strings.TrimSpace(subject) == "" {
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
	if entityType == domain.EntityTypeGTD {
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
	return dto.ApprovalItemResponse{
		EntityType:                entityType,
		EntityID:                  row.EntityID,
		BranchID:                  row.BranchID,
		BranchName:                row.BranchName,
		DocumentNumber:            row.DocumentNumber,
		DocumentPath:              row.DocumentPath,
		DocumentDate:              row.DocumentDate.Format("02.01.2006"),
		Subject:                   subject,
		Amount:                    row.Amount,
		Currency:                  row.Currency,
		CounterpartyName:          row.CounterpartyName,
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

func (r *approvalRepo) setReviewDecision(ctx context.Context, stage, entityType string, id int64, decision, comment, reviewer string) (*dto.ApprovalItemResponse, error) {
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

	var newStatus string
	normDecision := strings.ToLower(strings.TrimSpace(decision))

	switch stage {
	case "currency_control":
		if currentStatus != domain.ApprovalStatusPendingCurrencyControl && currentStatus != domain.ApprovalStatusRevisionRequired {
			return nil, fmt.Errorf("документ находится в статусе '%s' и не ожидает проверки валютным контролем", currentStatus)
		}
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
	case "compliance":
		if currentStatus != domain.ApprovalStatusPendingCompliance {
			return nil, fmt.Errorf("документ находится в статусе '%s'. Комплаенс-контроль может рассматривать только документы, прошедшие проверку валютным контролем (pending_compliance)", currentStatus)
		}
		switch normDecision {
		case "approve":
			newStatus = domain.ApprovalStatusApproved
		case "revision":
			newStatus = domain.ApprovalStatusRevisionRequired
		case "reject":
			newStatus = domain.ApprovalStatusRejectedCompliance
		default:
			return nil, fmt.Errorf("недопустимое решение комплаенса: %s (допустимы: approve, revision, reject)", decision)
		}
	default:
		return nil, fmt.Errorf("неизвестный этап согласования: %s", stage)
	}

	rejectionReason := ""
	if newStatus == domain.ApprovalStatusRevisionRequired ||
		newStatus == domain.ApprovalStatusRejectedCurrencyControl ||
		newStatus == domain.ApprovalStatusRejectedCompliance {
		rejectionReason = comment
	}

	now := time.Now()
	updates := map[string]interface{}{
		"approval_status":          newStatus,
		stage + "_decision":    normDecision,
		stage + "_comment":     comment,
		stage + "_reviewed_by": reviewer,
		stage + "_reviewed_at": &now,
		"rejection_reason":         rejectionReason,
		"updated_at":               &now,
	}

	if newStatus == domain.ApprovalStatusRejectedCurrencyControl || newStatus == domain.ApprovalStatusRejectedCompliance {
		updates["deleted_at"] = &now
		updates["deleted_by"] = reviewer
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

func (r *approvalRepo) SetCurrencyControlDecision(ctx context.Context, entityType string, id int64, decision, comment, reviewer string) (*dto.ApprovalItemResponse, error) {
	return r.setReviewDecision(ctx, "currency_control", entityType, id, decision, comment, reviewer)
}

func (r *approvalRepo) SetComplianceDecision(ctx context.Context, entityType string, id int64, decision, comment, reviewer string) (*dto.ApprovalItemResponse, error) {
	return r.setReviewDecision(ctx, "compliance", entityType, id, decision, comment, reviewer)
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
	case domain.EntityTypeGTD:
		res, err = r.getGTDApproval(ctx, id)
	case domain.EntityTypePaymentOrder:
		res, err = r.getPaymentOrderApproval(ctx, id)
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

func (r *approvalRepo) getGTDApproval(ctx context.Context, id int64) (*dto.ApprovalItemResponse, error) {
	var row rawApprovalRow
	err := r.db.WithContext(ctx).Table("gtd g").
		Select(BuildApprovalSelect("g", "gtd", "g.gtd_number", "g.gtd_date", "''", "g.gtd_amount", "g.currency")).
		Joins("LEFT JOIN contracts c ON c.id = g.contract_id").
		Joins("LEFT JOIN invoices i ON i.id = g.invoice_id").
		Joins("LEFT JOIN branches b ON b.id = c.branch_id").
		Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
		Where("g.id = ?", id).
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	res := row.toDTO(domain.EntityTypeGTD)
	return &res, nil
}


func (r *approvalRepo) getPaymentOrderApproval(ctx context.Context, id int64) (*dto.ApprovalItemResponse, error) {
	var row rawApprovalRow
	err := r.db.WithContext(ctx).Table("payment_orders po").
		Select(BuildApprovalSelect("po", "payment_order", "po.id::text", "po.value_date", "po.payment_purpose", "po.amount", "COALESCE(po.currency, '')")).
		Joins("LEFT JOIN contracts c ON c.id = po.contract_id").
		Joins("LEFT JOIN invoices i ON i.id = po.invoice_id").
		Joins("LEFT JOIN branches b ON b.id = c.branch_id").
		Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
		Where("po.id = ?", id).
		Take(&row).Error
	if err != nil {
		return nil, err
	}
	res := row.toDTO(domain.EntityTypePaymentOrder)
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
	includeGTD := normEntity == "" || normEntity == domain.EntityTypeGTD
	includePO := normEntity == "" || normEntity == domain.EntityTypePaymentOrder

	var subQueries []*gorm.DB

	if includeGTD {
		q := r.db.WithContext(ctx).Table("gtd g").
			Select(BuildApprovalSelect("g", "gtd", "g.gtd_number", "g.gtd_date", "''", "g.gtd_amount", "g.currency")).
			Joins("JOIN contracts c ON c.id = g.contract_id").
			Joins("LEFT JOIN invoices i ON i.id = g.invoice_id").
			Joins("LEFT JOIN branches b ON b.id = c.branch_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("g.deleted_at IS NULL AND g.approval_status IN ?", targetStatuses)
		if filter.BranchID != nil && *filter.BranchID > 0 {
			q = q.Where("c.branch_id = ?", *filter.BranchID)
		}
		subQueries = append(subQueries, q)
	}

	if includePO {
		q := r.db.WithContext(ctx).Table("payment_orders po").
			Select(BuildApprovalSelect("po", "payment_order", "po.id::text", "po.value_date", "po.payment_purpose", "po.amount", "po.currency")).
			Joins("JOIN contracts c ON c.id = po.contract_id").
			Joins("LEFT JOIN invoices i ON i.id = po.invoice_id").
			Joins("LEFT JOIN branches b ON b.id = c.branch_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("po.deleted_at IS NULL AND po.approval_status IN ?", targetStatuses)
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

func (r *approvalRepo) GetAcceptedApprovals(ctx context.Context, filter dto.AcceptedApprovalsFilter) ([]dto.ApprovalItemResponse, int, error) {
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
	includeGTD := normEntity == "" || normEntity == domain.EntityTypeGTD
	includePO := normEntity == "" || normEntity == domain.EntityTypePaymentOrder

	var subQueries []*gorm.DB

	if includeGTD {
		q := r.db.WithContext(ctx).Table("gtd g").
			Select(BuildApprovalSelect("g", "gtd", "g.gtd_number", "g.gtd_date", "''", "g.gtd_amount", "g.currency")).
			Joins("JOIN contracts c ON c.id = g.contract_id").
			Joins("LEFT JOIN invoices i ON i.id = g.invoice_id").
			Joins("LEFT JOIN branches b ON b.id = c.branch_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("g.deleted_at IS NULL AND (g.approval_status = ? OR (g.compliance_decision IN ('approve', 'approved', 'accepted') AND g.currency_control_decision IN ('approve', 'approved', 'accepted')))", domain.ApprovalStatusApproved)
		if filter.BranchID != nil && *filter.BranchID > 0 {
			q = q.Where("c.branch_id = ?", *filter.BranchID)
		}
		subQueries = append(subQueries, q)
	}

	if includePO {
		q := r.db.WithContext(ctx).Table("payment_orders po").
			Select(BuildApprovalSelect("po", "payment_order", "po.id::text", "po.value_date", "po.payment_purpose", "po.amount", "po.currency")).
			Joins("JOIN contracts c ON c.id = po.contract_id").
			Joins("LEFT JOIN invoices i ON i.id = po.invoice_id").
			Joins("LEFT JOIN branches b ON b.id = c.branch_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("po.deleted_at IS NULL AND (po.approval_status = ? OR (po.compliance_decision IN ('approve', 'approved', 'accepted') AND po.currency_control_decision IN ('approve', 'approved', 'accepted')))", domain.ApprovalStatusApproved)
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
	if err := combinedQuery.Order("COALESCE(compliance_reviewed_at, currency_control_reviewed_at, created_at) DESC").
		Limit(pageSize).Offset(offset).Scan(&rows).Error; err != nil {
		return nil, 0, err
	}

	items := make([]dto.ApprovalItemResponse, len(rows))
	for i := range rows {
		items[i] = rows[i].toDTO("")
	}
	_ = r.enrichApprovalItems(ctx, items)

	return items, int(total), nil
}
