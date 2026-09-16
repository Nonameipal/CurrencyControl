package repository

import (
	"context"
	"errors"
	"fmt"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"

	"gorm.io/gorm"
)

func NewBranchRepository(db *gorm.DB) ports.BranchRepository {
	return &branchRepo{db: db}
}

type branchRepo struct {
	db *gorm.DB
}

func (r *branchRepo) Create(ctx context.Context, b domain.Branch) (domain.Branch, error) {
	if err := r.db.WithContext(ctx).Create(&b).Error; err != nil {
		return domain.Branch{}, err
	}
	return b, nil
}

func (r *branchRepo) GetByID(ctx context.Context, id int) (*domain.Branch, error) {
	var res domain.Branch
	if err := r.db.WithContext(ctx).First(&res, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("филиал не найден")
		}
		return nil, err
	}
	return &res, nil
}

func (r *branchRepo) GetAll(ctx context.Context) ([]domain.Branch, error) {
	var branches []domain.Branch
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&branches).Error; err != nil {
		return nil, err
	}
	if branches == nil {
		branches = []domain.Branch{}
	}
	return branches, nil
}

func (r *branchRepo) Update(ctx context.Context, id int, name string) (*domain.Branch, error) {
	var res domain.Branch
	if err := r.db.WithContext(ctx).First(&res, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("филиал не найден")
		}
		return nil, err
	}
	if err := r.db.WithContext(ctx).Model(&res).Update("name", name).Error; err != nil {
		return nil, err
	}
	res.Name = name
	return &res, nil
}

func (r *branchRepo) SoftDelete(ctx context.Context, id int) error {
	result := r.db.WithContext(ctx).Delete(&domain.Branch{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("филиал не найден или уже удален")
	}
	return nil
}

func (r *branchRepo) CheckExists(ctx context.Context, id int) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Branch{}).Where("id = ?", id).Count(&count).Error
	return count > 0, err
}

func (r *branchRepo) CheckHasRelations(ctx context.Context, id int) (bool, error) {
	var count int64
	if err := r.db.WithContext(ctx).Model(&domain.User{}).Where("branch_id = ?", id).Count(&count).Error; err == nil && count > 0 {
		return true, nil
	}
	if err := r.db.WithContext(ctx).Model(&domain.Counterparty{}).Where("branch_id = ?", id).Count(&count).Error; err == nil && count > 0 {
		return true, nil
	}
	if err := r.db.WithContext(ctx).Model(&domain.AccessRequest{}).Where("branch_id = ? AND status = ?", id, "pending").Count(&count).Error; err == nil && count > 0 {
		return true, nil
	}
	return false, nil
}
