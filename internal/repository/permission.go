package repository

import (
	"context"

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

func (r *permissionRepo) GetCurrencyControlPermissions(ctx context.Context) ([]domain.CurrencyControlPermission, error) {
	var perms []domain.CurrencyControlPermission
	if err := r.db.WithContext(ctx).Order("granted_at DESC").Find(&perms).Error; err != nil {
		return nil, err
	}
	return perms, nil
}

func (r *permissionRepo) CheckCurrencyControlPermission(ctx context.Context, login string, action string) (bool, error) {
	var perm domain.CurrencyControlPermission
	err := r.db.WithContext(ctx).Where("login = ?", login).First(&perm).Error
	if err != nil {
		err = r.db.WithContext(ctx).Where("login = ?", "*").First(&perm).Error
	}
	if err != nil {
		return false, nil
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
		DoUpdates: clause.AssignmentColumns([]string{"can_edit", "can_delete", "granted_by", "granted_at"}),
	}).Create(&perm).Error
}

func (r *permissionRepo) RevokeCurrencyControlPermission(ctx context.Context, login string) error {
	return r.db.WithContext(ctx).Where("login = ?", login).Delete(&domain.CurrencyControlPermission{}).Error
}
