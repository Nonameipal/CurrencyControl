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

func tryArchiveAdditionalAgreementGorm(ctx context.Context, db *gorm.DB, addlID int64) {
	var aa domain.AdditionalAgreement
	if err := db.WithContext(ctx).
		Select("contract_id, remaining_amount, status").
		Where("id = ? AND status = 'active'", addlID).
		First(&aa).Error; err != nil {
		return
	}

	if aa.RemainingAmount > 0 {
		return
	}

	var unclosedCount int64
	_ = db.WithContext(ctx).Table("invoices i").
		Where("i.additional_agreement_id = ? AND i.deleted_at IS NULL AND "+unclosedInvoiceCondition, addlID).
		Count(&unclosedCount).Error

	if unclosedCount > 0 {
		return
	}

	now := time.Now()
	_ = db.WithContext(ctx).Model(&domain.AdditionalAgreement{}).
		Where("id = ?", addlID).
		Updates(map[string]interface{}{
			"status":      "archived",
			"archived_at": &now,
		})

	if aa.ContractID > 0 {
		tryArchiveContractGorm(ctx, db, aa.ContractID)
	}
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
		Where("i.contract_id = ? AND i.additional_agreement_id IS NULL AND i.deleted_at IS NULL AND "+unclosedInvoiceCondition, contractID).
		Count(&unclosedContractInvoices).Error

	if unclosedContractInvoices > 0 {
		return
	}

	var activeAgreements int64
	_ = db.WithContext(ctx).Model(&domain.AdditionalAgreement{}).
		Where("contract_id = ? AND status = 'active'", contractID).
		Count(&activeAgreements).Error

	if activeAgreements > 0 {
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

func syncContractRemainingGorm(ctx context.Context, db *gorm.DB, contractID int64) {
	var usedDeduct float64
	db.WithContext(ctx).Model(&domain.Invoice{}).
		Where("contract_id = ? AND additional_agreement_id IS NULL AND deleted_at IS NULL", contractID).
		Select("COALESCE(SUM(deduct_amount), 0)").
		Scan(&usedDeduct)

	var c domain.Contract
	if err := db.WithContext(ctx).Select("total_amount").First(&c, contractID).Error; err == nil {
		rem := c.TotalAmount - usedDeduct
		db.WithContext(ctx).Model(&domain.Contract{}).Where("id = ?", contractID).Update("remaining_amount", rem)
	}
}

func syncAdditionalAgreementRemainingGorm(ctx context.Context, db *gorm.DB, addlID int64) {
	var sumInvoices float64
	db.WithContext(ctx).Model(&domain.Invoice{}).
		Where("additional_agreement_id = ? AND deleted_at IS NULL", addlID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&sumInvoices)

	var aa domain.AdditionalAgreement
	if err := db.WithContext(ctx).Select("foreign_amount").First(&aa, addlID).Error; err == nil {
		fa := 0.0
		if aa.ForeignAmount != nil {
			fa = *aa.ForeignAmount
		}
		rem := fa - sumInvoices
		db.WithContext(ctx).Model(&domain.AdditionalAgreement{}).Where("id = ?", addlID).Update("remaining_amount", rem)
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
