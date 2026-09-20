package repository

import (
	"context"
	"strings"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"

	"gorm.io/gorm"
)

type auditLogRepo struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) ports.AuditLogRepository {
	return &auditLogRepo{db: db}
}

func (r *auditLogRepo) Create(ctx context.Context, log domain.AuditLog) error {
	return r.db.WithContext(ctx).Create(&log).Error
}

func (r *auditLogRepo) List(ctx context.Context, filter ports.AuditLogFilter) ([]domain.AuditLog, int64, error) {
	tx := r.db.WithContext(ctx).Model(&domain.AuditLog{})

	if userLogin := strings.TrimSpace(filter.UserLogin); userLogin != "" {
		tx = tx.Where("user_login ILIKE ?", "%"+userLogin+"%")
	}
	if action := strings.TrimSpace(filter.Action); action != "" {
		tx = tx.Where("action = ?", action)
	}
	if entity := strings.TrimSpace(filter.Entity); entity != "" {
		tx = tx.Where("entity = ?", entity)
	}
	if filter.BranchID != nil && *filter.BranchID > 0 {
		tx = tx.Where("branch_id = ?", *filter.BranchID)
	}
	if filter.FromDate != nil {
		tx = tx.Where("created_at >= ?", *filter.FromDate)
	}
	if filter.ToDate != nil {
		tx = tx.Where("created_at <= ?", *filter.ToDate)
	}

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	limit := filter.Limit
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	var logs []domain.AuditLog
	if err := tx.Order("created_at DESC").Limit(limit).Offset(offset).Find(&logs).Error; err != nil {
		return nil, 0, err
	}
	return logs, total, nil
}
