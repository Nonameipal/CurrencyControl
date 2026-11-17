package repository

import (
	"context"
	"errors"
	"strings"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type permissionRepo struct {
	db *gorm.DB
}

func NewPermissionRepository(db *gorm.DB) ports.PermissionRepository {
	return &permissionRepo{db: db}
}

func (r *permissionRepo) GetUserByLogin(ctx context.Context, login string) (*domain.User, error) {
	cleanLogin := strings.ToLower(strings.TrimSpace(login))
	if cleanLogin == "" {
		return nil, nil
	}
	var u domain.User
	if err := r.db.WithContext(ctx).Where("LOWER(TRIM(login)) = ?", cleanLogin).First(&u).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &u, nil
}

func (r *permissionRepo) GetCurrencyControlPermissions(ctx context.Context) ([]domain.CurrencyControlPermission, error) {
	var perms []domain.CurrencyControlPermission
	if err := r.db.WithContext(ctx).Order("granted_at DESC").Find(&perms).Error; err != nil {
		return nil, err
	}
	enrichPermissions(ctx, r.db, perms)
	return perms, nil
}

func (r *permissionRepo) GetCurrencyControlPermissionByLogin(ctx context.Context, login string) (*domain.CurrencyControlPermission, error) {
	cleanLogin := strings.ToLower(strings.TrimSpace(login))
	if cleanLogin == "" {
		return nil, nil
	}
	var perm domain.CurrencyControlPermission
	if err := r.db.WithContext(ctx).Where("LOWER(TRIM(login)) = ?", cleanLogin).First(&perm).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &perm, nil
}

func (r *permissionRepo) SearchCurrencyControlUsers(ctx context.Context, query string) ([]domain.User, error) {
	var users []domain.User
	cleanQuery := strings.ToLower(strings.TrimSpace(query))

	q := r.db.WithContext(ctx).Table("users u").
		Select("u.id, u.login, u.last_name, u.first_name, u.email, u.role, u.branch_id, COALESCE(b.name, '') as branch_name, u.created_at").
		Joins("LEFT JOIN branches b ON b.id = u.branch_id").
		Where("u.role IN (?, ?)", domain.RoleCurrencyControl, domain.RoleCurrencyController)

	if cleanQuery != "" {
		likePattern := "%" + cleanQuery + "%"
		q = q.Where("LOWER(u.login) LIKE ? OR LOWER(u.first_name) LIKE ? OR LOWER(u.last_name) LIKE ?", likePattern, likePattern, likePattern)
	}

	if err := q.Order("u.login ASC").Limit(50).Scan(&users).Error; err != nil {
		return nil, err
	}
	if users == nil {
		users = []domain.User{}
	}
	return users, nil
}

func (r *permissionRepo) CheckCurrencyControlPermission(ctx context.Context, login string, action string) (bool, error) {
	cleanLogin := strings.ToLower(strings.TrimSpace(login))
	if cleanLogin == "" {
		return false, nil
	}

	var perm domain.CurrencyControlPermission
	err := r.db.WithContext(ctx).Where("LOWER(TRIM(login)) = ?", cleanLogin).First(&perm).Error
	if err != nil {
		return false, nil
	}

	if action == "create" {
		return perm.CanCreate, nil
	}
	if action == "edit" {
		return perm.CanEdit, nil
	}
	if action == "delete" {
		return perm.CanDelete, nil
	}
	return false, nil
}

func (r *permissionRepo) GrantCurrencyControlPermission(ctx context.Context, perm domain.CurrencyControlPermission) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "login"}},
		DoUpdates: clause.AssignmentColumns([]string{"can_create", "can_edit", "can_delete", "granted_by", "granted_at"}),
	}).Create(&perm).Error
}

func (r *permissionRepo) RevokeCurrencyControlPermission(ctx context.Context, login string) error {
	cleanLogin := strings.TrimSpace(login)
	return r.db.WithContext(ctx).Where("LOWER(TRIM(login)) = LOWER(TRIM(?))", cleanLogin).Delete(&domain.CurrencyControlPermission{}).Error
}
