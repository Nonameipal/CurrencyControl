package repository

import (
	"context"
	"fmt"
	"time"

	"CurrencyControl/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type GTDRepository interface {
	Create(ctx context.Context, g domain.GTD) (domain.GTD, error)
	GetByID(ctx context.Context, id int64) (*domain.GTD, error)
	GetByInvoiceID(ctx context.Context, invoiceID int64) (*domain.GTD, error)
	GetListByInvoiceID(ctx context.Context, invoiceID int64) ([]domain.GTD, error)
	GetByContractID(ctx context.Context, contractID int64) ([]domain.GTD, error)
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
	var invoiceNumber string
	err := r.db.QueryRow(ctx, `
		SELECT i.amount, COALESCE(SUM(gtd.closes_amount), 0), i.invoice_number
		FROM invoices i
		LEFT JOIN gtd ON gtd.invoice_id = i.id AND gtd.deleted_at IS NULL
		WHERE i.id = $1
		GROUP BY i.amount, i.invoice_number`, g.InvoiceID,
	).Scan(&invoiceAmount, &alreadyClosed, &invoiceNumber)
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

	// 1. Получаем плановый регламентированный срок поставки по контракту
	var contractDate time.Time
	var contractDeliveryDate *time.Time
	var contractDeliveryTermDays int
	var aaDeliveryDate *time.Time
	var aaDeliveryTermDays *int

	_ = r.db.QueryRow(ctx, `
		SELECT 
			c.contract_date, 
			c.delivery_date, 
			c.delivery_term_days,
			(SELECT aa.delivery_date 
			 FROM additional_agreements aa 
			 WHERE aa.contract_id = c.id AND aa.deleted_at IS NULL AND aa.delivery_date IS NOT NULL 
			 ORDER BY aa.agreement_date DESC, aa.id DESC LIMIT 1),
			(SELECT aa.new_delivery_term_days 
			 FROM additional_agreements aa 
			 WHERE aa.contract_id = c.id AND aa.deleted_at IS NULL AND aa.new_delivery_term_days IS NOT NULL 
			 ORDER BY aa.agreement_date DESC, aa.id DESC LIMIT 1)
		FROM contracts c 
		WHERE c.id = $1 AND c.deleted_at IS NULL`, g.ContractID,
	).Scan(&contractDate, &contractDeliveryDate, &contractDeliveryTermDays, &aaDeliveryDate, &aaDeliveryTermDays)

	var deadline *time.Time
	if aaDeliveryDate != nil && !aaDeliveryDate.IsZero() {
		deadline = aaDeliveryDate
	} else if contractDeliveryDate != nil && !contractDeliveryDate.IsZero() {
		deadline = contractDeliveryDate
	} else if aaDeliveryTermDays != nil && *aaDeliveryTermDays > 0 && !contractDate.IsZero() {
		d := contractDate.AddDate(0, 0, *aaDeliveryTermDays)
		deadline = &d
	} else if contractDeliveryTermDays > 0 && !contractDate.IsZero() {
		d := contractDate.AddDate(0, 0, contractDeliveryTermDays)
		deadline = &d
	}

	// 2. Дата добавления ГТД формируется автоматически по текущей дате на момент загрузки в систему
	now := time.Now()
	submissionDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	g.SubmissionDate = &submissionDate
	g.DeliveryDeadline = deadline

	// 3. Автоматический расчет разницы в днях и формирование информационного уведомления
	diffDays, status, notice := domain.CalculateDeliveryComparison(g.DocumentType, submissionDate, deadline)
	g.DaysDifference = diffDays
	g.DeliveryStatus = status
	g.DeliveryNotice = notice

	query := `
		INSERT INTO gtd (
			contract_id, invoice_id, document_type, gtd_number, gtd_amount, gtd_currency, gtd_date,
			closes_amount, hs_code, destination_country, document_path, original_document_name, created_by,
			submission_date, delivery_deadline, days_difference, delivery_status, delivery_notice
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING id, contract_id, invoice_id, COALESCE(document_type, 'gtd'), gtd_number, gtd_amount,
			gtd_currency, gtd_date, closes_amount, COALESCE(hs_code, ''), COALESCE(destination_country, ''),
			document_path, original_document_name, COALESCE(created_by, ''),
			submission_date, delivery_deadline, COALESCE(days_difference, 0),
			COALESCE(delivery_status, ''), COALESCE(delivery_notice, ''),
			created_at, updated_at`

	var result domain.GTD
	err = r.db.QueryRow(ctx, query,
		g.ContractID, g.InvoiceID, g.DocumentType, g.GTDNumber, g.GTDAmount, g.GTDCurrency, g.GTDDate,
		g.ClosesAmount, g.HSCode, g.DestinationCountry, g.DocumentPath, g.OriginalDocumentName, g.CreatedBy,
		g.SubmissionDate, g.DeliveryDeadline, g.DaysDifference, g.DeliveryStatus, g.DeliveryNotice,
	).Scan(
		&result.ID, &result.ContractID, &result.InvoiceID, &result.DocumentType, &result.GTDNumber, &result.GTDAmount,
		&result.GTDCurrency, &result.GTDDate, &result.ClosesAmount, &result.HSCode, &result.DestinationCountry,
		&result.DocumentPath, &result.OriginalDocumentName, &result.CreatedBy,
		&result.SubmissionDate, &result.DeliveryDeadline, &result.DaysDifference,
		&result.DeliveryStatus, &result.DeliveryNotice,
		&result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		return domain.GTD{}, err
	}
	result.InvoiceNumber = invoiceNumber
	return result, nil
}

func (r *gtdRepo) GetByID(ctx context.Context, id int64) (*domain.GTD, error) {
	query := `
		SELECT g.id, g.contract_id, g.invoice_id, COALESCE(g.document_type, 'gtd'), g.gtd_number, g.gtd_amount,
		       g.gtd_currency, g.gtd_date, g.closes_amount, COALESCE(g.hs_code, ''), COALESCE(g.destination_country, ''),
		       g.document_path, g.original_document_name, COALESCE(g.created_by, ''),
		       g.submission_date, g.delivery_deadline, COALESCE(g.days_difference, 0),
		       COALESCE(g.delivery_status, ''), COALESCE(g.delivery_notice, ''),
		       g.created_at, g.updated_at, COALESCE(i.invoice_number, '')
		FROM gtd g
		LEFT JOIN invoices i ON i.id = g.invoice_id
		WHERE g.id = $1 AND g.deleted_at IS NULL`

	var g domain.GTD
	err := r.db.QueryRow(ctx, query, id).Scan(
		&g.ID, &g.ContractID, &g.InvoiceID, &g.DocumentType, &g.GTDNumber, &g.GTDAmount,
		&g.GTDCurrency, &g.GTDDate, &g.ClosesAmount, &g.HSCode, &g.DestinationCountry,
		&g.DocumentPath, &g.OriginalDocumentName, &g.CreatedBy,
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

func (r *gtdRepo) GetByInvoiceID(ctx context.Context, invoiceID int64) (*domain.GTD, error) {
	query := `
		SELECT g.id, g.contract_id, g.invoice_id, COALESCE(g.document_type, 'gtd'), g.gtd_number, g.gtd_amount,
		       g.gtd_currency, g.gtd_date, g.closes_amount, COALESCE(g.hs_code, ''), COALESCE(g.destination_country, ''),
		       g.document_path, g.original_document_name, COALESCE(g.created_by, ''),
		       g.submission_date, g.delivery_deadline, COALESCE(g.days_difference, 0),
		       COALESCE(g.delivery_status, ''), COALESCE(g.delivery_notice, ''),
		       g.created_at, g.updated_at, COALESCE(i.invoice_number, '')
		FROM gtd g
		LEFT JOIN invoices i ON i.id = g.invoice_id
		WHERE g.invoice_id = $1 AND g.deleted_at IS NULL
		ORDER BY g.id DESC LIMIT 1`

	var g domain.GTD
	err := r.db.QueryRow(ctx, query, invoiceID).Scan(
		&g.ID, &g.ContractID, &g.InvoiceID, &g.DocumentType, &g.GTDNumber, &g.GTDAmount,
		&g.GTDCurrency, &g.GTDDate, &g.ClosesAmount, &g.HSCode, &g.DestinationCountry,
		&g.DocumentPath, &g.OriginalDocumentName, &g.CreatedBy,
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
		SELECT g.id, g.contract_id, g.invoice_id, COALESCE(g.document_type, 'gtd'), g.gtd_number, g.gtd_amount,
		       g.gtd_currency, g.gtd_date, g.closes_amount, COALESCE(g.hs_code, ''), COALESCE(g.destination_country, ''),
		       g.document_path, g.original_document_name, COALESCE(g.created_by, ''),
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
			&g.ID, &g.ContractID, &g.InvoiceID, &g.DocumentType, &g.GTDNumber, &g.GTDAmount,
			&g.GTDCurrency, &g.GTDDate, &g.ClosesAmount, &g.HSCode, &g.DestinationCountry,
			&g.DocumentPath, &g.OriginalDocumentName, &g.CreatedBy,
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
		SELECT g.id, g.contract_id, g.invoice_id, COALESCE(g.document_type, 'gtd'), g.gtd_number, g.gtd_amount,
		       g.gtd_currency, g.gtd_date, g.closes_amount, COALESCE(g.hs_code, ''), COALESCE(g.destination_country, ''),
		       g.document_path, g.original_document_name, COALESCE(g.created_by, ''),
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
			&g.ID, &g.ContractID, &g.InvoiceID, &g.DocumentType, &g.GTDNumber, &g.GTDAmount,
			&g.GTDCurrency, &g.GTDDate, &g.ClosesAmount, &g.HSCode, &g.DestinationCountry,
			&g.DocumentPath, &g.OriginalDocumentName, &g.CreatedBy,
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
			closes_amount = $7, hs_code = $8, destination_country = $9, document_path = $10, original_document_name = $11, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, contract_id, invoice_id, COALESCE(document_type, 'gtd'), gtd_number, gtd_amount,
			gtd_currency, gtd_date, closes_amount, COALESCE(hs_code, ''), COALESCE(destination_country, ''),
			document_path, original_document_name, COALESCE(created_by, ''),
			submission_date, delivery_deadline, COALESCE(days_difference, 0),
			COALESCE(delivery_status, ''), COALESCE(delivery_notice, ''),
			created_at, updated_at`

	var result domain.GTD
	err := r.db.QueryRow(ctx, query,
		id, g.DocumentType, g.GTDNumber, g.GTDAmount, g.GTDCurrency, g.GTDDate, g.ClosesAmount, g.HSCode, g.DestinationCountry, g.DocumentPath, g.OriginalDocumentName,
	).Scan(
		&result.ID, &result.ContractID, &result.InvoiceID, &result.DocumentType, &result.GTDNumber, &result.GTDAmount,
		&result.GTDCurrency, &result.GTDDate, &result.ClosesAmount, &result.HSCode, &result.DestinationCountry,
		&result.DocumentPath, &result.OriginalDocumentName, &result.CreatedBy,
		&result.SubmissionDate, &result.DeliveryDeadline, &result.DaysDifference,
		&result.DeliveryStatus, &result.DeliveryNotice,
		&result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		return domain.GTD{}, err
	}

	if result.InvoiceID > 0 {
		var invoiceNumber string
		_ = r.db.QueryRow(ctx, `SELECT invoice_number FROM invoices WHERE id = $1`, result.InvoiceID).Scan(&invoiceNumber)
		result.InvoiceNumber = invoiceNumber
	}

	return result, nil
}



