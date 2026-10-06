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
	if inv.AdditionalAgreementID == nil {
		var contract domain.Contract
		if err := r.db.WithContext(ctx).First(&contract, inv.ContractID).Error; err != nil {
			return domain.Invoice{}, fmt.Errorf("контракт не найден: %w", err)
		}

		today := time.Now().UTC().Truncate(24 * time.Hour)
		var expiryDate *time.Time
		if contract.ExtendDateTo != nil && !contract.ExtendDateTo.IsZero() {
			expiryDate = contract.ExtendDateTo
		} else if !contract.DeliveryDate.IsZero() {
			expiryDate = &contract.DeliveryDate
		}
		if expiryDate != nil && !expiryDate.IsZero() {
			exp := time.Date(expiryDate.Year(), expiryDate.Month(), expiryDate.Day(), 0, 0, 0, 0, time.UTC)
			if today.After(exp) {
				return domain.Invoice{}, fmt.Errorf("срок действия контракта истёк (%s). Добавление инвойсов запрещено", exp.Format("02.01.2006"))
			}
		}

		if !strings.EqualFold(inv.Currency, contract.ContractCurrency) {
			return domain.Invoice{}, fmt.Errorf("валюта инвойса (%s) должна совпадать с валютой контракта (%s)", inv.Currency, contract.ContractCurrency)
		}

		var usedDeductAmount float64
		r.db.WithContext(ctx).Model(&domain.Invoice{}).
			Where("contract_id = ? AND additional_agreement_id IS NULL AND deleted_at IS NULL", contract.ID).
			Select("COALESCE(SUM(deduct_amount), 0)").
			Scan(&usedDeductAmount)

		contractRemaining := contract.TotalAmount - usedDeductAmount
		inv.DeductAmount = inv.Amount

		if inv.Amount > contractRemaining {
			return domain.Invoice{}, fmt.Errorf(
				"сумма инвойса (%.2f %s) превышает остаток по контракту (%.2f %s)",
				inv.Amount, contract.ContractCurrency, contractRemaining, contract.ContractCurrency,
			)
		}
	} else {
		var aa domain.AdditionalAgreement
		if err := r.db.WithContext(ctx).Where("id = ? AND contract_id = ?", *inv.AdditionalAgreementID, inv.ContractID).First(&aa).Error; err != nil {
			return domain.Invoice{}, fmt.Errorf("дополнительное соглашение не найдено: %w", err)
		}

		aaCurr := ""
		if aa.ForeignCurrency != nil {
			aaCurr = *aa.ForeignCurrency
		}
		if !strings.EqualFold(inv.Currency, aaCurr) {
			return domain.Invoice{}, fmt.Errorf("валюта инвойса (%s) должна совпадать с валютой доп. соглашения (%s)", inv.Currency, aaCurr)
		}

		var usedAmount float64
		r.db.WithContext(ctx).Model(&domain.Invoice{}).
			Where("additional_agreement_id = ? AND deleted_at IS NULL", aa.ID).
			Select("COALESCE(SUM(amount), 0)").
			Scan(&usedAmount)

		fa := 0.0
		if aa.ForeignAmount != nil {
			fa = *aa.ForeignAmount
		}
		addlRemaining := fa - usedAmount
		inv.DeductAmount = inv.Amount

		if inv.Amount > addlRemaining {
			return domain.Invoice{}, fmt.Errorf(
				"сумма инвойса (%.2f %s) превышает остаток по доп. соглашению (%.2f %s)",
				inv.Amount, aaCurr, addlRemaining, aaCurr,
			)
		}
	}

	if inv.ApprovalStatus == "" {
		inv.ApprovalStatus = domain.ApprovalStatusPendingCurrencyControl
	}

	if strings.TrimSpace(inv.HSCode) == "" {
		inv.HSCode = domain.DefaultHSCode
	}

	if err := r.db.WithContext(ctx).Create(&inv).Error; err != nil {
		return domain.Invoice{}, err
	}

	if inv.AdditionalAgreementID != nil {
		syncAdditionalAgreementRemainingGorm(ctx, r.db, *inv.AdditionalAgreementID)
		tryArchiveAdditionalAgreementGorm(ctx, r.db, *inv.AdditionalAgreementID)
	} else {
		syncContractRemainingGorm(ctx, r.db, inv.ContractID)
		tryArchiveContractGorm(ctx, r.db, inv.ContractID)
	}

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

	var expectedCurrency string
	if existing.AdditionalAgreementID != nil {
		var aa domain.AdditionalAgreement
		if err := r.db.WithContext(ctx).Select("currency").First(&aa, *existing.AdditionalAgreementID).Error; err == nil && aa.ForeignCurrency != nil {
			expectedCurrency = *aa.ForeignCurrency
		}
	} else {
		var c domain.Contract
		if err := r.db.WithContext(ctx).Select("contract_currency").First(&c, existing.ContractID).Error; err == nil {
			expectedCurrency = c.ContractCurrency
		}
	}

	if inv.Currency != "" && expectedCurrency != "" && !strings.EqualFold(inv.Currency, expectedCurrency) {
		return domain.Invoice{}, fmt.Errorf("валюта инвойса (%s) должна совпадать с валютой документа (%s)", inv.Currency, expectedCurrency)
	}

	if inv.DeductAmount <= 0 {
		inv.DeductAmount = inv.Amount
	}

	if strings.TrimSpace(inv.HSCode) == "" {
		inv.HSCode = domain.DefaultHSCode
	}

	updates := map[string]interface{}{
		"invoice_number":   inv.InvoiceNumber,
		"invoice_date":     inv.InvoiceDate,
		"amount":           inv.Amount,
		"currency":         inv.Currency,
		"hs_code":          inv.HSCode,
		"deduct_amount":    inv.DeductAmount,
		"document_path":    inv.DocumentPath,
		"sender_name":      inv.SenderName,
		"sender_bank":      inv.SenderBank,
		"sender_country":   inv.SenderCountry,
		"approval_status":  domain.ApprovalStatusPendingCurrencyControl,
		"rejection_reason": "",
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

	if err := r.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		return domain.Invoice{}, err
	}
	if existing.AdditionalAgreementID != nil {
		syncAdditionalAgreementRemainingGorm(ctx, r.db, *existing.AdditionalAgreementID)
		tryArchiveAdditionalAgreementGorm(ctx, r.db, *existing.AdditionalAgreementID)
	} else {
		syncContractRemainingGorm(ctx, r.db, existing.ContractID)
		tryArchiveContractGorm(ctx, r.db, existing.ContractID)
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

	if existing.AdditionalAgreementID != nil {
		syncAdditionalAgreementRemainingGorm(ctx, r.db, *existing.AdditionalAgreementID)
		tryArchiveAdditionalAgreementGorm(ctx, r.db, *existing.AdditionalAgreementID)
	} else if existing.ContractID > 0 {
		syncContractRemainingGorm(ctx, r.db, existing.ContractID)
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

func (r *invoiceRepo) GetByAdditionalAgreementID(ctx context.Context, agreementID int64) ([]domain.InvoiceWithDetails, error) {
	var invoices []domain.Invoice
	if err := r.db.WithContext(ctx).
		Where("additional_agreement_id = ?", agreementID).
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

	// Собираем все ID инвойсов для батч-запросов
	invoiceIDs := make([]int64, len(invoices))
	for i, inv := range invoices {
		invoiceIDs[i] = inv.ID
	}

	// Батч-запрос GTD — один запрос вместо N
	var allGTDs []domain.GTD
	r.db.WithContext(ctx).Table("gtd g").
		Select("g.*, COALESCE(i.invoice_number, '') as invoice_number").
		Joins("LEFT JOIN invoices i ON i.id = g.invoice_id").
		Where("g.invoice_id IN ? AND g.deleted_at IS NULL", invoiceIDs).
		Order("g.invoice_id, g.id DESC").
		Find(&allGTDs)
	enrichGTDs(ctx, r.db, allGTDs)

	// Группируем GTD по invoice_id: берём только последний на инвойс
	gtdByInvoice := make(map[int64]*domain.GTD, len(allGTDs))
	for i := range allGTDs {
		g := &allGTDs[i]
		if _, exists := gtdByInvoice[g.InvoiceID]; !exists {
			gtdByInvoice[g.InvoiceID] = g
		}
	}

	// Батч-запрос PaymentOrders — один запрос вместо N
	var allPOs []domain.PaymentOrder
	r.db.WithContext(ctx).
		Where("invoice_id IN ?", invoiceIDs).
		Order("invoice_id, operation_date DESC, id DESC").
		Find(&allPOs)
	enrichPaymentOrders(ctx, r.db, allPOs)

	// Группируем PaymentOrders по invoice_id
	posByInvoice := make(map[int64][]domain.PaymentOrder, len(invoices))
	for _, po := range allPOs {
		posByInvoice[po.InvoiceID] = append(posByInvoice[po.InvoiceID], po)
	}

	// Собираем результат без дополнительных запросов к БД
	result := make([]domain.InvoiceWithDetails, 0, len(invoices))
	for _, inv := range invoices {
		detail := domain.InvoiceWithDetails{Invoice: inv}

		if gtd, ok := gtdByInvoice[inv.ID]; ok {
			detail.GTD = gtd
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

// ResetApprovalStatus сбрасывает статус согласования инвойса обратно
// в pending_currency_control (используется после редактирования при статусе revision_required).
func (r *invoiceRepo) ResetApprovalStatus(ctx context.Context, id int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Table("invoices").
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"approval_status":              domain.ApprovalStatusPendingCurrencyControl,
			"currency_control_decision":    "",
			"currency_control_comment":     "",
			"currency_control_reviewed_by": "",
			"currency_control_reviewed_at": nil,
			"rejection_reason":             "",
			"updated_at":                   &now,
		}).Error
}

