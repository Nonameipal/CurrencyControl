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

func NewGTDRepository(db *pgxpool.Pool) ports.GTDRepository {
	return &gtdRepo{db: db}
}

type gtdRepo struct{ db *pgxpool.Pool }

func (r *gtdRepo) Create(ctx context.Context, g domain.GTD) (domain.GTD, error) {
	var invoiceAmount float64
	var alreadyClosed float64
	var invoiceNumber string
	var invoiceCurrency string
	var invoiceAddlID *int64
	err := r.db.QueryRow(ctx, `
		SELECT i.amount, COALESCE(SUM(gtd.closes_amount), 0), i.invoice_number, i.currency, i.additional_agreement_id
		FROM invoices i
		LEFT JOIN gtd ON gtd.invoice_id = i.id AND gtd.deleted_at IS NULL
		WHERE i.id = $1
		GROUP BY i.amount, i.invoice_number, i.currency, i.additional_agreement_id`, g.InvoiceID,
	).Scan(&invoiceAmount, &alreadyClosed, &invoiceNumber, &invoiceCurrency, &invoiceAddlID)
	if err != nil {
		return domain.GTD{}, fmt.Errorf("инвойс не найден")
	}

	if g.GTDCurrency != nil && !strings.EqualFold(*g.GTDCurrency, invoiceCurrency) {
		return domain.GTD{}, fmt.Errorf("валюта ГТД (%s) должна совпадать с валютой инвойса (%s)", *g.GTDCurrency, invoiceCurrency)
	}

	g.AdditionalAgreementID = invoiceAddlID

	if g.ClosesAmount <= 0 {
		g.ClosesAmount = g.GTDAmount
	}

	invoiceRemaining := invoiceAmount - alreadyClosed
	if g.ClosesAmount > invoiceRemaining {
		return domain.GTD{}, fmt.Errorf(
			"сумма ГТД (%.2f) превышает товарный остаток по инвойсу (%.2f)",
			g.ClosesAmount, invoiceRemaining,
		)
	}

	if g.DocumentType == "" {
		g.DocumentType = domain.DocumentTypeGTD
	}

	var contractDeliveryDate *time.Time
	var aaDeliveryDate *time.Time

	_ = r.db.QueryRow(ctx, `
		SELECT 
			c.delivery_date, 
			(SELECT aa.delivery_date 
			 FROM additional_agreements aa 
			 WHERE aa.contract_id = c.id AND aa.deleted_at IS NULL AND aa.delivery_date IS NOT NULL 
			 ORDER BY aa.agreement_date DESC, aa.id DESC LIMIT 1)
		FROM contracts c 
		WHERE c.id = $1 AND c.deleted_at IS NULL`, g.ContractID,
	).Scan(&contractDeliveryDate, &aaDeliveryDate)

	var deadline *time.Time
	if aaDeliveryDate != nil && !aaDeliveryDate.IsZero() {
		deadline = aaDeliveryDate
	} else if contractDeliveryDate != nil && !contractDeliveryDate.IsZero() {
		deadline = contractDeliveryDate
	}

	now := time.Now()
	submissionDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	g.SubmissionDate = &submissionDate
	g.DeliveryDeadline = deadline

	// Rule 3: GTD submission deadline = delivery_date + 5 days
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

	query := `
		INSERT INTO gtd (
			contract_id, additional_agreement_id, invoice_id, document_type, gtd_number, gtd_amount, gtd_currency, gtd_date,
			closes_amount, hs_code, destination_country, document_path, created_by,
			submission_date, delivery_deadline, days_difference, delivery_status, delivery_notice, approval_status
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)
		RETURNING id, contract_id, additional_agreement_id, invoice_id, COALESCE(document_type, 'gtd'), gtd_number, gtd_amount,
			gtd_currency, gtd_date, closes_amount, COALESCE(hs_code, ''), COALESCE(destination_country, ''),
			document_path, COALESCE(created_by, ''),
			submission_date, delivery_deadline, COALESCE(days_difference, 0),
			COALESCE(delivery_status, ''), COALESCE(delivery_notice, ''),
			COALESCE(approval_status, 'pending_currency_control'), created_at, updated_at`

	var result domain.GTD
	err = r.db.QueryRow(ctx, query,
		g.ContractID, g.AdditionalAgreementID, g.InvoiceID, g.DocumentType, g.GTDNumber, g.GTDAmount, g.GTDCurrency, g.GTDDate,
		g.ClosesAmount, g.HSCode, g.DestinationCountry, g.DocumentPath, g.CreatedBy,
		g.SubmissionDate, g.DeliveryDeadline, g.DaysDifference, g.DeliveryStatus, g.DeliveryNotice, g.ApprovalStatus,
	).Scan(
		&result.ID, &result.ContractID, &result.AdditionalAgreementID, &result.InvoiceID, &result.DocumentType, &result.GTDNumber, &result.GTDAmount,
		&result.GTDCurrency, &result.GTDDate, &result.ClosesAmount, &result.HSCode, &result.DestinationCountry,
		&result.DocumentPath, &result.CreatedBy,
		&result.SubmissionDate, &result.DeliveryDeadline, &result.DaysDifference,
		&result.DeliveryStatus, &result.DeliveryNotice,
		&result.ApprovalStatus, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		return domain.GTD{}, err
	}
	result.InvoiceNumber = invoiceNumber

	// Trigger archive check — a new GTD may close the last open invoice
	if result.AdditionalAgreementID != nil {
		tryArchiveAdditionalAgreement(ctx, r.db, *result.AdditionalAgreementID)
	} else {
		tryArchiveContract(ctx, r.db, result.ContractID)
	}

	return result, nil
}

func (r *gtdRepo) GetByID(ctx context.Context, id int64) (*domain.GTD, error) {
	query := `
		SELECT g.id, g.contract_id, g.additional_agreement_id, g.invoice_id, COALESCE(g.document_type, 'gtd'), g.gtd_number, g.gtd_amount,
		       g.gtd_currency, g.gtd_date, g.closes_amount, COALESCE(g.hs_code, ''), COALESCE(g.destination_country, ''),
		       g.document_path, COALESCE(g.created_by, ''),
		       g.submission_date, g.delivery_deadline, COALESCE(g.days_difference, 0),
		       COALESCE(g.delivery_status, ''), COALESCE(g.delivery_notice, ''),
		       COALESCE(g.approval_status, 'pending_currency_control'),
		       g.created_at, g.updated_at, COALESCE(i.invoice_number, '')
		FROM gtd g
		LEFT JOIN invoices i ON i.id = g.invoice_id
		WHERE g.id = $1 AND g.deleted_at IS NULL`

	var g domain.GTD
	err := r.db.QueryRow(ctx, query, id).Scan(
		&g.ID, &g.ContractID, &g.AdditionalAgreementID, &g.InvoiceID, &g.DocumentType, &g.GTDNumber, &g.GTDAmount,
		&g.GTDCurrency, &g.GTDDate, &g.ClosesAmount, &g.HSCode, &g.DestinationCountry,
		&g.DocumentPath, &g.CreatedBy,
		&g.SubmissionDate, &g.DeliveryDeadline, &g.DaysDifference,
		&g.DeliveryStatus, &g.DeliveryNotice,
		&g.ApprovalStatus,
		&g.CreatedAt, &g.UpdatedAt,
		&g.InvoiceNumber,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("ГТД не найдена")
		}
		return nil, err
	}
	return &g, nil
}

func (r *gtdRepo) GetByInvoiceID(ctx context.Context, invoiceID int64) (*domain.GTD, error) {
	query := `
		SELECT g.id, g.contract_id, g.additional_agreement_id, g.invoice_id, COALESCE(g.document_type, 'gtd'), g.gtd_number, g.gtd_amount,
		       g.gtd_currency, g.gtd_date, g.closes_amount, COALESCE(g.hs_code, ''), COALESCE(g.destination_country, ''),
		       g.document_path, COALESCE(g.created_by, ''),
		       g.submission_date, g.delivery_deadline, COALESCE(g.days_difference, 0),
		       COALESCE(g.delivery_status, ''), COALESCE(g.delivery_notice, ''),
		       g.created_at, g.updated_at, COALESCE(i.invoice_number, '')
		FROM gtd g
		LEFT JOIN invoices i ON i.id = g.invoice_id
		WHERE g.invoice_id = $1 AND g.deleted_at IS NULL
		ORDER BY g.id DESC LIMIT 1`

	var g domain.GTD
	err := r.db.QueryRow(ctx, query, invoiceID).Scan(
		&g.ID, &g.ContractID, &g.AdditionalAgreementID, &g.InvoiceID, &g.DocumentType, &g.GTDNumber, &g.GTDAmount,
		&g.GTDCurrency, &g.GTDDate, &g.ClosesAmount, &g.HSCode, &g.DestinationCountry,
		&g.DocumentPath, &g.CreatedBy,
		&g.SubmissionDate, &g.DeliveryDeadline, &g.DaysDifference,
		&g.DeliveryStatus, &g.DeliveryNotice,
		&g.CreatedAt, &g.UpdatedAt,
		&g.InvoiceNumber,
	)
	if err != nil {
		return nil, nil
	}
	return &g, nil
}

func (r *gtdRepo) GetListByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.GTD, error) {
	query := `
		SELECT g.id, g.contract_id, g.additional_agreement_id, g.invoice_id, COALESCE(g.document_type, 'gtd'), g.gtd_number, g.gtd_amount,
		       g.gtd_currency, g.gtd_date, g.closes_amount, COALESCE(g.hs_code, ''), COALESCE(g.destination_country, ''),
		       g.document_path, COALESCE(g.created_by, ''),
		       g.submission_date, g.delivery_deadline, COALESCE(g.days_difference, 0),
		       COALESCE(g.delivery_status, ''), COALESCE(g.delivery_notice, ''),
		       g.created_at, g.updated_at, COALESCE(i.invoice_number, '')
		FROM gtd g
		LEFT JOIN invoices i ON i.id = g.invoice_id
		WHERE g.invoice_id = $1 AND g.deleted_at IS NULL
		ORDER BY g.id ASC`

	rows, err := r.db.Query(ctx, query, invoiceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.GTD
	for rows.Next() {
		var g domain.GTD
		err := rows.Scan(
			&g.ID, &g.ContractID, &g.AdditionalAgreementID, &g.InvoiceID, &g.DocumentType, &g.GTDNumber, &g.GTDAmount,
			&g.GTDCurrency, &g.GTDDate, &g.ClosesAmount, &g.HSCode, &g.DestinationCountry,
			&g.DocumentPath, &g.CreatedBy,
			&g.SubmissionDate, &g.DeliveryDeadline, &g.DaysDifference,
			&g.DeliveryStatus, &g.DeliveryNotice,
			&g.CreatedAt, &g.UpdatedAt,
			&g.InvoiceNumber,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, g)
	}
	if list == nil {
		list = []domain.GTD{}
	}
	return list, nil
}

func (r *gtdRepo) GetByContractID(ctx context.Context, contractID int64) ([]domain.GTD, error) {
	query := `
		SELECT g.id, g.contract_id, g.additional_agreement_id, g.invoice_id, COALESCE(g.document_type, 'gtd'), g.gtd_number, g.gtd_amount,
		       g.gtd_currency, g.gtd_date, g.closes_amount, COALESCE(g.hs_code, ''), COALESCE(g.destination_country, ''),
		       g.document_path, COALESCE(g.created_by, ''),
		       g.submission_date, g.delivery_deadline, COALESCE(g.days_difference, 0),
		       COALESCE(g.delivery_status, ''), COALESCE(g.delivery_notice, ''),
		       g.created_at, g.updated_at, COALESCE(i.invoice_number, '')
		FROM gtd g
		LEFT JOIN invoices i ON i.id = g.invoice_id
		WHERE g.contract_id = $1 AND g.deleted_at IS NULL
		ORDER BY g.id ASC`

	rows, err := r.db.Query(ctx, query, contractID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.GTD
	for rows.Next() {
		var g domain.GTD
		err := rows.Scan(
			&g.ID, &g.ContractID, &g.AdditionalAgreementID, &g.InvoiceID, &g.DocumentType, &g.GTDNumber, &g.GTDAmount,
			&g.GTDCurrency, &g.GTDDate, &g.ClosesAmount, &g.HSCode, &g.DestinationCountry,
			&g.DocumentPath, &g.CreatedBy,
			&g.SubmissionDate, &g.DeliveryDeadline, &g.DaysDifference,
			&g.DeliveryStatus, &g.DeliveryNotice,
			&g.CreatedAt, &g.UpdatedAt,
			&g.InvoiceNumber,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, g)
	}
	if list == nil {
		list = []domain.GTD{}
	}
	return list, nil
}

func (r *gtdRepo) GetByAdditionalAgreementID(ctx context.Context, agreementID int64) ([]domain.GTD, error) {
	query := `
		SELECT g.id, g.contract_id, g.additional_agreement_id, g.invoice_id, COALESCE(g.document_type, 'gtd'), g.gtd_number, g.gtd_amount,
		       g.gtd_currency, g.gtd_date, g.closes_amount, COALESCE(g.hs_code, ''), COALESCE(g.destination_country, ''),
		       g.document_path, COALESCE(g.created_by, ''),
		       g.submission_date, g.delivery_deadline, COALESCE(g.days_difference, 0),
		       COALESCE(g.delivery_status, ''), COALESCE(g.delivery_notice, ''),
		       g.created_at, g.updated_at, COALESCE(i.invoice_number, '')
		FROM gtd g
		LEFT JOIN invoices i ON i.id = g.invoice_id
		WHERE g.additional_agreement_id = $1 AND g.deleted_at IS NULL
		ORDER BY g.id ASC`

	rows, err := r.db.Query(ctx, query, agreementID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var list []domain.GTD
	for rows.Next() {
		var g domain.GTD
		err := rows.Scan(
			&g.ID, &g.ContractID, &g.AdditionalAgreementID, &g.InvoiceID, &g.DocumentType, &g.GTDNumber, &g.GTDAmount,
			&g.GTDCurrency, &g.GTDDate, &g.ClosesAmount, &g.HSCode, &g.DestinationCountry,
			&g.DocumentPath, &g.CreatedBy,
			&g.SubmissionDate, &g.DeliveryDeadline, &g.DaysDifference,
			&g.DeliveryStatus, &g.DeliveryNotice,
			&g.CreatedAt, &g.UpdatedAt,
			&g.InvoiceNumber,
		)
		if err != nil {
			return nil, err
		}
		list = append(list, g)
	}
	if list == nil {
		list = []domain.GTD{}
	}
	return list, nil
}

func (r *gtdRepo) SoftDelete(ctx context.Context, id int64) error {
	var contractID int64
	var addlID *int64
	err := r.db.QueryRow(ctx,
		`SELECT contract_id, additional_agreement_id FROM gtd WHERE id = $1 AND deleted_at IS NULL`, id,
	).Scan(&contractID, &addlID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("ГТД не найдена")
		}
		return err
	}

	cmdTag, err := r.db.Exec(ctx, `UPDATE gtd SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return errors.New("ГТД не найдена")
	}

	if addlID != nil {
		tryArchiveAdditionalAgreement(ctx, r.db, *addlID)
	} else if contractID > 0 {
		tryArchiveContract(ctx, r.db, contractID)
	}
	return nil
}

func (r *gtdRepo) Update(ctx context.Context, id int64, g domain.GTD) (domain.GTD, error) {
	var exists bool
	err := r.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM gtd WHERE id = $1 AND deleted_at IS NULL)`, id).Scan(&exists)
	if err != nil {
		return domain.GTD{}, err
	}
	if !exists {
		return domain.GTD{}, errors.New("ГТД не найдена")
	}

	var invoiceCurrency string
	err = r.db.QueryRow(ctx, `
		SELECT i.currency 
		FROM gtd g 
		JOIN invoices i ON i.id = g.invoice_id 
		WHERE g.id = $1 AND g.deleted_at IS NULL`, id,
	).Scan(&invoiceCurrency)
	if err == nil && g.GTDCurrency != nil && invoiceCurrency != "" && !strings.EqualFold(*g.GTDCurrency, invoiceCurrency) {
		return domain.GTD{}, fmt.Errorf("валюта ГТД (%s) должна совпадать с валютой инвойса (%s)", *g.GTDCurrency, invoiceCurrency)
	}

	if g.DocumentType == "" {
		g.DocumentType = domain.DocumentTypeGTD
	}
	if g.ClosesAmount <= 0 {
		g.ClosesAmount = g.GTDAmount
	}
	query := `
		UPDATE gtd SET
			document_type = $2, gtd_number = $3, gtd_amount = $4, gtd_currency = $5, gtd_date = $6,
			closes_amount = $7, hs_code = $8, destination_country = $9, document_path = $10,
			approval_status = 'pending_currency_control', rejection_reason = '', updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, contract_id, additional_agreement_id, invoice_id, COALESCE(document_type, 'gtd'), gtd_number, gtd_amount,
			gtd_currency, gtd_date, closes_amount, COALESCE(hs_code, ''), COALESCE(destination_country, ''),
			document_path, COALESCE(created_by, ''),
			submission_date, delivery_deadline, COALESCE(days_difference, 0),
			COALESCE(delivery_status, ''), COALESCE(delivery_notice, ''),
			COALESCE(approval_status, 'pending_currency_control'), created_at, updated_at`

	var result domain.GTD
	err = r.db.QueryRow(ctx, query,
		id, g.DocumentType, g.GTDNumber, g.GTDAmount, g.GTDCurrency, g.GTDDate, g.ClosesAmount, g.HSCode, g.DestinationCountry, g.DocumentPath, 
	).Scan(
		&result.ID, &result.ContractID, &result.AdditionalAgreementID, &result.InvoiceID, &result.DocumentType, &result.GTDNumber, &result.GTDAmount,
		&result.GTDCurrency, &result.GTDDate, &result.ClosesAmount, &result.HSCode, &result.DestinationCountry,
		&result.DocumentPath, &result.CreatedBy,
		&result.SubmissionDate, &result.DeliveryDeadline, &result.DaysDifference,
		&result.DeliveryStatus, &result.DeliveryNotice,
		&result.ApprovalStatus, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.GTD{}, errors.New("ГТД не найдена")
		}
		return domain.GTD{}, err
	}

	if result.InvoiceID > 0 {
		var invoiceNumber string
		_ = r.db.QueryRow(ctx, `SELECT invoice_number FROM invoices WHERE id = $1`, result.InvoiceID).Scan(&invoiceNumber)
		result.InvoiceNumber = invoiceNumber
	}

	return result, nil
}



