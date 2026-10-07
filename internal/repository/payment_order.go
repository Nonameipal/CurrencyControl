package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

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

	var contract domain.Contract
	if err := r.db.WithContext(ctx).Select("contract_number").First(&contract, inv.ContractID).Error; err != nil {
		return domain.PaymentOrder{}, fmt.Errorf("контракт не найден: %w", err)
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
	po.AdditionalAgreementID = inv.AdditionalAgreementID
	po.InvoiceNumber = inv.InvoiceNumber
	po.ContractNumber = contract.ContractNumber
	po.Currency = strings.ToUpper(strings.TrimSpace(po.Currency))

	if err := r.db.WithContext(ctx).Create(&po).Error; err != nil {
		return domain.PaymentOrder{}, fmt.Errorf("ошибка сохранения платежного поручения: %w", err)
	}
	tryArchivePaymentOrderEntity(ctx, r.db, po.ContractID, po.AdditionalAgreementID)
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
		Order("operation_date DESC, id DESC").
		Find(&list).Error; err != nil {
		return nil, err
	}
	enrichPaymentOrders(ctx, r.db, list)
	return list, nil
}

func (r *paymentOrderRepo) GetByContractID(ctx context.Context, contractID int64) ([]domain.PaymentOrder, error) {
	var list []domain.PaymentOrder
	if err := r.db.WithContext(ctx).
		Where("contract_id = ? AND additional_agreement_id IS NULL", contractID).
		Order("operation_date DESC, id DESC").
		Find(&list).Error; err != nil {
		return nil, err
	}
	enrichPaymentOrders(ctx, r.db, list)
	return list, nil
}

func (r *paymentOrderRepo) GetByAdditionalAgreementID(ctx context.Context, agreementID int64) ([]domain.PaymentOrder, error) {
	var list []domain.PaymentOrder
	if err := r.db.WithContext(ctx).
		Where("additional_agreement_id = ?", agreementID).
		Order("operation_date DESC, id DESC").
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
		"operation_date":       po.OperationDate,
		"payment_order_number": po.PaymentOrderNumber,
		"amount":               po.Amount,
		"currency":             strings.ToUpper(curr),
		"payer":                po.Payer,
		"receiver_name":        po.ReceiverName,
		"receiver_bank":        po.ReceiverBank,
		"payment_purpose":      po.PaymentPurpose,
		"receiver_country":     po.ReceiverCountry,
		"sender_name":          po.SenderName,
		"sender_bank":          po.SenderBank,
		"sender_country":       po.SenderCountry,
		"value_date":           po.ValueDate,
		"document_path":        docPath,
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

	tryArchivePaymentOrderEntity(ctx, r.db, existing.ContractID, existing.AdditionalAgreementID)

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

	tryArchivePaymentOrderEntity(ctx, r.db, existing.ContractID, existing.AdditionalAgreementID)

	return nil
}

func tryArchivePaymentOrderEntity(ctx context.Context, db *gorm.DB, contractID int64, addlID *int64) {
	if addlID != nil {
		tryArchiveAdditionalAgreementGorm(ctx, db, *addlID)
	} else if contractID > 0 {
		tryArchiveContractGorm(ctx, db, contractID)
	}
}
