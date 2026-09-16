package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type paymentOrderRepo struct {
	db *pgxpool.Pool
}

func NewPaymentOrderRepository(db *pgxpool.Pool) ports.PaymentOrderRepository {
	return &paymentOrderRepo{db: db}
}

const paymentOrderSelectCols = `
	id, contract_id, additional_agreement_id, invoice_id, operation_date,
	payment_order_number, amount, currency, payer, receiver_name,
	receiver_bank, payment_purpose, receiver_country, contract_number,
	invoice_number, value_date, created_by, document_path, created_at, updated_at
`

func scanPaymentOrder(row pgx.Row) (*domain.PaymentOrder, error) {
	var po domain.PaymentOrder
	err := row.Scan(
		&po.ID,
		&po.ContractID,
		&po.AdditionalAgreementID,
		&po.InvoiceID,
		&po.OperationDate,
		&po.PaymentOrderNumber,
		&po.Amount,
		&po.Currency,
		&po.Payer,
		&po.ReceiverName,
		&po.ReceiverBank,
		&po.PaymentPurpose,
		&po.ReceiverCountry,
		&po.ContractNumber,
		&po.InvoiceNumber,
		&po.ValueDate,
		&po.CreatedBy,
		&po.DocumentPath,
		&po.CreatedAt,
		&po.UpdatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &po, nil
}

func (r *paymentOrderRepo) Create(ctx context.Context, po domain.PaymentOrder) (domain.PaymentOrder, error) {
	if po.InvoiceID <= 0 {
		return domain.PaymentOrder{}, fmt.Errorf("поле invoice_id обязательно")
	}
	if po.Amount <= 0 {
		return domain.PaymentOrder{}, fmt.Errorf("сумма платежного поручения должна быть больше 0")
	}

	var invoiceAmount float64
	var alreadyPaid float64
	var invoiceNumber string
	var invoiceCurrency string
	var contractID int64
	var invoiceAddlID *int64
	var contractNumber string

	queryInvoice := `
		SELECT i.amount,
		       COALESCE((SELECT SUM(p.amount) FROM payment_orders p WHERE p.invoice_id = i.id AND p.deleted_at IS NULL), 0),
		       i.invoice_number, i.currency, i.contract_id, i.additional_agreement_id, c.contract_number
		FROM invoices i
		JOIN contracts c ON c.id = i.contract_id
		WHERE i.id = $1 AND i.deleted_at IS NULL`

	err := r.db.QueryRow(ctx, queryInvoice, po.InvoiceID).Scan(
		&invoiceAmount, &alreadyPaid, &invoiceNumber, &invoiceCurrency, &contractID, &invoiceAddlID, &contractNumber,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.PaymentOrder{}, fmt.Errorf("инвойс не найден")
		}
		return domain.PaymentOrder{}, fmt.Errorf("ошибка проверки инвойса: %w", err)
	}

	if !strings.EqualFold(strings.TrimSpace(po.Currency), strings.TrimSpace(invoiceCurrency)) {
		return domain.PaymentOrder{}, fmt.Errorf("валюта платежного поручения (%s) должна совпадать с валютой инвойса (%s)", po.Currency, invoiceCurrency)
	}
	remainingPayment := invoiceAmount - alreadyPaid
	if po.Amount > remainingPayment {
		return domain.PaymentOrder{}, fmt.Errorf(
			"сумма платежного поручения (%.2f %s) превышает доступный остаток по оплате инвойса (%.2f %s)",
			po.Amount, invoiceCurrency, remainingPayment, invoiceCurrency,
		)
	}

	po.ContractID = contractID
	po.AdditionalAgreementID = invoiceAddlID
	po.InvoiceNumber = invoiceNumber
	po.ContractNumber = contractNumber
	po.Currency = strings.ToUpper(strings.TrimSpace(po.Currency))

	queryInsert := fmt.Sprintf(`
		INSERT INTO payment_orders (
			contract_id, additional_agreement_id, invoice_id, operation_date,
			payment_order_number, amount, currency, payer, receiver_name,
			receiver_bank, payment_purpose, receiver_country, contract_number,
			invoice_number, value_date, created_by, document_path
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		RETURNING %s`, paymentOrderSelectCols)

	row := r.db.QueryRow(ctx, queryInsert,
		po.ContractID, po.AdditionalAgreementID, po.InvoiceID, po.OperationDate,
		po.PaymentOrderNumber, po.Amount, po.Currency, po.Payer, po.ReceiverName,
		po.ReceiverBank, po.PaymentPurpose, po.ReceiverCountry, po.ContractNumber,
		po.InvoiceNumber, po.ValueDate, po.CreatedBy, po.DocumentPath,
	)

	created, err := scanPaymentOrder(row)
	if err != nil {
		return domain.PaymentOrder{}, fmt.Errorf("ошибка сохранения платежного поручения: %w", err)
	}
	return *created, nil
}

func (r *paymentOrderRepo) GetByID(ctx context.Context, id int64) (*domain.PaymentOrder, error) {
	query := fmt.Sprintf(`SELECT %s FROM payment_orders WHERE id = $1 AND deleted_at IS NULL`, paymentOrderSelectCols)
	row := r.db.QueryRow(ctx, query, id)
	po, err := scanPaymentOrder(row)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, fmt.Errorf("платежное поручение не найдено")
		}
		return nil, err
	}
	return po, nil
}

func (r *paymentOrderRepo) GetByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.PaymentOrder, error) {
	query := fmt.Sprintf(`SELECT %s FROM payment_orders WHERE invoice_id = $1 AND deleted_at IS NULL ORDER BY operation_date DESC, id DESC`, paymentOrderSelectCols)
	rows, err := r.db.Query(ctx, query, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.PaymentOrder
	for rows.Next() {
		po, err := scanPaymentOrder(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *po)
	}
	if list == nil {
		list = []domain.PaymentOrder{}
	}
	return list, nil
}

func (r *paymentOrderRepo) GetByContractID(ctx context.Context, contractID int64) ([]domain.PaymentOrder, error) {
	query := fmt.Sprintf(`SELECT %s FROM payment_orders WHERE contract_id = $1 AND deleted_at IS NULL ORDER BY operation_date DESC, id DESC`, paymentOrderSelectCols)
	rows, err := r.db.Query(ctx, query, contractID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.PaymentOrder
	for rows.Next() {
		po, err := scanPaymentOrder(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, *po)
	}
	if list == nil {
		list = []domain.PaymentOrder{}
	}
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

	var invoiceAmount float64
	var alreadyPaidOther float64
	var invoiceCurrency string

	queryInvoice := `
		SELECT i.amount,
		       COALESCE((SELECT SUM(p.amount) FROM payment_orders p WHERE p.invoice_id = i.id AND p.id <> $1 AND p.deleted_at IS NULL), 0),
		       i.currency
		FROM invoices i
		WHERE i.id = $2 AND i.deleted_at IS NULL`

	err = r.db.QueryRow(ctx, queryInvoice, id, existing.InvoiceID).Scan(&invoiceAmount, &alreadyPaidOther, &invoiceCurrency)
	if err != nil {
		return nil, fmt.Errorf("ошибка проверки инвойса: %w", err)
	}

	if po.Currency != "" && !strings.EqualFold(strings.TrimSpace(po.Currency), strings.TrimSpace(invoiceCurrency)) {
		return nil, fmt.Errorf("валюта платежного поручения (%s) должна совпадать с валютой инвойса (%s)", po.Currency, invoiceCurrency)
	}

	remainingPayment := invoiceAmount - alreadyPaidOther
	if po.Amount > remainingPayment {
		return nil, fmt.Errorf(
			"сумма платежного поручения (%.2f %s) превышает доступный остаток по оплате инвойса (%.2f %s)",
			po.Amount, invoiceCurrency, remainingPayment, invoiceCurrency,
		)
	}

	queryUpdate := fmt.Sprintf(`
		UPDATE payment_orders
		SET operation_date = $1,
		    payment_order_number = $2,
		    amount = $3,
		    currency = $4,
		    payer = $5,
		    receiver_name = $6,
		    receiver_bank = $7,
		    payment_purpose = $8,
		    receiver_country = $9,
		    value_date = $10,
		    document_path = $11,
		    updated_at = NOW()
		WHERE id = $12 AND deleted_at IS NULL
		RETURNING %s`, paymentOrderSelectCols)

	curr := po.Currency
	if curr == "" {
		curr = existing.Currency
	}

	docPath := existing.DocumentPath
	if po.DocumentPath != nil {
		docPath = po.DocumentPath
	}

	row := r.db.QueryRow(ctx, queryUpdate,
		po.OperationDate, po.PaymentOrderNumber, po.Amount, strings.ToUpper(curr),
		po.Payer, po.ReceiverName, po.ReceiverBank, po.PaymentPurpose,
		po.ReceiverCountry, po.ValueDate, docPath, id,
	)

	updated, err := scanPaymentOrder(row)
	if err != nil {
		return nil, fmt.Errorf("ошибка обновления платежного поручения: %w", err)
	}
	return updated, nil
}

func (r *paymentOrderRepo) SoftDelete(ctx context.Context, id int64) error {
	query := `UPDATE payment_orders SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`
	cmd, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return fmt.Errorf("платежное поручение не найдено или уже удалено")
	}
	return nil
}
