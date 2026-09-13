package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type invoiceRepo struct{ db *pgxpool.Pool }

func NewInvoiceRepository(db *pgxpool.Pool) ports.InvoiceRepository {
	return &invoiceRepo{db: db}
}
func (r *invoiceRepo) Create(ctx context.Context, inv domain.Invoice, contractCurrency string) (domain.Invoice, error) {
	if inv.AdditionalAgreementID == nil {
		var contractTotal float64
		var dbContractCurrency string
		var usedDeductAmount float64
		var extendDateTo *time.Time
		var deliveryDate *time.Time

		err := r.db.QueryRow(ctx, `
			SELECT 
				c.total_amount,
				c.contract_currency,
				COALESCE((SELECT SUM(i.deduct_amount) FROM invoices i WHERE i.contract_id = c.id AND i.additional_agreement_id IS NULL AND i.deleted_at IS NULL), 0),
				c.extend_date_to,
				c.delivery_date
			FROM contracts c WHERE c.id = $1 AND c.deleted_at IS NULL`,
			inv.ContractID,
		).Scan(&contractTotal, &dbContractCurrency, &usedDeductAmount, &extendDateTo, &deliveryDate)
		if err != nil {
			return domain.Invoice{}, fmt.Errorf("контракт не найден: %w", err)
		}

		// Rule 1: block new invoice if contract has expired
		today := time.Now().UTC().Truncate(24 * time.Hour)
		var expiryDate *time.Time
		if extendDateTo != nil && !extendDateTo.IsZero() {
			expiryDate = extendDateTo
		} else if deliveryDate != nil && !deliveryDate.IsZero() {
			expiryDate = deliveryDate
		}
		if expiryDate != nil && !expiryDate.IsZero() {
			exp := time.Date(expiryDate.Year(), expiryDate.Month(), expiryDate.Day(), 0, 0, 0, 0, time.UTC)
			if today.After(exp) {
				return domain.Invoice{}, fmt.Errorf("срок действия контракта истёк (%s). Добавление инвойсов запрещено", exp.Format("02.01.2006"))
			}
		}

		if !strings.EqualFold(inv.Currency, dbContractCurrency) {
			return domain.Invoice{}, fmt.Errorf("валюта инвойса (%s) должна совпадать с валютой контракта (%s)", inv.Currency, dbContractCurrency)
		}

		contractRemaining := contractTotal - usedDeductAmount
		inv.DeductAmount = inv.Amount

		if inv.Amount > contractRemaining {
			return domain.Invoice{}, fmt.Errorf(
				"сумма инвойса (%.2f %s) превышает остаток по контракту (%.2f %s)",
				inv.Amount, dbContractCurrency, contractRemaining, dbContractCurrency,
			)
		}
	} else {
		var addlTotal float64
		var dbAddlCurrency string
		var usedAmount float64

		err := r.db.QueryRow(ctx, `
			SELECT 
				COALESCE(aa.foreign_amount, 0),
				COALESCE(aa.currency, ''),
				COALESCE((SELECT SUM(i.amount) FROM invoices i WHERE i.additional_agreement_id = aa.id AND i.deleted_at IS NULL), 0)
			FROM additional_agreements aa 
			WHERE aa.id = $1 AND aa.contract_id = $2 AND aa.deleted_at IS NULL`,
			*inv.AdditionalAgreementID, inv.ContractID,
		).Scan(&addlTotal, &dbAddlCurrency, &usedAmount)
		if err != nil {
			return domain.Invoice{}, fmt.Errorf("дополнительное соглашение не найдено: %w", err)
		}

		if !strings.EqualFold(inv.Currency, dbAddlCurrency) {
			return domain.Invoice{}, fmt.Errorf("валюта инвойса (%s) должна совпадать с валютой доп. соглашения (%s)", inv.Currency, dbAddlCurrency)
		}

		addlRemaining := addlTotal - usedAmount
		inv.DeductAmount = inv.Amount

		if inv.Amount > addlRemaining {
			return domain.Invoice{}, fmt.Errorf(
				"сумма инвойса (%.2f %s) превышает остаток по доп. соглашению (%.2f %s)",
				inv.Amount, dbAddlCurrency, addlRemaining, dbAddlCurrency,
			)
		}
	}

	if inv.ApprovalStatus == "" {
		inv.ApprovalStatus = domain.ApprovalStatusPendingCurrencyControl
	}

	query := `
		INSERT INTO invoices (contract_id, additional_agreement_id, invoice_number, invoice_date, amount, currency, hs_code, deduct_amount, document_path, created_by, approval_status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, contract_id, additional_agreement_id, invoice_number, invoice_date, amount, currency, COALESCE(hs_code, ''), deduct_amount, document_path, COALESCE(created_by, ''), COALESCE(approval_status, 'pending_currency_control'), created_at, updated_at`

	var result domain.Invoice
	err := r.db.QueryRow(ctx, query,
		inv.ContractID, inv.AdditionalAgreementID, inv.InvoiceNumber, inv.InvoiceDate, inv.Amount, inv.Currency, inv.HSCode, inv.DeductAmount, inv.DocumentPath, inv.CreatedBy, inv.ApprovalStatus,
	).Scan(
		&result.ID, &result.ContractID, &result.AdditionalAgreementID, &result.InvoiceNumber,
		&result.InvoiceDate, &result.Amount, &result.Currency, &result.HSCode, &result.DeductAmount, 
		&result.DocumentPath, 
		&result.CreatedBy, &result.ApprovalStatus, &result.CreatedAt, &result.UpdatedAt,
	)
	if err == nil {
		if result.AdditionalAgreementID != nil {
			syncAdditionalAgreementRemaining(ctx, r.db, *result.AdditionalAgreementID)
			tryArchiveAdditionalAgreement(ctx, r.db, *result.AdditionalAgreementID)
		} else {
			syncContractRemaining(ctx, r.db, result.ContractID)
			tryArchiveContract(ctx, r.db, result.ContractID)
		}
	}
	return result, err
}

func (r *invoiceRepo) GetByID(ctx context.Context, id int64) (domain.Invoice, error) {
	query := `SELECT id, contract_id, additional_agreement_id, invoice_number, invoice_date, amount, currency, COALESCE(hs_code, ''), deduct_amount, document_path, COALESCE(created_by, ''), COALESCE(approval_status, 'pending_currency_control'), created_at, updated_at FROM invoices WHERE id = $1 AND deleted_at IS NULL`
	var inv domain.Invoice
	err := r.db.QueryRow(ctx, query, id).Scan(
		&inv.ID, &inv.ContractID, &inv.AdditionalAgreementID, &inv.InvoiceNumber,
		&inv.InvoiceDate, &inv.Amount, &inv.Currency, &inv.HSCode, &inv.DeductAmount,
		&inv.DocumentPath, &inv.CreatedBy, &inv.ApprovalStatus, &inv.CreatedAt, &inv.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Invoice{}, errors.New("Инвойс не найден")
		}
		return inv, err
	}
	return inv, nil
}

func (r *invoiceRepo) Update(ctx context.Context, id int64, inv domain.Invoice) (domain.Invoice, error) {
	var currentContractID int64
	var currentAddlID *int64
	var expectedCurrency string
	err := r.db.QueryRow(ctx, `SELECT contract_id, additional_agreement_id FROM invoices WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&currentContractID, &currentAddlID)
	if err != nil {
		return domain.Invoice{}, fmt.Errorf("Инвойс не найден")
	}
	if currentAddlID != nil {
		_ = r.db.QueryRow(ctx, `SELECT currency FROM additional_agreements WHERE id = $1`, *currentAddlID).Scan(&expectedCurrency)
	} else {
		_ = r.db.QueryRow(ctx, `SELECT currency FROM contracts WHERE id = $1`, currentContractID).Scan(&expectedCurrency)
	}
	if inv.Currency != "" && expectedCurrency != "" && !strings.EqualFold(inv.Currency, expectedCurrency) {
		return domain.Invoice{}, fmt.Errorf("валюта инвойса (%s) должна совпадать с валютой документа (%s)", inv.Currency, expectedCurrency)
	}

	if inv.DeductAmount <= 0 {
		inv.DeductAmount = inv.Amount
	}

	query := `
		UPDATE invoices SET
			invoice_number = $2, invoice_date = $3,
			amount = $4, currency = $5, hs_code = $6, deduct_amount = $7,
			document_path = $8,
			approval_status = 'pending_currency_control',
			rejection_reason = '',
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, contract_id, additional_agreement_id, invoice_number, invoice_date, amount, currency, COALESCE(hs_code, ''), deduct_amount, document_path, COALESCE(created_by, ''), COALESCE(approval_status, 'pending_currency_control'), created_at, updated_at`
	var result domain.Invoice
	err = r.db.QueryRow(ctx, query,
		id, inv.InvoiceNumber, inv.InvoiceDate,
		inv.Amount, inv.Currency, inv.HSCode, inv.DeductAmount,
		inv.DocumentPath,
	).Scan(
		&result.ID, &result.ContractID, &result.AdditionalAgreementID, &result.InvoiceNumber,
		&result.InvoiceDate, &result.Amount, &result.Currency, &result.HSCode, &result.DeductAmount,
		&result.DocumentPath, &result.CreatedBy, &result.ApprovalStatus, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Invoice{}, errors.New("Инвойс не найден")
		}
		return domain.Invoice{}, err
	}
	if result.AdditionalAgreementID != nil {
		syncAdditionalAgreementRemaining(ctx, r.db, *result.AdditionalAgreementID)
		tryArchiveAdditionalAgreement(ctx, r.db, *result.AdditionalAgreementID)
	} else {
		syncContractRemaining(ctx, r.db, result.ContractID)
		tryArchiveContract(ctx, r.db, result.ContractID)
	}
	return result, nil
}

func (r *invoiceRepo) SoftDelete(ctx context.Context, id int64) error {
	var contractID int64
	var addlID *int64
	err := r.db.QueryRow(ctx, `SELECT contract_id, additional_agreement_id FROM invoices WHERE id = $1 AND deleted_at IS NULL`, id).Scan(&contractID, &addlID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("Инвойс не найден")
		}
		return err
	}
	cmdTag, err := r.db.Exec(ctx, `UPDATE invoices SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return errors.New("Инвойс не найден")
	}
	if addlID != nil {
		syncAdditionalAgreementRemaining(ctx, r.db, *addlID)
		tryArchiveAdditionalAgreement(ctx, r.db, *addlID)
	} else if contractID > 0 {
		syncContractRemaining(ctx, r.db, contractID)
		tryArchiveContract(ctx, r.db, contractID)
	}
	return nil
}

func syncContractRemaining(ctx context.Context, db *pgxpool.Pool, contractID int64) {
	_, _ = db.Exec(ctx, `
		UPDATE contracts
		SET remaining_amount = total_amount 
			- COALESCE((SELECT SUM(deduct_amount) FROM invoices WHERE contract_id = contracts.id AND additional_agreement_id IS NULL AND deleted_at IS NULL), 0),
			updated_at = NOW()
		WHERE id = $1`, contractID,
	)
}


func (r *invoiceRepo) GetByContractID(ctx context.Context, contractID int64) ([]domain.InvoiceWithDetails, error) {
	query := `SELECT id, contract_id, additional_agreement_id, invoice_number, invoice_date, amount, currency, COALESCE(hs_code, ''), deduct_amount, document_path, COALESCE(created_by, ''), created_at, updated_at FROM invoices WHERE contract_id = $1 AND deleted_at IS NULL ORDER BY invoice_date ASC`

	rows, err := r.db.Query(ctx, query, contractID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.InvoiceWithDetails
	for rows.Next() {
		var inv domain.InvoiceWithDetails
		if err := rows.Scan(
			&inv.ID, &inv.ContractID, &inv.AdditionalAgreementID, &inv.InvoiceNumber,
			&inv.InvoiceDate, &inv.Amount, &inv.Currency, &inv.HSCode, &inv.DeductAmount,
			&inv.DocumentPath, &inv.CreatedBy, &inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, err
		}

		gtdData, _ := (&gtdRepo{db: r.db}).GetByInvoiceID(ctx, inv.ID)
		if gtdData != nil {
			inv.GTD = gtdData
		}
		result = append(result, inv)
	}
	if result == nil {
		result = []domain.InvoiceWithDetails{}
	}
	return result, nil
}

func (r *invoiceRepo) GetByAdditionalAgreementID(ctx context.Context, agreementID int64) ([]domain.InvoiceWithDetails, error) {
	query := `SELECT id, contract_id, additional_agreement_id, invoice_number, invoice_date, amount, currency, COALESCE(hs_code, ''), deduct_amount, document_path, COALESCE(created_by, ''), created_at, updated_at FROM invoices WHERE additional_agreement_id = $1 AND deleted_at IS NULL ORDER BY invoice_date ASC`

	rows, err := r.db.Query(ctx, query, agreementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.InvoiceWithDetails
	for rows.Next() {
		var inv domain.InvoiceWithDetails
		if err := rows.Scan(
			&inv.ID, &inv.ContractID, &inv.AdditionalAgreementID, &inv.InvoiceNumber,
			&inv.InvoiceDate, &inv.Amount, &inv.Currency, &inv.HSCode, &inv.DeductAmount,
			&inv.DocumentPath, &inv.CreatedBy, &inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, err
		}

		gtdData, _ := (&gtdRepo{db: r.db}).GetByInvoiceID(ctx, inv.ID)
		if gtdData != nil {
			inv.GTD = gtdData
		}
		result = append(result, inv)
	}
	if result == nil {
		result = []domain.InvoiceWithDetails{}
	}
	return result, nil
}


