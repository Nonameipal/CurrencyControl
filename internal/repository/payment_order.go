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

type paymentOrderRepo struct {
	db *gorm.DB
}

func NewPaymentOrderRepository(db *gorm.DB) ports.PaymentOrderRepository {
	return &paymentOrderRepo{db: db}
}

func (r *paymentOrderRepo) Create(ctx context.Context, po domain.PaymentOrder) (domain.PaymentOrder, error) {
	if po.InvoiceID <= 0 {
		return domain.PaymentOrder{}, fmt.Errorf("поле invoice_id обязательно")
	}
	if po.Amount <= 0 {
		return domain.PaymentOrder{}, fmt.Errorf("сумма платежного поручения должна быть больше 0")
	}

	var inv domain.Invoice
	if err := r.db.WithContext(ctx).First(&inv, po.InvoiceID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.PaymentOrder{}, fmt.Errorf("инвойс не найден")
		}
		return domain.PaymentOrder{}, fmt.Errorf("ошибка проверки инвойса: %w", err)
	}

	var alreadyPaid float64
	r.db.WithContext(ctx).Model(&domain.PaymentOrder{}).
		Where("invoice_id = ?", po.InvoiceID).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&alreadyPaid)

	if !strings.EqualFold(strings.TrimSpace(po.Currency), strings.TrimSpace(inv.Currency)) {
		return domain.PaymentOrder{}, fmt.Errorf("валюта платежного поручения (%s) должна совпадать с валютой инвойса (%s)", po.Currency, inv.Currency)
	}
	remainingPayment := inv.Amount - alreadyPaid
	if po.Amount > remainingPayment {
		return domain.PaymentOrder{}, fmt.Errorf(
			"сумма платежного поручения (%.2f %s) превышает доступный остаток по оплате инвойса (%.2f %s)",
			po.Amount, inv.Currency, remainingPayment, inv.Currency,
		)
	}

	po.ContractID = inv.ContractID
	var contract domain.Contract
	if err := r.db.WithContext(ctx).First(&contract, po.ContractID).Error; err != nil {
		return domain.PaymentOrder{}, fmt.Errorf("контракт не найден: %w", err)
	}
	if err := contract.ValidateDocumentDate("Платежное поручение", po.ValueDate); err != nil {
		return domain.PaymentOrder{}, err
	}
	po.Currency = strings.ToUpper(strings.TrimSpace(po.Currency))
	if po.ApprovalStatus == "" {
		po.ApprovalStatus = domain.ApprovalStatusPendingCurrencyControl
	}

	if err := r.db.WithContext(ctx).Create(&po).Error; err != nil {
		return domain.PaymentOrder{}, fmt.Errorf("ошибка сохранения платежного поручения: %w", err)
	}
	tryArchiveContractGorm(ctx, r.db, po.ContractID)
	return po, nil
}

func (r *paymentOrderRepo) GetByID(ctx context.Context, id int64) (*domain.PaymentOrder, error) {
	var po domain.PaymentOrder
	if err := r.db.WithContext(ctx).First(&po, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("платежное поручение не найдено")
		}
		return nil, err
	}
	enrichPaymentOrder(ctx, r.db, &po)
	return &po, nil
}

func (r *paymentOrderRepo) GetByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.PaymentOrder, error) {
	var list []domain.PaymentOrder
	if err := r.db.WithContext(ctx).
		Where("invoice_id = ?", invoiceID).
		Order("value_date DESC, id DESC").
		Find(&list).Error; err != nil {
		return nil, err
	}
	enrichPaymentOrders(ctx, r.db, list)
	return list, nil
}

func (r *paymentOrderRepo) GetByContractID(ctx context.Context, contractID int64) ([]domain.PaymentOrder, error) {
	var list []domain.PaymentOrder
	if err := r.db.WithContext(ctx).
		Where("contract_id = ?", contractID).
		Order("value_date DESC, id DESC").
		Find(&list).Error; err != nil {
		return nil, err
	}
	enrichPaymentOrders(ctx, r.db, list)
	return list, nil
}

func (r *paymentOrderRepo) Update(ctx context.Context, id int64, po domain.PaymentOrder) (*domain.PaymentOrder, error) {
	existing, err := r.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if po.Amount <= 0 {
		return nil, fmt.Errorf("сумма платежного поручения должна быть больше 0")
	}

	var inv domain.Invoice
	if err := r.db.WithContext(ctx).First(&inv, existing.InvoiceID).Error; err != nil {
		return nil, fmt.Errorf("ошибка проверки инвойса: %w", err)
	}

	var alreadyPaidOther float64
	r.db.WithContext(ctx).Model(&domain.PaymentOrder{}).
		Where("invoice_id = ? AND id <> ?", existing.InvoiceID, id).
		Select("COALESCE(SUM(amount), 0)").
		Scan(&alreadyPaidOther)

	if po.Currency != "" && !strings.EqualFold(strings.TrimSpace(po.Currency), strings.TrimSpace(inv.Currency)) {
		return nil, fmt.Errorf("валюта платежного поручения (%s) должна совпадать с валютой инвойса (%s)", po.Currency, inv.Currency)
	}

	var contract domain.Contract
	if err := r.db.WithContext(ctx).First(&contract, existing.ContractID).Error; err != nil {
		return nil, fmt.Errorf("контракт не найден: %w", err)
	}
	targetDate := po.ValueDate
	if targetDate.IsZero() {
		targetDate = existing.ValueDate
	}
	if err := contract.ValidateDocumentDate("Платежное поручение", targetDate); err != nil {
		return nil, err
	}

	remainingPayment := inv.Amount - alreadyPaidOther
	if po.Amount > remainingPayment {
		return nil, fmt.Errorf(
			"сумма платежного поручения (%.2f %s) превышает доступный остаток по оплате инвойса (%.2f %s)",
			po.Amount, inv.Currency, remainingPayment, inv.Currency,
		)
	}

	curr := po.Currency
	if curr == "" {
		curr = existing.Currency
	}

	docPath := existing.DocumentPath
	if po.DocumentPath != nil {
		docPath = po.DocumentPath
	}
	updates := map[string]interface{}{
		"value_date":       po.ValueDate,
		"amount":           po.Amount,
		"currency":         strings.ToUpper(curr),
		"sender_name":      po.SenderName,
		"receiver_name":    po.ReceiverName,
		"receiver_bank":    po.ReceiverBank,
		"receiver_country": po.ReceiverCountry,
		"payment_purpose":  po.PaymentPurpose,
		"document_path":    docPath,
	}

	userLogin := domain.GetLoginFromCtx(ctx)
	if userLogin == "" {
		userLogin = po.UpdatedBy
	}
	if userLogin != "" {
		updates["updated_by"] = userLogin
	}

	if err := r.db.WithContext(ctx).Model(existing).Updates(updates).Error; err != nil {
		return nil, fmt.Errorf("ошибка обновления платежного поручения: %w", err)
	}

	if err := r.db.WithContext(ctx).First(existing, id).Error; err != nil {
		return nil, fmt.Errorf("ошибка получения обновлённого платежного поручения: %w", err)
	}

	tryArchiveContractGorm(ctx, r.db, existing.ContractID)

	enrichPaymentOrder(ctx, r.db, existing)
	return existing, nil
}

func (r *paymentOrderRepo) SoftDelete(ctx context.Context, id int64) error {
	existing, err := r.GetByID(ctx, id)
	if err != nil {
		return err
	}

	userLogin := domain.GetLoginFromCtx(ctx)
	if userLogin != "" {
		_ = r.db.WithContext(ctx).Model(&domain.PaymentOrder{}).Where("id = ?", id).Update("deleted_by", userLogin).Error
	}

	res := r.db.WithContext(ctx).Delete(&domain.PaymentOrder{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return fmt.Errorf("платежное поручение не найдено или уже удалено")
	}

	tryArchiveContractGorm(ctx, r.db, existing.ContractID)

	return nil
}

func (r *paymentOrderRepo) ResetApprovalStatus(ctx context.Context, id int64) error {
	now := time.Now()
	return r.db.WithContext(ctx).Table("payment_orders").
		Where("id = ? AND deleted_at IS NULL", id).
		Updates(map[string]interface{}{
			"approval_status":              domain.ApprovalStatusPendingCurrencyControl,
			"currency_control_decision":    "",
			"currency_control_comment":     "",
			"currency_control_reviewed_by": "",
			"currency_control_reviewed_at": nil,
			"compliance_decision":          "",
			"compliance_comment":           "",
			"compliance_reviewed_by":       "",
			"compliance_reviewed_at":       nil,
			"rejection_reason":             "",
			"updated_at":                   &now,
		}).Error
}