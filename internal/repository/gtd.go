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

func NewGTDRepository(db *gorm.DB) ports.GTDRepository {
	return &gtdRepo{db: db}
}

type gtdRepo struct{ db *gorm.DB }

func (r *gtdRepo) Create(ctx context.Context, g domain.GTD) (domain.GTD, error) {
	var inv domain.Invoice
	if err := r.db.WithContext(ctx).First(&inv, g.InvoiceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.GTD{}, fmt.Errorf("инвойс не найден")
		}
		return domain.GTD{}, err
	}

	var alreadyClosed float64
	r.db.WithContext(ctx).Model(&domain.GTD{}).
		Where("invoice_id = ?", g.InvoiceID).
		Select("COALESCE(SUM(closes_amount), 0)").
		Scan(&alreadyClosed)

	if g.GTDCurrency != nil && !strings.EqualFold(*g.GTDCurrency, inv.Currency) {
		return domain.GTD{}, fmt.Errorf("валюта ГТД (%s) должна совпадать с валютой инвойса (%s)", *g.GTDCurrency, inv.Currency)
	}

	g.AdditionalAgreementID = inv.AdditionalAgreementID

	if g.ClosesAmount <= 0 {
		g.ClosesAmount = g.GTDAmount
	}

	invoiceRemaining := inv.Amount - alreadyClosed
	if g.ClosesAmount > invoiceRemaining {
		return domain.GTD{}, fmt.Errorf(
			"сумма ГТД (%.2f) превышает товарный остаток по инвойсу (%.2f)",
			g.ClosesAmount, invoiceRemaining,
		)
	}

	if g.DocumentType == "" {
		g.DocumentType = domain.DocumentTypeGTD
	}

	var contract domain.Contract
	_ = r.db.WithContext(ctx).First(&contract, g.ContractID)

	var aa domain.AdditionalAgreement
	hasAA := r.db.WithContext(ctx).
		Where("contract_id = ? AND delivery_date IS NOT NULL", g.ContractID).
		Order("agreement_date DESC, id DESC").
		First(&aa).Error == nil

	var deadline *time.Time
	if hasAA && aa.DeliveryDate != nil && !aa.DeliveryDate.IsZero() {
		deadline = aa.DeliveryDate
	} else if !contract.DeliveryDate.IsZero() {
		deadline = &contract.DeliveryDate
	}

	now := time.Now()
	submissionDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	g.SubmissionDate = &submissionDate
	g.DeliveryDeadline = deadline

	var adjustedDeadline *time.Time
	if deadline != nil && !deadline.IsZero() {
		dl := time.Date(deadline.Year(), deadline.Month(), deadline.Day(), 0, 0, 0, 0, time.UTC)
		dl5 := dl.AddDate(0, 0, 5)
		adjustedDeadline = &dl5
	}

	diffDays, status, notice := domain.CalculateDeliveryComparison(g.DocumentType, submissionDate, adjustedDeadline)
	g.DaysDifference = diffDays
	g.DeliveryStatus = status
	g.DeliveryNotice = notice

	if g.ApprovalStatus == "" {
		g.ApprovalStatus = domain.ApprovalStatusPendingCurrencyControl
	}

	if strings.TrimSpace(g.HSCode) == "" {
		g.HSCode = domain.DefaultHSCode
	}

	if err := r.db.WithContext(ctx).Create(&g).Error; err != nil {
		return domain.GTD{}, err
	}
	g.InvoiceNumber = inv.InvoiceNumber

	if g.AdditionalAgreementID != nil {
		tryArchiveAdditionalAgreementGorm(ctx, r.db, *g.AdditionalAgreementID)
	} else {
		tryArchiveContractGorm(ctx, r.db, g.ContractID)
	}

	return g, nil
}

func (r *gtdRepo) GetByID(ctx context.Context, id int64) (*domain.GTD, error) {
	var g domain.GTD
	err := r.db.WithContext(ctx).Table("gtd g").
		Select("g.*, COALESCE(i.invoice_number, '') as invoice_number").
		Joins("LEFT JOIN invoices i ON i.id = g.invoice_id").
		Where("g.id = ? AND g.deleted_at IS NULL", id).
		Scan(&g).Error
	if err != nil || g.ID == 0 {
		return nil, errors.New("ГТД не найдена")
	}
	enrichGTD(ctx, r.db, &g)
	return &g, nil
}

func (r *gtdRepo) GetByInvoiceID(ctx context.Context, invoiceID int64) (*domain.GTD, error) {
	var g domain.GTD
	err := r.db.WithContext(ctx).Table("gtd g").
		Select("g.*, COALESCE(i.invoice_number, '') as invoice_number").
		Joins("LEFT JOIN invoices i ON i.id = g.invoice_id").
		Where("g.invoice_id = ? AND g.deleted_at IS NULL", invoiceID).
		Order("g.id DESC").
		Limit(1).
		Scan(&g).Error
	if err != nil || g.ID == 0 {
		return nil, nil
	}
	enrichGTD(ctx, r.db, &g)
	return &g, nil
}

func (r *gtdRepo) GetListByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.GTD, error) {
	var list []domain.GTD
	err := r.db.WithContext(ctx).Table("gtd g").
		Select("g.*, COALESCE(i.invoice_number, '') as invoice_number").
		Joins("LEFT JOIN invoices i ON i.id = g.invoice_id").
		Where("g.invoice_id = ? AND g.deleted_at IS NULL", invoiceID).
		Order("g.id ASC").
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	enrichGTDs(ctx, r.db, list)
	return list, nil
}

func (r *gtdRepo) GetByContractID(ctx context.Context, contractID int64) ([]domain.GTD, error) {
	var list []domain.GTD
	err := r.db.WithContext(ctx).Table("gtd g").
		Select("g.*, COALESCE(i.invoice_number, '') as invoice_number").
		Joins("LEFT JOIN invoices i ON i.id = g.invoice_id").
		Where("g.contract_id = ? AND g.deleted_at IS NULL", contractID).
		Order("g.id ASC").
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	enrichGTDs(ctx, r.db, list)
	return list, nil
}

func (r *gtdRepo) GetByAdditionalAgreementID(ctx context.Context, agreementID int64) ([]domain.GTD, error) {
	var list []domain.GTD
	err := r.db.WithContext(ctx).Table("gtd g").
		Select("g.*, COALESCE(i.invoice_number, '') as invoice_number").
		Joins("LEFT JOIN invoices i ON i.id = g.invoice_id").
		Where("g.additional_agreement_id = ? AND g.deleted_at IS NULL", agreementID).
		Order("g.id ASC").
		Scan(&list).Error
	if err != nil {
		return nil, err
	}
	enrichGTDs(ctx, r.db, list)
	return list, nil
}

func (r *gtdRepo) SoftDelete(ctx context.Context, id int64) error {
	var g domain.GTD
	if err := r.db.WithContext(ctx).First(&g, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("ГТД не найдена")
		}
		return err
	}

	userLogin := domain.GetLoginFromCtx(ctx)
	if userLogin != "" {
		_ = r.db.WithContext(ctx).Model(&domain.GTD{}).Where("id = ?", id).Update("deleted_by", userLogin).Error
	}

	res := r.db.WithContext(ctx).Delete(&domain.GTD{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("ГТД не найдена")
	}

	if g.AdditionalAgreementID != nil {
		tryArchiveAdditionalAgreementGorm(ctx, r.db, *g.AdditionalAgreementID)
	} else if g.ContractID > 0 {
		tryArchiveContractGorm(ctx, r.db, g.ContractID)
	}
	return nil
}

func (r *gtdRepo) Update(ctx context.Context, id int64, g domain.GTD) (domain.GTD, error) {
	var existing domain.GTD
	if err := r.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.GTD{}, errors.New("ГТД не найдена")
		}
		return domain.GTD{}, err
	}

	var inv domain.Invoice
	if err := r.db.WithContext(ctx).First(&inv, existing.InvoiceID).Error; err == nil {
		if g.GTDCurrency != nil && inv.Currency != "" && !strings.EqualFold(*g.GTDCurrency, inv.Currency) {
			return domain.GTD{}, fmt.Errorf("валюта ГТД (%s) должна совпадать с валютой инвойса (%s)", *g.GTDCurrency, inv.Currency)
		}
	}

	if g.DocumentType == "" {
		g.DocumentType = domain.DocumentTypeGTD
	}
	if g.ClosesAmount <= 0 {
		g.ClosesAmount = g.GTDAmount
	}

	if strings.TrimSpace(g.HSCode) == "" {
		g.HSCode = domain.DefaultHSCode
	}

	updates := map[string]interface{}{
		"document_type":       g.DocumentType,
		"gtd_number":          g.GTDNumber,
		"gtd_amount":          g.GTDAmount,
		"gtd_currency":        g.GTDCurrency,
		"gtd_date":            g.GTDDate,
		"closes_amount":       g.ClosesAmount,
		"hs_code":             g.HSCode,
		"destination_country": g.DestinationCountry,
		"document_path":       g.DocumentPath,
		"sender_name":         g.SenderName,
		"sender_bank":         g.SenderBank,
		"sender_country":      g.SenderCountry,
		"approval_status":     domain.ApprovalStatusPendingCurrencyControl,
		"rejection_reason":    "",
	}

	userLogin := domain.GetLoginFromCtx(ctx)
	if userLogin == "" {
		userLogin = g.UpdatedBy
	}
	if userLogin != "" {
		updates["updated_by"] = userLogin
	}

	if err := r.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
		return domain.GTD{}, err
	}

	var updated domain.GTD
	if err := r.db.WithContext(ctx).Table("gtd g").
		Select("g.*, COALESCE(i.invoice_number, '') as invoice_number").
		Joins("LEFT JOIN invoices i ON i.id = g.invoice_id").
		Where("g.id = ? AND g.deleted_at IS NULL", id).
		Scan(&updated).Error; err != nil {
		return domain.GTD{}, err
	}

	enrichGTD(ctx, r.db, &updated)
	return updated, nil
}
func (r *gtdRepo) ResetApprovalStatus(ctx context.Context, id int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Table("gtd").
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

