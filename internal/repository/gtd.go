package repository

import (
	"context"
	"fmt"

	"CurrencyControl/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type GTDRepository interface {
	Create(ctx context.Context, g domain.GTD) (domain.GTD, error)
	GetByInvoiceID(ctx context.Context, invoiceID int64) (*domain.GTD, error)
	Update(ctx context.Context, id int64, g domain.GTD) (domain.GTD, error)
	SoftDelete(ctx context.Context, id int64) error
}

func NewGTDRepository(db *pgxpool.Pool) GTDRepository {
	return &gtdRepo{db: db}
}

type gtdRepo struct{ db *pgxpool.Pool }

func (r *gtdRepo) Create(ctx context.Context, g domain.GTD) (domain.GTD, error) {
	var invoiceAmount float64
	var alreadyClosed float64
	err := r.db.QueryRow(ctx, `
		SELECT i.amount, COALESCE(SUM(gtd.closes_amount), 0)
		FROM invoices i
		LEFT JOIN gtd ON gtd.invoice_id = i.id
		WHERE i.id = $1
		GROUP BY i.amount`, g.InvoiceID,
	).Scan(&invoiceAmount, &alreadyClosed)
	if err != nil {
		return domain.GTD{}, fmt.Errorf("инвойс не найден")
	}

	invoiceRemaining := invoiceAmount - alreadyClosed
	if g.ClosesAmount > invoiceRemaining {
		return domain.GTD{}, fmt.Errorf(
			"сумма закрытия по ГТД (%.2f) превышает товарный остаток по инвойсу (%.2f)",
			g.ClosesAmount, invoiceRemaining,
		)
	}

	if g.DocumentType == "" {
		g.DocumentType = domain.DocumentTypeGTD
	}

	query := `
		INSERT INTO gtd (contract_id, invoice_id, document_type, gtd_number, gtd_amount, gtd_currency, gtd_date, closes_amount, document_path, original_document_name, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING id, contract_id, invoice_id, COALESCE(document_type, 'gtd'), gtd_number, gtd_amount, gtd_currency, gtd_date, closes_amount, document_path, original_document_name, COALESCE(created_by, ''), created_at, updated_at`

	var result domain.GTD
	err = r.db.QueryRow(ctx, query,
		g.ContractID, g.InvoiceID, g.DocumentType, g.GTDNumber, g.GTDAmount, g.GTDCurrency, g.GTDDate, g.ClosesAmount, g.DocumentPath, g.OriginalDocumentName, g.CreatedBy,
	).Scan(
		&result.ID, &result.ContractID, &result.InvoiceID, &result.DocumentType, &result.GTDNumber, &result.GTDAmount,
		&result.GTDCurrency, &result.GTDDate, &result.ClosesAmount,
		&result.DocumentPath, &result.OriginalDocumentName, &result.CreatedBy, &result.CreatedAt, &result.UpdatedAt,
	)
	return result, err
}

func (r *gtdRepo) GetByInvoiceID(ctx context.Context, invoiceID int64) (*domain.GTD, error) {
	query := `SELECT id, contract_id, invoice_id, COALESCE(document_type, 'gtd'), gtd_number, gtd_amount, gtd_currency, gtd_date, closes_amount, document_path, original_document_name, COALESCE(created_by, ''), created_at, updated_at FROM gtd WHERE invoice_id = $1 AND deleted_at IS NULL LIMIT 1`
	var g domain.GTD
	err := r.db.QueryRow(ctx, query, invoiceID).Scan(
		&g.ID, &g.ContractID, &g.InvoiceID, &g.DocumentType, &g.GTDNumber, &g.GTDAmount,
		&g.GTDCurrency, &g.GTDDate, &g.ClosesAmount,
		&g.DocumentPath, &g.OriginalDocumentName, &g.CreatedBy, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		return nil, nil
	}
	return &g, nil
}

func (r *gtdRepo) SoftDelete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `UPDATE gtd SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}

func (r *gtdRepo) Update(ctx context.Context, id int64, g domain.GTD) (domain.GTD, error) {
	if g.DocumentType == "" {
		g.DocumentType = domain.DocumentTypeGTD
	}
	query := `
		UPDATE gtd SET
			document_type = $2, gtd_number = $3, gtd_amount = $4, gtd_currency = $5, gtd_date = $6,
			closes_amount = $7, document_path = $8, original_document_name = $9, updated_at = NOW()
		WHERE id = $1
		RETURNING id, contract_id, invoice_id, COALESCE(document_type, 'gtd'), gtd_number, gtd_amount, gtd_currency, gtd_date, closes_amount, document_path, original_document_name, COALESCE(created_by, ''), created_at, updated_at`

	var result domain.GTD
	err := r.db.QueryRow(ctx, query,
		id, g.DocumentType, g.GTDNumber, g.GTDAmount, g.GTDCurrency, g.GTDDate, g.ClosesAmount, g.DocumentPath, g.OriginalDocumentName,
	).Scan(
		&result.ID, &result.ContractID, &result.InvoiceID, &result.DocumentType, &result.GTDNumber, &result.GTDAmount,
		&result.GTDCurrency, &result.GTDDate, &result.ClosesAmount,
		&result.DocumentPath, &result.OriginalDocumentName, &result.CreatedBy, &result.CreatedAt, &result.UpdatedAt,
	)
	return result, err
}



