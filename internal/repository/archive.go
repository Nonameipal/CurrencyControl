package repository

import (
	"context"
	"time"

	"CurrencyControl/internal/domain"

	"gorm.io/gorm"
)

const unclosedInvoiceCondition = `(
	i.amount > COALESCE((SELECT SUM(g.closes_amount) FROM gtd g WHERE g.invoice_id = i.id AND g.deleted_at IS NULL), 0)
	OR
	i.amount > COALESCE((SELECT SUM(po.amount) FROM payment_orders po WHERE po.invoice_id = i.id AND po.deleted_at IS NULL), 0)
)`

func syncContractBalances(ctx context.Context, db *gorm.DB, contractID int64) error {
	var c domain.Contract
	if err := db.WithContext(ctx).First(&c, contractID).Error; err != nil {
		return err
	}
	if c.OriginalAmount == 0 && c.TotalAmount > 0 {
		c.OriginalAmount = c.TotalAmount
		_ = db.WithContext(ctx).Model(&domain.Contract{}).Where("id = ?", contractID).Update("original_amount", c.OriginalAmount).Error
	}

	var sameCurrencyAddl float64
	db.WithContext(ctx).Model(&domain.AdditionalAgreement{}).
		Where("contract_id = ? AND deleted_at IS NULL AND status = 'active' AND currency = ?", contractID, c.ContractCurrency).
		Select("COALESCE(SUM(COALESCE(foreign_amount, 0)), 0)").
		Scan(&sameCurrencyAddl)

	totalAmount := c.OriginalAmount + sameCurrencyAddl

	var usedDeduct float64
	db.WithContext(ctx).Model(&domain.Invoice{}).
		Where("contract_id = ? AND deleted_at IS NULL AND currency = ?", contractID, c.ContractCurrency).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&usedDeduct)

	remainingAmount := totalAmount - usedDeduct

	return db.WithContext(ctx).Model(&domain.Contract{}).Where("id = ?", contractID).Updates(map[string]interface{}{
		"total_amount":     totalAmount,
		"remaining_amount": remainingAmount,
	}).Error
}

func tryArchiveContractGorm(ctx context.Context, db *gorm.DB, contractID int64) {
	var contract domain.Contract
	if err := db.WithContext(ctx).
		Select("remaining_amount, status").
		Where("id = ? AND status = 'active'", contractID).
		First(&contract).Error; err != nil {
		return
	}

	if contract.RemainingAmount > 0 {
		return
	}

	var unclosedContractInvoices int64
	_ = db.WithContext(ctx).Table("invoices i").
		Where("i.contract_id = ? AND i.deleted_at IS NULL AND "+unclosedInvoiceCondition, contractID).
		Count(&unclosedContractInvoices).Error

	if unclosedContractInvoices > 0 {
		return
	}

	now := time.Now()
	_ = db.WithContext(ctx).Model(&domain.Contract{}).
		Where("id = ?", contractID).
		Updates(map[string]interface{}{
			"status":      "archived",
			"archived_at": &now,
		})
}

func tryArchiveEntityGorm(ctx context.Context, db *gorm.DB, contractID int64) {
	if contractID > 0 {
		tryArchiveContractGorm(ctx, db, contractID)
	}
}

func propagateAADatesToContractGorm(ctx context.Context, db *gorm.DB, contractID int64, deliveryDate, extendDateTo *time.Time, returnDays *int) {
	updates := map[string]interface{}{}
	if deliveryDate != nil && !deliveryDate.IsZero() {
		updates["delivery_date"] = *deliveryDate
	}
	if extendDateTo != nil && !extendDateTo.IsZero() {
		updates["extend_date_to"] = *extendDateTo
	}
	if returnDays != nil && *returnDays > 0 {
		updates["return_days"] = *returnDays
	}
	if len(updates) > 0 {
		db.WithContext(ctx).Model(&domain.Contract{}).Where("id = ?", contractID).Updates(updates)
	}
}	
func (r *contractRepo) GetArchived(ctx context.Context, branchID int, page, pageSize int) ([]domain.Contract, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	tx := r.db.WithContext(ctx).Table("contracts c").
		Joins("JOIN counterparties cp ON cp.id = c.client_id").
		Where("(cp.branch_id = ? OR c.branch_id = ?) AND c.deleted_at IS NULL AND c.status = 'archived'", branchID, branchID)

	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var list []domain.Contract
	if err := tx.Select("c.*").
		Order("c.archived_at DESC").
		Limit(pageSize).Offset(offset).
		Find(&list).Error; err != nil {
		return nil, 0, err
	}
	enrichContracts(ctx, r.db, list)
	return list, int(total), nil
}

func (r *contractRepo) GetArchivedByClientID(ctx context.Context, clientID int64) ([]domain.Contract, error) {
	var list []domain.Contract
	if err := r.db.WithContext(ctx).
		Where("client_id = ? AND status = 'archived'", clientID).
		Order("archived_at DESC").
		Find(&list).Error; err != nil {
		return nil, err
	}
	enrichContracts(ctx, r.db, list)
	return list, nil
}
