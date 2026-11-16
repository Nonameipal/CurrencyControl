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

func NewBranchRepository(db *gorm.DB) ports.BranchRepository {
	return &branchRepo{db: db}
}

type branchRepo struct {
	db *gorm.DB
}

func (r *branchRepo) Create(ctx context.Context, b domain.Branch) (domain.Branch, error) {
	cleanName := strings.TrimSpace(b.Name)
	b.Name = cleanName
	_ = r.db.WithContext(ctx).Unscoped().
		Where("id != ? AND LOWER(TRIM(name)) = LOWER(?) AND deleted_at IS NOT NULL", b.ID, cleanName).
		Delete(&domain.Branch{}).Error

	var existing domain.Branch
	err := r.db.WithContext(ctx).Unscoped().Where("id = ?", b.ID).First(&existing).Error
	if err == nil {
		if !existing.DeletedAt.Valid {
			return domain.Branch{}, fmt.Errorf("филиал с кодом %d уже существует", b.ID)
		}
		now := time.Now()
		updates := map[string]interface{}{
			"name":       b.Name,
			"created_by": b.CreatedBy,
			"created_at": now,
			"updated_at": now,
			"deleted_at": nil,
			"deleted_by": nil,
			"updated_by": "",
		}
		if err := r.db.WithContext(ctx).Unscoped().Model(&domain.Branch{}).Where("id = ?", b.ID).Updates(updates).Error; err != nil {
			return domain.Branch{}, err
		}
		b.CreatedAt = now
		b.UpdatedAt = now
		b.DeletedAt = gorm.DeletedAt{}
		b.DeletedBy = nil
		b.UpdatedBy = ""
		return b, nil
	}

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
	enrichBranch(ctx, r.db, &res)
	return &res, nil
}

func (r *branchRepo) GetAll(ctx context.Context) ([]domain.Branch, error) {
	var branches []domain.Branch
	if err := r.db.WithContext(ctx).Order("id ASC").Find(&branches).Error; err != nil {
		return nil, err
	}
	enrichBranches(ctx, r.db, branches)
	return branches, nil
}

func (r *branchRepo) Update(ctx context.Context, id int, name string) (*domain.Branch, error) {
	cleanName := strings.TrimSpace(name)
	var res domain.Branch
	if err := r.db.WithContext(ctx).First(&res, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("филиал не найден")
		}
		return nil, err
	}

	// Очищаем старую soft-deleted запись с таким же именем (если была у другого ID),
	// чтобы не нарушать UNIQUE(name):
	_ = r.db.WithContext(ctx).Unscoped().
		Where("id != ? AND LOWER(TRIM(name)) = LOWER(?) AND deleted_at IS NOT NULL", id, cleanName).
		Delete(&domain.Branch{}).Error

	updates := map[string]interface{}{"name": cleanName}
	if userLogin := domain.GetLoginFromCtx(ctx); userLogin != "" {
		updates["updated_by"] = userLogin
	}
	if err := r.db.WithContext(ctx).Model(&res).Updates(updates).Error; err != nil {
		return nil, err
	}
	res.Name = cleanName
	if userLogin := domain.GetLoginFromCtx(ctx); userLogin != "" {
		res.UpdatedBy = userLogin
	}
	enrichBranch(ctx, r.db, &res)
	return &res, nil
}

func (r *branchRepo) SoftDelete(ctx context.Context, id int) error {
	if userLogin := domain.GetLoginFromCtx(ctx); userLogin != "" {
		_ = r.db.WithContext(ctx).Model(&domain.Branch{}).Where("id = ?", id).Update("deleted_by", userLogin).Error
	}

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

func (r *branchRepo) CheckNameExists(ctx context.Context, name string, excludeID int) (bool, error) {
	var count int64
	q := r.db.WithContext(ctx).Model(&domain.Branch{}).Where("LOWER(TRIM(name)) = LOWER(?)", strings.TrimSpace(name))
	if excludeID > 0 {
		q = q.Where("id != ?", excludeID)
	}
	err := q.Count(&count).Error
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
	if err := r.db.WithContext(ctx).Model(&domain.Contract{}).Where("branch_id = ?", id).Count(&count).Error; err == nil && count > 0 {
		return true, nil
	}
	if err := r.db.WithContext(ctx).Model(&domain.AccessRequest{}).Where("branch_id = ? AND status = ?", id, "pending").Count(&count).Error; err == nil && count > 0 {
		return true, nil
	}
	return false, nil
}
