package repository

import (
	"context"
	"time"

	"CurrencyControl/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type InvoiceRepository interface {
	GetByContractID(ctx context.Context, contractID int64) ([]domain.InvoiceWithDetails, error)
	Create(ctx context.Context, inv domain.Invoice) (domain.Invoice, error)
	GetByID(ctx context.Context, id int64) (domain.Invoice, error)
}

type GTDRepository interface {
	Create(ctx context.Context, gtd domain.GTD) (domain.GTD, error)
	GetByInvoiceID(ctx context.Context, invoiceID int64) (*domain.GTD, error)
}

type invoiceRepo struct{ db *pgxpool.Pool }
type gtdRepo struct{ db *pgxpool.Pool }

func NewInvoiceRepository(db *pgxpool.Pool) InvoiceRepository {
	return &invoiceRepo{db: db}
}

func NewGTDRepository(db *pgxpool.Pool) GTDRepository {
	return &gtdRepo{db: db}
}

func (r *invoiceRepo) Create(ctx context.Context, inv domain.Invoice) (domain.Invoice, error) {
	query := `
		INSERT INTO invoices (contract_id, invoice_number, invoice_name, invoice_date, amount, currency)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, contract_id, invoice_number, invoice_name, invoice_date, amount, currency, created_at, updated_at`

	var result domain.Invoice
	err := r.db.QueryRow(ctx, query,
		inv.ContractID, inv.InvoiceNumber, inv.InvoiceName, inv.InvoiceDate, inv.Amount, inv.Currency,
	).Scan(
		&result.ID, &result.ContractID, &result.InvoiceNumber, &result.InvoiceName,
		&result.InvoiceDate, &result.Amount, &result.Currency, &result.CreatedAt, &result.UpdatedAt,
	)
	return result, err
}

func (r *invoiceRepo) GetByID(ctx context.Context, id int64) (domain.Invoice, error) {
	query := `SELECT id, contract_id, invoice_number, invoice_name, invoice_date, amount, currency, created_at, updated_at FROM invoices WHERE id = $1`
	var inv domain.Invoice
	err := r.db.QueryRow(ctx, query, id).Scan(
		&inv.ID, &inv.ContractID, &inv.InvoiceNumber, &inv.InvoiceName,
		&inv.InvoiceDate, &inv.Amount, &inv.Currency, &inv.CreatedAt, &inv.UpdatedAt,
	)
	return inv, err
}

func (r *invoiceRepo) GetByContractID(ctx context.Context, contractID int64) ([]domain.InvoiceWithDetails, error) {
	query := `SELECT id, contract_id, invoice_number, invoice_name, invoice_date, amount, currency, created_at, updated_at FROM invoices WHERE contract_id = $1 ORDER BY invoice_date ASC`

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
			&inv.InvoiceDate, &inv.Amount, &inv.Currency, &inv.CreatedAt, &inv.UpdatedAt,
		); err != nil {
			return nil, err
		}

		gtdData, _ := (&gtdRepo{db: r.db}).GetByInvoiceID(ctx, inv.ID)
		if gtdData != nil {
			inv.GTD = gtdData
		}

		var docID int64
		var docName string
		docErr := r.db.QueryRow(ctx,
			`SELECT id, original_name FROM documents WHERE entity_type = 'invoice' AND entity_id = $1 LIMIT 1`,
			inv.ID,
		).Scan(&docID, &docName)
		if docErr == nil {
			inv.Document = &domain.Document{ID: docID, OriginalName: docName}
		}

		result = append(result, inv)
	}
	if result == nil {
		result = []domain.InvoiceWithDetails{}
	}
	return result, nil
}

func (r *gtdRepo) Create(ctx context.Context, g domain.GTD) (domain.GTD, error) {
	query := `
		INSERT INTO gtd (invoice_id, gtd_number, gtd_amount, gtd_date)
		VALUES ($1, $2, $3, $4)
		RETURNING id, invoice_id, gtd_number, gtd_amount, gtd_date, created_at, updated_at`

	var result domain.GTD
	err := r.db.QueryRow(ctx, query,
		g.InvoiceID, g.GTDNumber, g.GTDAmount, g.GTDDate,
	).Scan(
		&result.ID, &result.InvoiceID, &result.GTDNumber, &result.GTDAmount,
		&result.GTDDate, &result.CreatedAt, &result.UpdatedAt,
	)
	return result, err
}

func (r *gtdRepo) GetByInvoiceID(ctx context.Context, invoiceID int64) (*domain.GTD, error) {
	query := `SELECT id, invoice_id, gtd_number, gtd_amount, gtd_date, created_at, updated_at FROM gtd WHERE invoice_id = $1 LIMIT 1`
	var g domain.GTD
	err := r.db.QueryRow(ctx, query, invoiceID).Scan(
		&g.ID, &g.InvoiceID, &g.GTDNumber, &g.GTDAmount,
		&g.GTDDate, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		return nil, nil
	}
	_ = time.Now()
	return &g, nil
}