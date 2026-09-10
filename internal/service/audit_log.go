package service

import (
	"context"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/logger"
	"CurrencyControl/internal/repository"
)

type AuditLogService interface {
	Log(ctx context.Context, login, role string, branchID *int64, action, entity string, entityID *int64, details, ip string)
	List(ctx context.Context, filter repository.AuditLogFilter) ([]domain.AuditLog, int64, error)
}

type auditLogService struct {
	repo repository.AuditLogRepository
}

func NewAuditLogService(repo repository.AuditLogRepository) AuditLogService {
	return &auditLogService{repo: repo}
}

func (s *auditLogService) Log(ctx context.Context, login, role string, branchID *int64, action, entity string, entityID *int64, details, ip string) {
	if s == nil || s.repo == nil {
		return
	}
	go func() {
		defer func() {
			if r := recover(); r != nil {
				logger.Error(nil, "panic in auditLogService.Log: %v", r)
			}
		}()
		bgCtx := context.Background()
		entry := domain.AuditLog{
			UserLogin: login,
			Role:      role,
			BranchID:  branchID,
			Action:    action,
			Entity:    entity,
			EntityID:  entityID,
			Details:   details,
			IPAddress: ip,
		}
		if err := s.repo.Create(bgCtx, entry); err != nil {
			logger.Error(err, "failed to record audit log: action=%s user=%s", action, login)
		}
	}()
}

func (s *auditLogService) List(ctx context.Context, filter repository.AuditLogFilter) ([]domain.AuditLog, int64, error) {
	return s.repo.List(ctx, filter)
}
