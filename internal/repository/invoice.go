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

type invoiceRepo struct{ db *gorm.DB }

func NewInvoiceRepository(db *gorm.DB) ports.InvoiceRepository {
	return &invoiceRepo{db: db}
}

func (r *invoiceRepo) Create(ctx context.Context, inv domain.Invoice, contractCurrency string) (domain.Invoice, error) {
	var contract domain.Contract
	if err := r.db.WithContext(ctx).First(&contract, inv.ContractID).Error; err != nil {
		return domain.Invoice{}, fmt.Errorf("контракт не найден: %w", err)
	}

	if isExpired, exp := contract.IsExpired(time.Now()); isExpired {
		return domain.Invoice{}, fmt.Errorf("срок действия контракта истёк (%s). Добавление инвойсов запрещено", exp.Format("02.01.2006"))
	}

	if err := contract.ValidateDocumentDate("Инвойс", inv.InvoiceDate); err != nil {
		return domain.Invoice{}, err
	}

	invCurr := strings.ToUpper(strings.TrimSpace(inv.Currency))
	contractCurr := strings.ToUpper(strings.TrimSpace(contract.ContractCurrency))

	if strings.EqualFold(invCurr, contractCurr) {
		var usedDeduct float64
		r.db.WithContext(ctx).Model(&domain.Invoice{}).
			Where("contract_id = ? AND deleted_at IS NULL AND UPPER(TRIM(currency)) = ?", contract.ID, contractCurr).
			Select("COALESCE(SUM(amount), 0)").
			Scan(&usedDeduct)

		rem := contract.TotalAmount - usedDeduct
		if inv.Amount > rem {
			return domain.Invoice{}, fmt.Errorf(
				"сумма инвойса (%.2f %s) превышает остаток по контракту (%.2f %s)",
				inv.Amount, contractCurr, rem, contractCurr,
			)
		}
	} else {
		var addlTotal float64
		r.db.WithContext(ctx).Model(&domain.AdditionalAgreement{}).
			Where("contract_id = ? AND deleted_at IS NULL AND status = 'active' AND UPPER(TRIM(currency)) = ?", contract.ID, invCurr).
			Select("COALESCE(SUM(COALESCE(foreign_amount, 0)), 0)").
			Scan(&addlTotal)

		if addlTotal <= 0 {
			return domain.Invoice{}, fmt.Errorf("в контракте отсутствует валюта %s (добавьте соответствующее доп. соглашение)", invCurr)
		}

		var usedForCurrency float64
		r.db.WithContext(ctx).Model(&domain.Invoice{}).
			Where("contract_id = ? AND deleted_at IS NULL AND UPPER(TRIM(currency)) = ?", contract.ID, invCurr).
			Select("COALESCE(SUM(amount), 0)").
			Scan(&usedForCurrency)

		rem := addlTotal - usedForCurrency
		if inv.Amount > rem {
			return domain.Invoice{}, fmt.Errorf(
				"сумма инвойса (%.2f %s) превышает доступный лимит по доп. соглашениям в этой валюте (остаток: %.2f %s)",
				inv.Amount, invCurr, rem, invCurr,
			)
		}
	}

	inv.Currency = invCurr

	if err := r.db.WithContext(ctx).Create(&inv).Error; err != nil {
		return domain.Invoice{}, err
	}

	_ = syncContractBalances(ctx, r.db, inv.ContractID)
	tryArchiveContractGorm(ctx, r.db, inv.ContractID)

	return inv, nil
}

func (r *invoiceRepo) GetByID(ctx context.Context, id int64) (domain.Invoice, error) {
	var inv domain.Invoice
	if err := r.db.WithContext(ctx).First(&inv, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Invoice{}, errors.New("Инвойс не найден")
		}
		return inv, err
	}
	enrichInvoice(ctx, r.db, &inv)
	return inv, nil
}

func (r *invoiceRepo) Update(ctx context.Context, id int64, inv domain.Invoice) (domain.Invoice, error) {
	var existing domain.Invoice
	if err := r.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Invoice{}, errors.New("Инвойс не найден")
		}
		return domain.Invoice{}, err
	}
	var contract domain.Contract
	if err := r.db.WithContext(ctx).First(&contract, existing.ContractID).Error; err != nil {
		return domain.Invoice{}, fmt.Errorf("контракт не найден: %w", err)
	}

	targetDate := inv.InvoiceDate
	if targetDate.IsZero() {
		targetDate = existing.InvoiceDate
	}
	if err := contract.ValidateDocumentDate("Инвойс", targetDate); err != nil {
		return domain.Invoice{}, err
	}
	invCurr := strings.ToUpper(strings.TrimSpace(inv.Currency))
	if invCurr == "" {
		invCurr = strings.ToUpper(strings.TrimSpace(existing.Currency))
	}
	contractCurr := strings.ToUpper(strings.TrimSpace(contract.ContractCurrency))
	if strings.EqualFold(invCurr, contractCurr) {
		var usedDeduct float64
		r.db.WithContext(ctx).Model(&domain.Invoice{}).
			Where("contract_id = ? AND deleted_at IS NULL AND id != ? AND UPPER(TRIM(currency)) = ?", contract.ID, id, contractCurr).
			Select("COALESCE(SUM(amount), 0)").
			Scan(&usedDeduct)
		rem := contract.TotalAmount - usedDeduct
		if inv.Amount > rem {
			return domain.Invoice{}, fmt.Errorf(
				"сумма инвойса (%.2f %s) превышает остаток по контракту (%.2f %s)",
				inv.Amount, contractCurr, rem, contractCurr,
			)
		}
	} else {
		var addlTotal float64
		r.db.WithContext(ctx).Model(&domain.AdditionalAgreement{}).
			Where("contract_id = ? AND deleted_at IS NULL AND status = 'active' AND UPPER(TRIM(currency)) = ?", contract.ID, invCurr).
			Select("COALESCE(SUM(COALESCE(foreign_amount, 0)), 0)").
			Scan(&addlTotal)
		if addlTotal <= 0 {
			return domain.Invoice{}, fmt.Errorf("в контракте отсутствует валюта %s (добавьте соответствующее доп. соглашение)", invCurr)
		}
		var usedForCurrency float64
		r.db.WithContext(ctx).Model(&domain.Invoice{}).
			Where("contract_id = ? AND deleted_at IS NULL AND id != ? AND UPPER(TRIM(currency)) = ?", contract.ID, id, invCurr).
			Select("COALESCE(SUM(amount), 0)").
			Scan(&usedForCurrency)
		rem := addlTotal - usedForCurrency
		if inv.Amount > rem {
			return domain.Invoice{}, fmt.Errorf(
				"сумма инвойса (%.2f %s) превышает доступный лимит по доп. соглашениям в этой валюте (остаток: %.2f %s)",
				inv.Amount, invCurr, rem, invCurr,
			)
		}
	}

	updates := map[string]interface{}{
		"invoice_number": inv.InvoiceNumber,
		"invoice_date":   inv.InvoiceDate,
		"amount":         inv.Amount,
		"currency":       invCurr,
		"document_path":  inv.DocumentPath,
	}

	userLogin := domain.GetLoginFromCtx(ctx)
	if userLogin == "" {
		userLogin = inv.UpdatedBy
	}
	if userLogin != "" {
		updates["updated_by"] = userLogin
	}

	if err := r.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
		return domain.Invoice{}, err
	}

	_ = syncContractBalances(ctx, r.db, existing.ContractID)
	tryArchiveContractGorm(ctx, r.db, existing.ContractID)

	if err := r.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		return domain.Invoice{}, err
	}
	enrichInvoice(ctx, r.db, &existing)
	return existing, nil
}

func (r *invoiceRepo) SoftDelete(ctx context.Context, id int64) error {
	var existing domain.Invoice
	if err := r.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("Инвойс не найден")
		}
		return err
	}

	userLogin := domain.GetLoginFromCtx(ctx)
	if userLogin != "" {
		_ = r.db.WithContext(ctx).Model(&domain.Invoice{}).Where("id = ?", id).Update("deleted_by", userLogin).Error
	}

	res := r.db.WithContext(ctx).Delete(&domain.Invoice{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("Инвойс не найден")
	}

	if existing.ContractID > 0 {
		_ = syncContractBalances(ctx, r.db, existing.ContractID)
		tryArchiveContractGorm(ctx, r.db, existing.ContractID)
	}

	return nil
}

func (r *invoiceRepo) GetByContractID(ctx context.Context, contractID int64) ([]domain.InvoiceWithDetails, error) {
	var invoices []domain.Invoice
	if err := r.db.WithContext(ctx).
		Where("contract_id = ?", contractID).
		Order("invoice_date ASC").
		Find(&invoices).Error; err != nil {
		return nil, err
	}
	return r.buildInvoiceDetails(ctx, invoices)
}

func (r *invoiceRepo) buildInvoiceDetails(ctx context.Context, invoices []domain.Invoice) ([]domain.InvoiceWithDetails, error) {
	enrichInvoices(ctx, r.db, invoices)
	if len(invoices) == 0 {
		return []domain.InvoiceWithDetails{}, nil
	}
	invoiceIDs := make([]int64, len(invoices))
	for i, inv := range invoices {
		invoiceIDs[i] = inv.ID
	}

	var allGTDs []domain.GTD
	r.db.WithContext(ctx).Table("gtd g").
		Select("g.*, COALESCE(i.invoice_number, '') as invoice_number").
		Joins("LEFT JOIN invoices i ON i.id = g.invoice_id").
		Where("g.invoice_id IN ? AND g.deleted_at IS NULL", invoiceIDs).
		Order("g.invoice_id, g.id DESC").
		Find(&allGTDs)
	enrichGTDs(ctx, r.db, allGTDs)

	gtdByInvoice := make(map[int64]*domain.GTD, len(allGTDs))
	for i := range allGTDs {
		g := &allGTDs[i]
		if _, exists := gtdByInvoice[g.InvoiceID]; !exists {
			gtdByInvoice[g.InvoiceID] = g
		}
	}

	var allPOs []domain.PaymentOrder
	r.db.WithContext(ctx).
		Where("invoice_id IN ?", invoiceIDs).
		Order("invoice_id, value_date DESC, id DESC").
		Find(&allPOs)
	enrichPaymentOrders(ctx, r.db, allPOs)

	posByInvoice := make(map[int64][]domain.PaymentOrder, len(invoices))
	for _, po := range allPOs {
		posByInvoice[po.InvoiceID] = append(posByInvoice[po.InvoiceID], po)
	}

	var allActs []domain.Act
	r.db.WithContext(ctx).Table("acts").
		Where("invoice_id IN ? AND deleted_at IS NULL", invoiceIDs).
		Order("invoice_id, act_date DESC, id DESC").
		Find(&allActs)
	enrichActs(ctx, r.db, allActs)

	actsByInvoice := make(map[int64][]domain.Act, len(invoices))
	for _, a := range allActs {
		actsByInvoice[a.InvoiceID] = append(actsByInvoice[a.InvoiceID], a)
	}

	result := make([]domain.InvoiceWithDetails, 0, len(invoices))
	for _, inv := range invoices {
		detail := domain.InvoiceWithDetails{Invoice: inv}
		if gtd, ok := gtdByInvoice[inv.ID]; ok {
			detail.GTD = gtd
		}
		if acts, ok := actsByInvoice[inv.ID]; ok {
			detail.Acts = acts
		}
		if pos, ok := posByInvoice[inv.ID]; ok {
			detail.PaymentOrders = pos
			var paid float64
			for _, p := range pos {
				paid += p.Amount
			}
			detail.PaidAmount = paid
			rem := inv.Amount - paid
			if rem < 0 {
				rem = 0
			}
			detail.RemainingPaymentAmount = rem
		} else {
			detail.RemainingPaymentAmount = inv.Amount
		}

		result = append(result, detail)
	}
	return result, nil
}
