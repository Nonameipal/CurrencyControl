package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"CurrencyControl/internal/delivery/dto"
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

func (s *permissionService) SearchCurrencyControlUsers(ctx context.Context, query string) ([]dto.CurrencyControlUserItem, error) {
	users, err := s.repo.SearchCurrencyControlUsers(ctx, query)
	if err != nil {
		return nil, err
	}

	res := make([]dto.CurrencyControlUserItem, 0, len(users))
	for _, u := range users {
		res = append(res, dto.CurrencyControlUserItem{
			ID:         u.ID,
			Login:      u.Login,
			FirstName:  u.FirstName,
			LastName:   u.LastName,
			Email:      u.Email,
			Role:       u.Role,
			RoleName:   domain.FormatRoleName(u.Role),
			BranchID:   u.BranchID,
			BranchName: u.BranchName,
		})
	}
	return res, nil
}

func (s *permissionService) CanCreateFiles(ctx context.Context, role, login string) bool {
	if role == domain.RoleOperator || role == domain.RoleAdmin || role == domain.RoleCompliance {
		return true
	}
	if role == domain.RoleCurrencyControl || role == domain.RoleCurrencyController {
		allowed, err := s.repo.CheckCurrencyControlPermission(ctx, login, "create")
		return err == nil && allowed
	}
	return false
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
	login := strings.TrimSpace(perm.Login)
	if login == "" {
		return fmt.Errorf("логин пользователя обязателен")
	}

	user, err := s.repo.GetUserByLogin(ctx, login)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("пользователь с логином '%s' не найден в системе", login)
	}

	if user.Role != domain.RoleCurrencyControl && user.Role != domain.RoleCurrencyController {
		return fmt.Errorf("недопустимая роль пользователя '%s' (%s): права могут быть назначены только сотрудникам валютного контроля ('%s' или '%s')",
			user.Login, domain.FormatRoleName(user.Role), domain.FormatRoleName(domain.RoleCurrencyControl), domain.FormatRoleName(domain.RoleCurrencyController))
	}

	perm.Login = user.Login
	return s.repo.GrantCurrencyControlPermission(ctx, perm)
}

func (s *permissionService) UpdateCurrencyControlPermission(ctx context.Context, login string, canCreate, canEdit, canDelete *bool, granterLogin string) (*domain.CurrencyControlPermission, error) {
	cleanLogin := strings.TrimSpace(login)
	if cleanLogin == "" {
		return nil, fmt.Errorf("логин пользователя обязателен")
	}

	user, err := s.repo.GetUserByLogin(ctx, cleanLogin)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("пользователь с логином '%s' не найден в системе", cleanLogin)
	}

	if user.Role != domain.RoleCurrencyControl && user.Role != domain.RoleCurrencyController {
		return nil, fmt.Errorf("недопустимая роль пользователя '%s' (%s): права могут быть назначены только сотрудникам валютного контроля ('%s' или '%s')",
			user.Login, domain.FormatRoleName(user.Role), domain.FormatRoleName(domain.RoleCurrencyControl), domain.FormatRoleName(domain.RoleCurrencyController))
	}

	existing, err := s.repo.GetCurrencyControlPermissionByLogin(ctx, user.Login)
	if err != nil {
		return nil, err
	}

	perm := domain.CurrencyControlPermission{
		Login:     user.Login,
		CanCreate: false,
		CanEdit:   false,
		CanDelete: false,
		GrantedBy: granterLogin,
		GrantedAt: time.Now(),
	}

	if existing != nil {
		perm = *existing
		perm.GrantedBy = granterLogin
		perm.GrantedAt = time.Now()
	}

	if canCreate != nil {
		perm.CanCreate = *canCreate
	}
	if canEdit != nil {
		perm.CanEdit = *canEdit
	}
	if canDelete != nil {
		perm.CanDelete = *canDelete
	}

	if err := s.repo.GrantCurrencyControlPermission(ctx, perm); err != nil {
		return nil, err
	}

	return &perm, nil
}

func (s *permissionService) RevokeCurrencyControlPermission(ctx context.Context, login string) error {
	cleanLogin := strings.TrimSpace(login)
	if cleanLogin == "" {
		return fmt.Errorf("логин пользователя обязателен")
	}

	user, err := s.repo.GetUserByLogin(ctx, cleanLogin)
	if err != nil {
		return err
	}
	if user == nil {
		return fmt.Errorf("пользователь с логином '%s' не найден в системе", cleanLogin)
	}

	return s.repo.RevokeCurrencyControlPermission(ctx, user.Login)
}
