package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"

	"gorm.io/gorm"
)

type actRepo struct {
	db *gorm.DB
}

func NewActRepository(db *gorm.DB) ports.ActRepository {
	return &actRepo{db: db}
}

func (r *actRepo) Create(ctx context.Context, act domain.Act) (domain.Act, error) {
	var inv domain.Invoice
	if err := r.db.WithContext(ctx).First(&inv, act.InvoiceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Act{}, fmt.Errorf("инвойс не найден")
		}
		return domain.Act{}, err
	}

	act.ContractID = inv.ContractID
	if act.Currency == "" {
		act.Currency = inv.Currency
	}

	if err := r.db.WithContext(ctx).Create(&act).Error; err != nil {
		return domain.Act{}, err
	}

	enrichAct(ctx, r.db, &act)
	return act, nil
}

func (r *actRepo) GetByID(ctx context.Context, id int64) (*domain.Act, error) {
	var act domain.Act
	err := r.db.WithContext(ctx).Where("id = ? AND deleted_at IS NULL", id).First(&act).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	enrichAct(ctx, r.db, &act)
	return &act, nil
}

func (r *actRepo) GetByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.Act, error) {
	var list []domain.Act
	err := r.db.WithContext(ctx).
		Where("invoice_id = ? AND deleted_at IS NULL", invoiceID).
		Order("act_date DESC, id DESC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	enrichActs(ctx, r.db, list)
	return list, nil
}

func (r *actRepo) GetByContractID(ctx context.Context, contractID int64) ([]domain.Act, error) {
	var list []domain.Act
	err := r.db.WithContext(ctx).
		Where("contract_id = ? AND deleted_at IS NULL", contractID).
		Order("act_date DESC, id DESC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	enrichActs(ctx, r.db, list)
	return list, nil
}

func (r *actRepo) Update(ctx context.Context, id int64, act domain.Act) (domain.Act, error) {
	updates := map[string]interface{}{
		"act_number": act.ActNumber,
		"act_date":   act.ActDate,
		"act_amount": act.ActAmount,
		"updated_by": act.UpdatedBy,
		"updated_at": time.Now(),
	}
	if act.DocumentPath != nil {
		updates["document_path"] = act.DocumentPath
	}

	if err := r.db.WithContext(ctx).Model(&domain.Act{}).Where("id = ? AND deleted_at IS NULL", id).Updates(updates).Error; err != nil {
		return domain.Act{}, err
	}

	var updated domain.Act
	if err := r.db.WithContext(ctx).First(&updated, id).Error; err != nil {
		return domain.Act{}, err
	}
	enrichAct(ctx, r.db, &updated)
	return updated, nil
}

func (r *actRepo) SoftDelete(ctx context.Context, id int64) error {
	return r.db.WithContext(ctx).Where("id = ?", id).Delete(&domain.Act{}).Error
}
