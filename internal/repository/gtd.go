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

	g.ContractID = inv.ContractID
	if g.Currency == "" {
		g.Currency = inv.Currency
	}
	if !strings.EqualFold(g.Currency, inv.Currency) {
		return domain.GTD{}, fmt.Errorf("валюта ГТД (%s) должна совпадать с валютой инвойса (%s)", g.Currency, inv.Currency)
	}

	g.ClosesAmount = g.GTDAmount

	var alreadyClosed float64
	r.db.WithContext(ctx).Model(&domain.GTD{}).
		Where("invoice_id = ? AND deleted_at IS NULL", g.InvoiceID).
		Select("COALESCE(SUM(closes_amount), 0)").
		Scan(&alreadyClosed)

	invoiceRemaining := inv.Amount - alreadyClosed
	if g.ClosesAmount > invoiceRemaining {
		return domain.GTD{}, fmt.Errorf(
			"сумма ГТД (%.2f %s) превышает доступный остаток по инвойсу (%.2f %s). Уже закрыто: %.2f из %.2f %s",
			g.ClosesAmount, inv.Currency, invoiceRemaining, inv.Currency, alreadyClosed, inv.Amount, inv.Currency,
		)
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

	diffDays, status, notice := domain.CalculateDeliveryComparison(submissionDate, adjustedDeadline)
	g.DaysDifference = diffDays
	g.DeliveryStatus = status
	g.DeliveryNotice = notice

	if g.ApprovalStatus == "" {
		g.ApprovalStatus = domain.ApprovalStatusPendingCurrencyControl
	}

	if err := r.db.WithContext(ctx).Create(&g).Error; err != nil {
		return domain.GTD{}, err
	}
	g.InvoiceNumber = inv.InvoiceNumber

	tryArchiveContractGorm(ctx, r.db, g.ContractID)

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

	tryArchiveContractGorm(ctx, r.db, g.ContractID)
	return nil
}

func (r *gtdRepo) Update(ctx context.Context, id int64, g domain.GTD) (domain.GTD, error) {
	var existing domain.GTD
	if err := r.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.GTD{}, fmt.Errorf("ГТД не найдена")
		}
		return domain.GTD{}, err
	}

	g.ClosesAmount = g.GTDAmount

	var inv domain.Invoice
	if err := r.db.WithContext(ctx).First(&inv, existing.InvoiceID).Error; err == nil {
		if g.Currency != "" && inv.Currency != "" && !strings.EqualFold(g.Currency, inv.Currency) {
			return domain.GTD{}, fmt.Errorf("валюта ГТД (%s) должна совпадать с валютой инвойса (%s)", g.Currency, inv.Currency)
		}

		var otherClosed float64
		r.db.WithContext(ctx).Model(&domain.GTD{}).
			Where("invoice_id = ? AND id != ? AND deleted_at IS NULL", existing.InvoiceID, id).
			Select("COALESCE(SUM(closes_amount), 0)").
			Scan(&otherClosed)

		invoiceRemaining := inv.Amount - otherClosed
		if g.ClosesAmount > invoiceRemaining {
			return domain.GTD{}, fmt.Errorf(
				"сумма ГТД (%.2f %s) превышает доступный остаток по инвойсу (%.2f %s). Уже закрыто другими документами: %.2f из %.2f %s",
				g.ClosesAmount, inv.Currency, invoiceRemaining, inv.Currency, otherClosed, inv.Amount, inv.Currency,
			)
		}
	}

	updates := map[string]interface{}{
		"gtd_number":       g.GTDNumber,
		"gtd_amount":       g.GTDAmount,
		"gtd_date":         g.GTDDate,
		"closes_amount":    g.ClosesAmount,
		"document_path":    g.DocumentPath,
		"approval_status":  domain.ApprovalStatusPendingCurrencyControl,
		"rejection_reason": "",
	}
	if g.Currency != "" {
		updates["currency"] = g.Currency
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

	tryArchiveContractGorm(ctx, r.db, updated.ContractID)

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
