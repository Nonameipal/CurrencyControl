package repository

import (
	"context"
	"fmt"
	"strings"

	"CurrencyControl/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type InvoiceRepository interface {
	GetByContractID(ctx context.Context, contractID int64) ([]domain.InvoiceWithDetails, error)
	Create(ctx context.Context, inv domain.Invoice, contractCurrency string) (domain.Invoice, error)
	GetByID(ctx context.Context, id int64) (domain.Invoice, error)
	Update(ctx context.Context, id int64, inv domain.Invoice) (domain.Invoice, error)
	SoftDelete(ctx context.Context, id int64) error
}




type invoiceRepo struct{ db *pgxpool.Pool }



func NewInvoiceRepository(db *pgxpool.Pool) InvoiceRepository {
	return &invoiceRepo{db: db}
}


func (r *invoiceRepo) Create(ctx context.Context, inv domain.Invoice, contractCurrency string) (domain.Invoice, error) {
	var totalAmount float64
	var addlAmount float64
	var usedDeductAmount float64
	var dbContractCurrency string

	err := r.db.QueryRow(ctx, `
		SELECT 
			c.total_amount,
			c.contract_currency,
			COALESCE((SELECT SUM(aa.amount_in_contract_currency) FROM additional_agreements aa WHERE aa.contract_id = c.id), 0),
			COALESCE((SELECT SUM(i.deduct_amount) FROM invoices i WHERE i.contract_id = c.id), 0)
		FROM contracts c WHERE c.id = $1`,
		inv.ContractID,
	).Scan(&totalAmount, &dbContractCurrency, &addlAmount, &usedDeductAmount)
	if err != nil {
		return domain.Invoice{}, fmt.Errorf("")
	}

	effectiveLimit := totalAmount + addlAmount
	contractRemaining := effectiveLimit - usedDeductAmount

	if strings.EqualFold(inv.Currency, dbContractCurrency) && inv.DeductAmount == 0 {
		inv.DeductAmount = inv.Amount
	}

	if inv.DeductAmount <= 0 {
		return domain.Invoice{}, fmt.Errorf(
			"поле deduct_amount обязательно: валюта инвойса (%s) отличается от валюты контракта (%s). Укажите, сколько %s списать с баланса контракта",
			inv.Currency, dbContractCurrency, dbContractCurrency,
		)
	}

	if inv.DeductAmount > contractRemaining {
		return domain.Invoice{}, fmt.Errorf(
			"сумма списания (%.2f %s) превышает остаток по контракту (%.2f %s)",
			inv.DeductAmount, dbContractCurrency, contractRemaining, dbContractCurrency,
		)
	}

	query := `
		INSERT INTO invoices (contract_id, invoice_number, invoice_name, invoice_date, amount, currency, deduct_amount, document_path, original_document_name, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, contract_id, invoice_number, invoice_name, invoice_date, amount, currency, deduct_amount, document_path, original_document_name, COALESCE(created_by, ''), created_at, updated_at`

	var result domain.Invoice
	err = r.db.QueryRow(ctx, query,
		inv.ContractID, inv.InvoiceNumber, inv.InvoiceName, inv.InvoiceDate, inv.Amount, inv.Currency, inv.DeductAmount, inv.DocumentPath, inv.OriginalDocumentName, inv.CreatedBy,
	).Scan(
		&result.ID, &result.ContractID, &result.InvoiceNumber, &result.InvoiceName,
		&result.InvoiceDate, &result.Amount, &result.Currency, &result.DeductAmount, 
		&result.DocumentPath, &result.OriginalDocumentName, 
		&result.CreatedBy, &result.CreatedAt, &result.UpdatedAt,
	)
	return result, err
}

func (r *invoiceRepo) GetByID(ctx context.Context, id int64) (domain.Invoice, error) {
	query := `SELECT id, contract_id, invoice_number, invoice_name, invoice_date, amount, currency, deduct_amount, document_path, original_document_name, COALESCE(created_by, ''), created_at, updated_at FROM invoices WHERE id = $1 AND deleted_at IS NULL`
	var inv domain.Invoice
	err := r.db.QueryRow(ctx, query, id).Scan(
		&inv.ID, &inv.ContractID, &inv.InvoiceNumber, &inv.InvoiceName,
		&inv.InvoiceDate, &inv.Amount, &inv.Currency, &inv.DeductAmount,
		&inv.DocumentPath, &inv.OriginalDocumentName, &inv.CreatedBy, &inv.CreatedAt, &inv.UpdatedAt,
	)
	return inv, err
}

func (r *invoiceRepo) Update(ctx context.Context, id int64, inv domain.Invoice) (domain.Invoice, error) {
	query := `
		UPDATE invoices SET
			invoice_number = $2, invoice_name = $3, invoice_date = $4,
			amount = $5, currency = $6, deduct_amount = $7,
			document_path = $8, original_document_name = $9, updated_at = NOW()
		WHERE id = $1
		RETURNING id, contract_id, invoice_number, invoice_name, invoice_date, amount, currency, deduct_amount, document_path, original_document_name, COALESCE(created_by, ''), created_at, updated_at`
	var result domain.Invoice
	err := r.db.QueryRow(ctx, query,
		id, inv.InvoiceNumber, inv.InvoiceName, inv.InvoiceDate,
		inv.Amount, inv.Currency, inv.DeductAmount,
		inv.DocumentPath, inv.OriginalDocumentName,
	).Scan(
		&result.ID, &result.ContractID, &result.InvoiceNumber, &result.InvoiceName,
		&result.InvoiceDate, &result.Amount, &result.Currency, &result.DeductAmount,
		&result.DocumentPath, &result.OriginalDocumentName, &result.CreatedBy, &result.CreatedAt, &result.UpdatedAt,
	)
	return result, err
}

func (r *invoiceRepo) SoftDelete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `UPDATE invoices SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}


func (r *invoiceRepo) GetByContractID(ctx context.Context, contractID int64) ([]domain.InvoiceWithDetails, error) {
	query := `SELECT id, contract_id, invoice_number, invoice_name, invoice_date, amount, currency, deduct_amount, document_path, original_document_name, COALESCE(created_by, ''), created_at, updated_at FROM invoices WHERE contract_id = $1 AND deleted_at IS NULL ORDER BY invoice_date ASC`

	rows, err := r.db.Query(ctx, query, contractID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.InvoiceWithDetails
	for rows.Next() {
		var inv domain.InvoiceWithDetails
		if err := rows.Scan(
			&inv.ID, &inv.ContractID, &inv.InvoiceNumber, &inv.InvoiceName,
			&inv.InvoiceDate, &inv.Amount, &inv.Currency, &inv.DeductAmount,
			&inv.DocumentPath, &inv.OriginalDocumentName, &inv.CreatedBy, &inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, err
		}


		gtdData, _ := (&gtdRepo{db: r.db}).GetByInvoiceID(ctx, inv.ID)
		if gtdData != nil {
			inv.GTD = gtdData
			inv.InvoiceRemaining = inv.Amount - gtdData.ClosesAmount
		} else {
			inv.InvoiceRemaining = inv.Amount
		}
		result = append(result, inv)
	}
	if result == nil {
		result = []domain.InvoiceWithDetails{}
	}
	return result, nil
}


