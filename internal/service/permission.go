package service

import (
	"context"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"
)

type permissionService struct {
	repo ports.PermissionRepository
}

func NewPermissionService(repo ports.PermissionRepository) ports.PermissionService {
	return &permissionService{repo: repo}
}

func (s *permissionService) GetCurrencyControlPermissions(ctx context.Context) ([]domain.CurrencyControlPermission, error) {
	return s.repo.GetCurrencyControlPermissions(ctx)
}

func (s *permissionService) CanEditFiles(ctx context.Context, role, login string) bool {
	if role == domain.RoleAdmin || role == domain.RoleCompliance {
		return true
	}
	if role == domain.RoleCurrencyControl || role == domain.RoleCurrencyController {
		allowed, err := s.repo.CheckCurrencyControlPermission(ctx, login, "edit")
		return err == nil && allowed
	}
	return false
}

func (s *permissionService) CanDeleteFiles(ctx context.Context, role, login string) bool {
	if role == domain.RoleAdmin || role == domain.RoleCompliance {
		return true
	}
	if role == domain.RoleCurrencyControl || role == domain.RoleCurrencyController {
		allowed, err := s.repo.CheckCurrencyControlPermission(ctx, login, "delete")
		return err == nil && allowed
	}
	return false
}

func (s *permissionService) GrantCurrencyControlPermission(ctx context.Context, perm domain.CurrencyControlPermission) error {
	return s.repo.GrantCurrencyControlPermission(ctx, perm)
}

func (s *permissionService) RevokeCurrencyControlPermission(ctx context.Context, login string) error {
	return s.repo.RevokeCurrencyControlPermission(ctx, login)
}
