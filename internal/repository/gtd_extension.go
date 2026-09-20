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

	var pendingCount int64
	err := r.db.WithContext(ctx).Model(&domain.GTDExtensionRequest{}).
		Where("gtd_id = ? AND status = ?", req.GTDID, domain.ExtensionStatusPending).
		Count(&pendingCount).Error
	if err != nil {
		return domain.GTDExtensionRequest{}, fmt.Errorf("ошибка проверки существующих заявок: %w", err)
	}
	if pendingCount > 0 {
		return domain.GTDExtensionRequest{}, fmt.Errorf("по данной ГТД уже есть активная заявка на увеличение срока, ожидающая рассмотрения валютным контролем")
	}

	req.ContractID = gtd.ContractID
	req.InvoiceID = gtd.InvoiceID
	req.CurrentDeadline = gtd.DeliveryDeadline
	req.Status = domain.ExtensionStatusPending

	if err := r.db.WithContext(ctx).Create(&req).Error; err != nil {
		return domain.GTDExtensionRequest{}, fmt.Errorf("ошибка создания заявки на продление ГТД: %w", err)
	}
	return req, nil
}

func (r *gtdExtensionRepo) GetByID(ctx context.Context, id int64) (*domain.GTDExtensionRequest, error) {
	var req domain.GTDExtensionRequest
	if err := r.db.WithContext(ctx).First(&req, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("заявка на продление не найдена")
		}
		return nil, err
	}
	return &req, nil
}

func (r *gtdExtensionRepo) GetByGTDID(ctx context.Context, gtdID int64) ([]domain.GTDExtensionRequest, error) {
	var list []domain.GTDExtensionRequest
	if err := r.db.WithContext(ctx).
		Where("gtd_id = ?", gtdID).
		Order("created_at DESC").
		Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

func (r *gtdExtensionRepo) GetPendingRequests(ctx context.Context, branchID *int, page, pageSize int) ([]domain.GTDExtensionRequest, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	tx := r.db.WithContext(ctx).Model(&domain.GTDExtensionRequest{}).
		Joins("JOIN contracts c ON c.id = gtd_extension_requests.contract_id").
		Where("gtd_extension_requests.status = ?", domain.ExtensionStatusPending)

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
	return list, int(total), nil
}

func (r *gtdExtensionRepo) ReviewRequest(ctx context.Context, id int64, decision string, approvedDeadline *time.Time, comment, reviewer string) (*domain.GTDExtensionRequest, error) {
	var req domain.GTDExtensionRequest
	if err := r.db.WithContext(ctx).First(&req, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("заявка на продление не найдена")
		}
		return nil, err
	}

	if req.Status != domain.ExtensionStatusPending {
		return nil, fmt.Errorf("заявка уже рассмотрена (текущий статус: %s)", req.Status)
	}

	normDecision := strings.ToLower(strings.TrimSpace(decision))
	switch normDecision {
	case "approve", "approved", "одобрить", "одобрено":
		targetDeadline := req.RequestedDeadline
		if approvedDeadline != nil && !approvedDeadline.IsZero() {
			targetDeadline = *approvedDeadline
		}

		now := time.Now()
		updates := map[string]interface{}{
			"status":            domain.ExtensionStatusApproved,
			"approved_deadline": targetDeadline,
			"reviewed_by":       reviewer,
			"reviewed_at":       &now,
			"comment":           comment,
		}
		if err := r.db.WithContext(ctx).Model(&req).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("ошибка утверждения заявки: %w", err)
		}

		var gtd domain.GTD
		if err := r.db.WithContext(ctx).First(&gtd, req.GTDID).Error; err == nil {
			actualDate := now
			if gtd.SubmissionDate != nil && !gtd.SubmissionDate.IsZero() {
				actualDate = *gtd.SubmissionDate
			}
			diffDays, status, notice := domain.CalculateDeliveryComparison(gtd.DocumentType, actualDate, &targetDeadline)
			noticeWithNotice := fmt.Sprintf("Срок продлен Валютным контролем до %s. %s", targetDeadline.Format("02.01.2006"), notice)

			gtdUpdates := map[string]interface{}{
				"delivery_deadline": targetDeadline,
				"days_difference":   diffDays,
				"delivery_status":   status,
				"delivery_notice":   noticeWithNotice,
			}
			_ = r.db.WithContext(ctx).Model(&domain.GTD{}).Where("id = ?", req.GTDID).Updates(gtdUpdates)
		}

		return r.GetByID(ctx, id)

	case "reject", "rejected", "отклонить", "отказ":
		if strings.TrimSpace(comment) == "" {
			return nil, fmt.Errorf("причина отказа обязательна для заполнения")
		}

		now := time.Now()
		updates := map[string]interface{}{
			"status":      domain.ExtensionStatusRejected,
			"reviewed_by": reviewer,
			"reviewed_at": &now,
			"comment":     comment,
		}
		if err := r.db.WithContext(ctx).Model(&req).Updates(updates).Error; err != nil {
			return nil, fmt.Errorf("ошибка отклонения заявки: %w", err)
		}

		return r.GetByID(ctx, id)

	default:
		return nil, fmt.Errorf("недопустимое решение: %s (допустимы: approve, reject)", decision)
	}
}
