package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"

	"gorm.io/gorm"
)

type gtdExtensionRepo struct {
	db *gorm.DB
}

func NewGTDExtensionRepository(db *gorm.DB) ports.GTDExtensionRepository {
	return &gtdExtensionRepo{db: db}
}

func (r *gtdExtensionRepo) CreateRequest(ctx context.Context, req domain.GTDExtensionRequest) (domain.GTDExtensionRequest, error) {
	var gtd domain.GTD
	if err := r.db.WithContext(ctx).First(&gtd, req.GTDID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.GTDExtensionRequest{}, fmt.Errorf("ГТД с ID %d не найдена", req.GTDID)
		}
		return domain.GTDExtensionRequest{}, fmt.Errorf("ошибка проверки ГТД: %w", err)
	}

	var activeCount int64
	err := r.db.WithContext(ctx).Model(&domain.GTDExtensionRequest{}).
		Where("gtd_id = ? AND status IN (?)", req.GTDID, []string{
			domain.ExtensionStatusPendingCurrencyControl,
			domain.ExtensionStatusPendingCompliance,
			domain.ExtensionStatusRevisionRequired,
		}).
		Count(&activeCount).Error
	if err != nil {
		return domain.GTDExtensionRequest{}, fmt.Errorf("ошибка проверки существующих заявок: %w", err)
	}
	if activeCount > 0 {
		return domain.GTDExtensionRequest{}, fmt.Errorf("по данной ГТД уже есть активная заявка на увеличение срока, находящаяся на согласовании или доработке")
	}

	req.ContractID = gtd.ContractID
	req.InvoiceID = gtd.InvoiceID
	req.CurrentDeadline = gtd.DeliveryDeadline
	req.Status = domain.ExtensionStatusPendingCurrencyControl

	if err := r.db.WithContext(ctx).Create(&req).Error; err != nil {
		return domain.GTDExtensionRequest{}, fmt.Errorf("ошибка создания заявки на продление ГТД: %w", err)
	}
	return req, nil
}

func (r *gtdExtensionRepo) UpdateRequest(ctx context.Context, id int64, login, role string, requestedDeadline time.Time, documentPath string) (*domain.GTDExtensionRequest, error) {
	var req domain.GTDExtensionRequest
	if err := r.db.WithContext(ctx).First(&req, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("заявка на продление не найдена")
		}
		return nil, err
	}

	if req.Status != domain.ExtensionStatusRevisionRequired {
		return nil, fmt.Errorf("редактировать можно только заявку, отправленную на доработку (текущий статус: %s)", req.Status)
	}

	if role != domain.RoleAdmin && req.CreatedBy != login {
		return nil, fmt.Errorf("редактировать заявку на доработке может только её создатель")
	}

	now := time.Now()
	updates := map[string]interface{}{
		"requested_deadline":            requestedDeadline,
		"status":                        domain.ExtensionStatusPendingCurrencyControl,
		"currency_control_decision":    "",
		"currency_control_comment":     "",
		"currency_control_reviewed_by": "",
		"currency_control_reviewed_at": nil,
		"rejection_reason":             "",
		"updated_at":                   now,
	}
	if documentPath != "" {
		updates["document_path"] = documentPath
	}

	if err := r.db.WithContext(ctx).Model(&req).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("ошибка обновления заявки: %w", err)
	}

	return r.GetByID(ctx, id)
}

func (r *gtdExtensionRepo) GetByID(ctx context.Context, id int64) (*domain.GTDExtensionRequest, error) {
	var req domain.GTDExtensionRequest
	if err := r.db.WithContext(ctx).First(&req, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("заявка на продление не найдена")
		}
		return nil, err
	}
	list := []domain.GTDExtensionRequest{req}
	r.enrichGTDExtensionRequests(ctx, list)
	return &list[0], nil
}

func (r *gtdExtensionRepo) GetByGTDID(ctx context.Context, gtdID int64) ([]domain.GTDExtensionRequest, error) {
	var list []domain.GTDExtensionRequest
	if err := r.db.WithContext(ctx).
		Where("gtd_id = ?", gtdID).
		Order("created_at DESC").
		Find(&list).Error; err != nil {
		return nil, err
	}
	r.enrichGTDExtensionRequests(ctx, list)
	return list, nil
}

func (r *gtdExtensionRepo) GetPendingRequests(ctx context.Context, stage string, branchID *int, page, pageSize int) ([]domain.GTDExtensionRequest, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	tx := r.db.WithContext(ctx).Model(&domain.GTDExtensionRequest{}).
		Joins("JOIN contracts c ON c.id = gtd_extension_requests.contract_id")

	switch stage {
	case "compliance":
		tx = tx.Where("gtd_extension_requests.status = ?", domain.ExtensionStatusPendingCompliance)
	case "all":
		tx = tx.Where("gtd_extension_requests.status IN (?)", []string{
			domain.ExtensionStatusPendingCurrencyControl,
			domain.ExtensionStatusPendingCompliance,
		})
	default:
		// stage == "currency_control" или пустой
		tx = tx.Where("gtd_extension_requests.status = ?", domain.ExtensionStatusPendingCurrencyControl)
	}

	if branchID != nil && *branchID > 0 {
		tx = tx.Where("c.branch_id = ?", *branchID)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []domain.GTDExtensionRequest
	if err := tx.Order("gtd_extension_requests.created_at ASC").
		Limit(pageSize).Offset(offset).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	r.enrichGTDExtensionRequests(ctx, list)
	return list, int(total), nil
}

func (r *gtdExtensionRepo) enrichGTDExtensionRequests(ctx context.Context, list []domain.GTDExtensionRequest) {
	if len(list) == 0 {
		return
	}
	logins := make([]string, 0, len(list)*4)
	for _, it := range list {
		logins = append(logins, it.CreatedBy, it.CurrencyControlReviewedBy, it.ComplianceReviewedBy)
		if it.ReviewedBy != nil {
			logins = append(logins, *it.ReviewedBy)
		}
	}
	userMap := fetchUserBriefs(ctx, r.db, logins)

	for i := range list {
		if list[i].CreatedBy != "" {
			b := userMap[list[i].CreatedBy]
			list[i].Creator = &b
		}
		if list[i].CurrencyControlReviewedBy != "" {
			b := userMap[list[i].CurrencyControlReviewedBy]
			list[i].CurrencyControlReviewer = &b
		}
		if list[i].ComplianceReviewedBy != "" {
			b := userMap[list[i].ComplianceReviewedBy]
			list[i].ComplianceReviewer = &b
		}
		if list[i].ReviewedBy != nil && *list[i].ReviewedBy != "" {
			b := userMap[*list[i].ReviewedBy]
			list[i].Reviewer = &b
		}
	}
}

func (r *gtdExtensionRepo) SetCurrencyControlDecision(ctx context.Context, id int64, decision, comment, reviewer string) (*domain.GTDExtensionRequest, error) {
	var req domain.GTDExtensionRequest
	if err := r.db.WithContext(ctx).First(&req, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("заявка на продление не найдена")
		}
		return nil, err
	}

	if req.Status != domain.ExtensionStatusPendingCurrencyControl {
		return nil, fmt.Errorf("заявка не ожидает рассмотрения валютным контролем (текущий статус: %s)", req.Status)
	}

	now := time.Now()
	normDecision := strings.ToLower(strings.TrimSpace(decision))
	updates := map[string]interface{}{
		"currency_control_reviewed_by": reviewer,
		"currency_control_reviewed_at": &now,
		"currency_control_comment":     comment,
		"reviewed_by":                  &reviewer,
		"reviewed_at":                  &now,
		"comment":                      comment,
		"updated_at":                   now,
	}

	switch normDecision {
	case "accept", "accepted", "одобрить", "одобрено":
		updates["currency_control_decision"] = "accepted"
		updates["status"] = domain.ExtensionStatusPendingCompliance
	case "revision", "на доработку", "доработка":
		if strings.TrimSpace(comment) == "" {
			return nil, fmt.Errorf("при отправке на доработку комментарий с замечаниями обязателен")
		}
		updates["currency_control_decision"] = "revision"
		updates["status"] = domain.ExtensionStatusRevisionRequired
		updates["rejection_reason"] = comment
	case "reject", "rejected", "отклонить", "отказ":
		if strings.TrimSpace(comment) == "" {
			return nil, fmt.Errorf("при отклонении причина отказа обязательна")
		}
		updates["currency_control_decision"] = "rejected"
		updates["status"] = domain.ExtensionStatusRejectedCurrencyControl
		updates["rejection_reason"] = comment
	default:
		return nil, fmt.Errorf("недопустимое решение валютного контроля: %s (допустимы: accept, revision, reject)", decision)
	}

	if err := r.db.WithContext(ctx).Model(&req).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("ошибка сохранения решения валютного контроля: %w", err)
	}

	return r.GetByID(ctx, id)
}

func (r *gtdExtensionRepo) SetComplianceDecision(ctx context.Context, id int64, decision, comment, reviewer string) (*domain.GTDExtensionRequest, error) {
	var req domain.GTDExtensionRequest
	if err := r.db.WithContext(ctx).First(&req, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("заявка на продление не найдена")
		}
		return nil, err
	}

	if req.Status != domain.ExtensionStatusPendingCompliance {
		return nil, fmt.Errorf("заявка не ожидает рассмотрения комплаенс-контролем (текущий статус: %s)", req.Status)
	}

	now := time.Now()
	normDecision := strings.ToLower(strings.TrimSpace(decision))
	updates := map[string]interface{}{
		"compliance_reviewed_by": reviewer,
		"compliance_reviewed_at": &now,
		"compliance_comment":     comment,
		"reviewed_by":            &reviewer,
		"reviewed_at":            &now,
		"comment":                comment,
		"updated_at":             now,
	}

	switch normDecision {
	case "approve", "approved", "одобрить", "одобрено", "accept", "accepted":
		updates["compliance_decision"] = "approved"
		updates["status"] = domain.ExtensionStatusApproved

		// Применяем продленный срок к ГТД только после финального одобрения Комплаенсом
		targetDeadline := req.RequestedDeadline
		var gtd domain.GTD
		if err := r.db.WithContext(ctx).First(&gtd, req.GTDID).Error; err == nil {
			actualDate := now
			if gtd.SubmissionDate != nil && !gtd.SubmissionDate.IsZero() {
				actualDate = *gtd.SubmissionDate
			}
			diffDays, status, notice := domain.CalculateDeliveryComparison(gtd.DocumentType, actualDate, &targetDeadline)
			noticeWithNotice := fmt.Sprintf("Срок продлен до %s. %s", targetDeadline.Format("02.01.2006"), notice)

			gtdUpdates := map[string]interface{}{
				"delivery_deadline": targetDeadline,
				"days_difference":   diffDays,
				"delivery_status":   status,
				"delivery_notice":   noticeWithNotice,
			}
			_ = r.db.WithContext(ctx).Model(&domain.GTD{}).Where("id = ?", req.GTDID).Updates(gtdUpdates)
		}

	case "reject", "rejected", "отклонить", "отказ":
		if strings.TrimSpace(comment) == "" {
			return nil, fmt.Errorf("при отклонении причина отказа обязательна")
		}
		updates["compliance_decision"] = "rejected"
		updates["status"] = domain.ExtensionStatusRejectedCompliance
		updates["rejection_reason"] = comment
	default:
		return nil, fmt.Errorf("недопустимое решение комплаенс-контроля: %s (допустимы: approve, reject)", decision)
	}

	if err := r.db.WithContext(ctx).Model(&req).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("ошибка сохранения решения комплаенс-контроля: %w", err)
	}

	return r.GetByID(ctx, id)
}

func (r *gtdExtensionRepo) ReviewRequest(ctx context.Context, id int64, decision, comment, reviewer string) (*domain.GTDExtensionRequest, error) {
	var req domain.GTDExtensionRequest
	if err := r.db.WithContext(ctx).First(&req, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("заявка на продление не найдена")
		}
		return nil, err
	}

	if req.Status == domain.ExtensionStatusPendingCompliance {
		return r.SetComplianceDecision(ctx, id, decision, comment, reviewer)
	}
	return r.SetCurrencyControlDecision(ctx, id, decision, comment, reviewer)
}
