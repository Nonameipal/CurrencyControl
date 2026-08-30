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
}

type GTDRepository interface {
	Create(ctx context.Context, g domain.GTD) (domain.GTD, error)
	GetByInvoiceID(ctx context.Context, invoiceID int64) (*domain.GTD, error)
}

type AdditionalAgreementRepository interface {
	Create(ctx context.Context, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error)
	GetByContractID(ctx context.Context, contractID int64) ([]domain.AdditionalAgreement, error)
}

type invoiceRepo struct{ db *pgxpool.Pool }
type gtdRepo struct{ db *pgxpool.Pool }
type additionalAgreementRepo struct{ db *pgxpool.Pool }

func NewInvoiceRepository(db *pgxpool.Pool) InvoiceRepository {
	return &invoiceRepo{db: db}
}
func NewGTDRepository(db *pgxpool.Pool) GTDRepository {
	return &gtdRepo{db: db}
}
func NewAdditionalAgreementRepository(db *pgxpool.Pool) AdditionalAgreementRepository {
	return &additionalAgreementRepo{db: db}
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
		return domain.Invoice{}, fmt.Errorf("РєРѕРЅС‚СЂР°РєС‚ РЅРµ РЅР°Р№РґРµРЅ")
	}

	effectiveLimit := totalAmount + addlAmount
	contractRemaining := effectiveLimit - usedDeductAmount

	if strings.EqualFold(inv.Currency, dbContractCurrency) && inv.DeductAmount == 0 {
		inv.DeductAmount = inv.Amount
	}

	if inv.DeductAmount <= 0 {
		return domain.Invoice{}, fmt.Errorf(
			"РїРѕР»Рµ deduct_amount РѕР±СЏР·Р°С‚РµР»СЊРЅРѕ: РІР°Р»СЋС‚Р° РёРЅРІРѕР№СЃР° (%s) РѕС‚Р»РёС‡Р°РµС‚СЃСЏ РѕС‚ РІР°Р»СЋС‚С‹ РєРѕРЅС‚СЂР°РєС‚Р° (%s). РЈРєР°Р¶РёС‚Рµ, СЃРєРѕР»СЊРєРѕ %s СЃРїРёСЃР°С‚СЊ СЃ Р±Р°Р»Р°РЅСЃР° РєРѕРЅС‚СЂР°РєС‚Р°",
			inv.Currency, dbContractCurrency, dbContractCurrency,
		)
	}

	if inv.DeductAmount > contractRemaining {
		return domain.Invoice{}, fmt.Errorf(
			"СЃСѓРјРјР° СЃРїРёСЃР°РЅРёСЏ (%.2f %s) РїСЂРµРІС‹С€Р°РµС‚ РѕСЃС‚Р°С‚РѕРє РїРѕ РєРѕРЅС‚СЂР°РєС‚Сѓ (%.2f %s)",
			inv.DeductAmount, dbContractCurrency, contractRemaining, dbContractCurrency,
		)
	}

	query := `
		INSERT INTO invoices (contract_id, invoice_number, invoice_name, invoice_date, amount, currency, deduct_amount)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, contract_id, invoice_number, invoice_name, invoice_date, amount, currency, deduct_amount, created_at, updated_at`

	var result domain.Invoice
	err = r.db.QueryRow(ctx, query,
		inv.ContractID, inv.InvoiceNumber, inv.InvoiceName, inv.InvoiceDate, inv.Amount, inv.Currency, inv.DeductAmount,
	).Scan(
		&result.ID, &result.ContractID, &result.InvoiceNumber, &result.InvoiceName,
		&result.InvoiceDate, &result.Amount, &result.Currency, &result.DeductAmount, &result.CreatedAt, &result.UpdatedAt,
	)
	return result, err
}

func (r *invoiceRepo) GetByID(ctx context.Context, id int64) (domain.Invoice, error) {
	query := `SELECT id, contract_id, invoice_number, invoice_name, invoice_date, amount, currency, deduct_amount, document_path, original_document_name, created_at, updated_at FROM invoices WHERE id = $1`
	var inv domain.Invoice
	err := r.db.QueryRow(ctx, query, id).Scan(
		&inv.ID, &inv.ContractID, &inv.InvoiceNumber, &inv.InvoiceName,
		&inv.InvoiceDate, &inv.Amount, &inv.Currency, &inv.DeductAmount,
		&inv.DocumentPath, &inv.OriginalDocumentName, &inv.CreatedAt, &inv.UpdatedAt,
	)
	return inv, err
}

func (r *invoiceRepo) GetByContractID(ctx context.Context, contractID int64) ([]domain.InvoiceWithDetails, error) {
	query := `SELECT id, contract_id, invoice_number, invoice_name, invoice_date, amount, currency, deduct_amount, document_path, original_document_name, created_at, updated_at FROM invoices WHERE contract_id = $1 ORDER BY invoice_date ASC`

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
			&inv.DocumentPath, &inv.OriginalDocumentName, &inv.CreatedAt, &inv.UpdatedAt,
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
		
		if inv.DocumentPath == nil {
			var docID int64
			var docName string
			docErr := r.db.QueryRow(ctx,
				`SELECT id, original_name FROM documents WHERE entity_type = 'invoice' AND entity_id = $1 LIMIT 1`,
				inv.ID,
			).Scan(&docID, &docName)
			if docErr == nil {
				inv.Document = &domain.Document{ID: docID, OriginalName: docName}
			}
		}

		result = append(result, inv)
	}
	if result == nil {
		result = []domain.InvoiceWithDetails{}
	}
	return result, nil
}


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

	query := `
		INSERT INTO gtd (invoice_id, gtd_number, gtd_amount, gtd_currency, gtd_date, closes_amount, document_path, original_document_name)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, invoice_id, gtd_number, gtd_amount, gtd_currency, gtd_date, closes_amount, document_path, original_document_name, created_at, updated_at`

	var result domain.GTD
	err = r.db.QueryRow(ctx, query,
		g.InvoiceID, g.GTDNumber, g.GTDAmount, g.GTDCurrency, g.GTDDate, g.ClosesAmount, g.DocumentPath, g.OriginalDocumentName,
	).Scan(
		&result.ID, &result.InvoiceID, &result.GTDNumber, &result.GTDAmount,
		&result.GTDCurrency, &result.GTDDate, &result.ClosesAmount,
		&result.DocumentPath, &result.OriginalDocumentName, &result.CreatedAt, &result.UpdatedAt,
	)
	return result, err
}

func (r *gtdRepo) GetByInvoiceID(ctx context.Context, invoiceID int64) (*domain.GTD, error) {
	query := `SELECT id, invoice_id, gtd_number, gtd_amount, gtd_currency, gtd_date, closes_amount, document_path, original_document_name, created_at, updated_at FROM gtd WHERE invoice_id = $1 LIMIT 1`
	var g domain.GTD
	err := r.db.QueryRow(ctx, query, invoiceID).Scan(
		&g.ID, &g.InvoiceID, &g.GTDNumber, &g.GTDAmount,
		&g.GTDCurrency, &g.GTDDate, &g.ClosesAmount,
		&g.DocumentPath, &g.OriginalDocumentName, &g.CreatedAt, &g.UpdatedAt,
	)
	if err != nil {
		return nil, nil
	}
	return &g, nil
}



func (r *additionalAgreementRepo) Create(ctx context.Context, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error) {
	var contractCurrency string
	err := r.db.QueryRow(ctx, "SELECT contract_currency FROM contracts WHERE id = $1", ag.ContractID).Scan(&contractCurrency)
	if err != nil {
		return domain.AdditionalAgreement{}, fmt.Errorf("РѕС€РёР±РєР° РїРѕР»СѓС‡РµРЅРёСЏ РєРѕРЅС‚СЂР°РєС‚Р°: %w", err)
	}

	if ag.ForeignCurrency != nil && ag.ForeignAmount != nil {
		if strings.EqualFold(*ag.ForeignCurrency, contractCurrency) && ag.AmountInContractCurrency == 0 {
			ag.AmountInContractCurrency = *ag.ForeignAmount
		}
	}

	query := `
		INSERT INTO additional_agreements (
			contract_id, delivery_conditions, delivery_term_days, return_term_days, subject,
			extend_date_to, foreign_amount, foreign_currency, amount_in_contract_currency,
			document_path, original_document_name
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)
		RETURNING id, contract_id, delivery_conditions, delivery_term_days, return_term_days, subject,
			extend_date_to, foreign_amount, foreign_currency, amount_in_contract_currency,
			document_path, original_document_name, created_at`

	var result domain.AdditionalAgreement
	err = r.db.QueryRow(ctx, query,
		ag.ContractID, ag.DeliveryConditions, ag.DeliveryTermDays, ag.ReturnTermDays, ag.Subject,
		ag.ExtendDateTo, ag.ForeignAmount, ag.ForeignCurrency, ag.AmountInContractCurrency,
		ag.DocumentPath, ag.OriginalDocumentName,
	).Scan(
		&result.ID, &result.ContractID, &result.DeliveryConditions,
		&result.DeliveryTermDays, &result.ReturnTermDays, &result.Subject,
		&result.ExtendDateTo, &result.ForeignAmount, &result.ForeignCurrency,
		&result.AmountInContractCurrency, &result.DocumentPath, &result.OriginalDocumentName,
		&result.CreatedAt,
	)
	return result, err
}

func (r *additionalAgreementRepo) GetByContractID(ctx context.Context, contractID int64) ([]domain.AdditionalAgreement, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, contract_id, delivery_conditions, delivery_term_days, return_term_days, subject,
			extend_date_to, foreign_amount, foreign_currency, amount_in_contract_currency,
			document_path, original_document_name, created_at
		FROM additional_agreements WHERE contract_id = $1 ORDER BY created_at ASC`,
		contractID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.AdditionalAgreement
	for rows.Next() {
		var ag domain.AdditionalAgreement
		if err := rows.Scan(
			&ag.ID, &ag.ContractID, &ag.DeliveryConditions,
			&ag.DeliveryTermDays, &ag.ReturnTermDays, &ag.Subject,
			&ag.ExtendDateTo, &ag.ForeignAmount, &ag.ForeignCurrency,
			&ag.AmountInContractCurrency, &ag.DocumentPath, &ag.OriginalDocumentName,
			&ag.CreatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, ag)
	}
	if result == nil {
		result = []domain.AdditionalAgreement{}
	}
	return result, nil
}