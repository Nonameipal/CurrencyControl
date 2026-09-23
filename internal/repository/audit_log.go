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
	tx := r.db.WithContext(ctx).Table("audit_logs al")

	if userLogin := strings.TrimSpace(filter.UserLogin); userLogin != "" {
		tx = tx.Where("al.user_login ILIKE ?", "%"+userLogin+"%")
	}
	if action := strings.TrimSpace(filter.Action); action != "" {
		tx = tx.Where("al.action = ?", action)
	}
	if entity := strings.TrimSpace(filter.Entity); entity != "" {
		tx = tx.Where("al.entity = ?", entity)
	}
	if filter.BranchID != nil && *filter.BranchID > 0 {
		tx = tx.Where("al.branch_id = ?", *filter.BranchID)
	}
	if filter.FromDate != nil {
		tx = tx.Where("al.created_at >= ?", *filter.FromDate)
	}
	if filter.ToDate != nil {
		tx = tx.Where("al.created_at <= ?", *filter.ToDate)
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

	var rawLogs []struct {
		domain.AuditLog
		DbLastName  string `gorm:"column:db_last_name"`
		DbFirstName string `gorm:"column:db_first_name"`
		DbEmail     string `gorm:"column:db_email"`
	}
	err := tx.Select("al.*, COALESCE(u.last_name, '') AS db_last_name, COALESCE(u.first_name, '') AS db_first_name, COALESCE(u.email, '') AS db_email").
		Joins("LEFT JOIN users u ON u.login = al.user_login").
		Order("al.created_at DESC").
		Limit(limit).Offset(offset).
		Scan(&rawLogs).Error
	if err != nil {
		return nil, 0, err
	}

	logs := make([]domain.AuditLog, len(rawLogs))
	for i, rl := range rawLogs {
		logs[i] = rl.AuditLog
		if logs[i].UserLastName == "" {
			logs[i].UserLastName = rl.DbLastName
		}
		if logs[i].UserFirstName == "" {
			logs[i].UserFirstName = rl.DbFirstName
		}
		if logs[i].UserEmail == "" {
			logs[i].UserEmail = rl.DbEmail
		}
		logs[i].User = &domain.UserBrief{
			Login:     logs[i].UserLogin,
			FirstName: logs[i].UserFirstName,
			LastName:  logs[i].UserLastName,
			Email:     logs[i].UserEmail,
		}
	}
	return logs, total, nil
}
