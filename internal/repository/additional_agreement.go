package repository
import (
	"context"
	"fmt"
	"strings"
	"time"

	"CurrencyControl/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)
type AdditionalAgreementRepository interface {
	Create(ctx context.Context, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error)
	GetByContractID(ctx context.Context, contractID int64) ([]domain.AdditionalAgreement, error)
	GetByID(ctx context.Context, id int64) (domain.AdditionalAgreement, error)
	Update(ctx context.Context, id int64, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error)
	SoftDelete(ctx context.Context, id int64) error
	RestoreAdditionalAgreement(ctx context.Context, id int64) error
}
func NewAdditionalAgreementRepository(db *pgxpool.Pool) AdditionalAgreementRepository {
	return &additionalAgreementRepo{db: db}
}
type additionalAgreementRepo struct{ db *pgxpool.Pool }

func (r *additionalAgreementRepo) Create(ctx context.Context, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error) {
	if ag.DocType == "" {
		ag.DocType = domain.DocTypeAdditionalAgreement
	}
	ag.Normalize()

	if ag.RemainingAmount == 0 && ag.ForeignAmount != nil {
		ag.RemainingAmount = *ag.ForeignAmount
	}

	// Rule 1: block new AA if contract has expired
	{
		var extendDateTo *time.Time
		var deliveryDate *time.Time
		_ = r.db.QueryRow(ctx,
			`SELECT extend_date_to, delivery_date FROM contracts WHERE id = $1 AND deleted_at IS NULL`,
			ag.ContractID,
		).Scan(&extendDateTo, &deliveryDate)

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
				return domain.AdditionalAgreement{}, fmt.Errorf("срок действия контракта истёк (%s). Добавление доп. соглашений запрещено", exp.Format("02.01.2006"))
			}
		}
	}

	query := `
		INSERT INTO additional_agreements (
			contract_id, doc_type, agreement_number, agreement_date,
			delivery_date, return_date, subject,
			extend_date_to, foreign_amount, currency, amount_in_contract_currency, remaining_amount,
			document_path, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id, contract_id, COALESCE(doc_type, 'additional_agreement'), agreement_number, agreement_date,
			delivery_date, return_date, subject,
			extend_date_to, foreign_amount, currency, amount_in_contract_currency, remaining_amount,
			document_path, COALESCE(created_by, ''), created_at, COALESCE(updated_at, created_at)`

	var result domain.AdditionalAgreement
	err := r.db.QueryRow(ctx, query,
		ag.ContractID, ag.DocType, ag.AgreementNumber, ag.AgreementDate,
		ag.DeliveryDate, ag.ReturnDate, ag.Subject,
		ag.ExtendDateTo, ag.ForeignAmount, ag.ForeignCurrency, ag.AmountInContractCurrency, ag.RemainingAmount,
		ag.DocumentPath, ag.CreatedBy,
	).Scan(
		&result.ID, &result.ContractID, &result.DocType, &result.AgreementNumber, &result.AgreementDate,
		&result.DeliveryDate, &result.ReturnDate, &result.Subject,
		&result.ExtendDateTo, &result.ForeignAmount, &result.ForeignCurrency,
		&result.AmountInContractCurrency, &result.RemainingAmount, &result.DocumentPath,
		&result.CreatedBy, &result.CreatedAt, &result.UpdatedAt,
	)
	if err == nil {
		result.Normalize()
		propagateAADatesToContract(ctx, r.db, result.ContractID, result.DeliveryDate, result.ReturnDate, result.ExtendDateTo)
	}
	return result, err
}

func (r *additionalAgreementRepo) GetByContractID(ctx context.Context, contractID int64) ([]domain.AdditionalAgreement, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, contract_id, COALESCE(doc_type, 'additional_agreement'), agreement_number, agreement_date,
			delivery_date, return_date, subject,
			extend_date_to, foreign_amount, currency, amount_in_contract_currency, remaining_amount,
			document_path, COALESCE(created_by, ''), created_at, COALESCE(updated_at, created_at)
		FROM additional_agreements WHERE contract_id = $1 AND deleted_at IS NULL ORDER BY created_at ASC`,
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
			&ag.ID, &ag.ContractID, &ag.DocType, &ag.AgreementNumber, &ag.AgreementDate,
			&ag.DeliveryDate, &ag.ReturnDate, &ag.Subject,
			&ag.ExtendDateTo, &ag.ForeignAmount, &ag.ForeignCurrency,
			&ag.AmountInContractCurrency, &ag.RemainingAmount, &ag.DocumentPath,
			&ag.CreatedBy, &ag.CreatedAt, &ag.UpdatedAt,
		); err != nil {
			return nil, err
		}
		ag.Normalize()
		result = append(result, ag)
	}
	if result == nil {
		result = []domain.AdditionalAgreement{}
	}
	return result, nil
}

func (r *additionalAgreementRepo) GetByID(ctx context.Context, id int64) (domain.AdditionalAgreement, error) {
	query := `
		SELECT id, contract_id, COALESCE(doc_type, 'additional_agreement'), agreement_number, agreement_date,
			delivery_date, return_date, subject,
			extend_date_to, foreign_amount, currency, amount_in_contract_currency, remaining_amount,
			document_path, COALESCE(created_by, ''), created_at, COALESCE(updated_at, created_at)
		FROM additional_agreements WHERE id = $1 AND deleted_at IS NULL`
	var ag domain.AdditionalAgreement
	err := r.db.QueryRow(ctx, query, id).Scan(
		&ag.ID, &ag.ContractID, &ag.DocType, &ag.AgreementNumber, &ag.AgreementDate,
		&ag.DeliveryDate, &ag.ReturnDate, &ag.Subject,
		&ag.ExtendDateTo, &ag.ForeignAmount, &ag.ForeignCurrency,
		&ag.AmountInContractCurrency, &ag.RemainingAmount, &ag.DocumentPath,
		&ag.CreatedBy, &ag.CreatedAt, &ag.UpdatedAt,
	)
	if err == nil {
		ag.Normalize()
	}
	return ag, err
}

func (r *additionalAgreementRepo) SoftDelete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `UPDATE additional_agreements SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}

func (r *additionalAgreementRepo) Update(ctx context.Context, id int64, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error) {
	if ag.DocType == "" {
		ag.DocType = domain.DocTypeAdditionalAgreement
	}
	ag.Normalize()
	query := `
		UPDATE additional_agreements SET
			doc_type = $2, agreement_number = $3, agreement_date = $4,
			delivery_date = $5, return_date = $6, subject = $7,
			extend_date_to = $8, foreign_amount = $9, currency = $10,
			amount_in_contract_currency = $11, document_path = $12,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, contract_id, COALESCE(doc_type, 'additional_agreement'), agreement_number, agreement_date,
			delivery_date, return_date, subject,
			extend_date_to, foreign_amount, currency, amount_in_contract_currency, remaining_amount,
			document_path, COALESCE(created_by, ''), created_at, COALESCE(updated_at, created_at)`
	var result domain.AdditionalAgreement
	err := r.db.QueryRow(ctx, query,
		id, ag.DocType, ag.AgreementNumber, ag.AgreementDate,
		ag.DeliveryDate, ag.ReturnDate, ag.Subject,
		ag.ExtendDateTo, ag.ForeignAmount, ag.ForeignCurrency,
		ag.AmountInContractCurrency, ag.DocumentPath,
	).Scan(
		&result.ID, &result.ContractID, &result.DocType, &result.AgreementNumber, &result.AgreementDate,
		&result.DeliveryDate, &result.ReturnDate, &result.Subject,
		&result.ExtendDateTo, &result.ForeignAmount, &result.ForeignCurrency,
		&result.AmountInContractCurrency, &result.RemainingAmount, &result.DocumentPath,
		&result.CreatedBy, &result.CreatedAt, &result.UpdatedAt,
	)
	if err == nil {
		result.Normalize()
		syncAdditionalAgreementRemaining(ctx, r.db, result.ID)
		propagateAADatesToContract(ctx, r.db, result.ContractID, result.DeliveryDate, result.ReturnDate, result.ExtendDateTo)
	}
	return result, err
}

func syncAdditionalAgreementRemaining(ctx context.Context, db *pgxpool.Pool, addlID int64) {
	_, _ = db.Exec(ctx, `
		UPDATE additional_agreements
		SET remaining_amount = COALESCE(foreign_amount, 0)
			- COALESCE((SELECT SUM(amount) FROM invoices WHERE additional_agreement_id = additional_agreements.id AND deleted_at IS NULL), 0),
			updated_at = NOW()
		WHERE id = $1`, addlID,
	)
}

// propagateAADatesToContract — Rule 2:
// When an additional agreement sets delivery_date, return_date or extend_date_to
// those values are pushed to the parent contract so that deadlines stay in sync.
// Only non-nil values overwrite the contract field.
func propagateAADatesToContract(ctx context.Context, db *pgxpool.Pool, contractID int64, deliveryDate, returnDate, extendDateTo *time.Time) {
	if deliveryDate == nil && returnDate == nil && extendDateTo == nil {
		return
	}
	parts := []string{"updated_at = NOW()"}
	args := []interface{}{contractID}
	argIdx := 2

	if deliveryDate != nil && !deliveryDate.IsZero() {
		parts = append(parts, fmt.Sprintf("delivery_date = $%d", argIdx))
		args = append(args, *deliveryDate)
		argIdx++
	}
	if returnDate != nil && !returnDate.IsZero() {
		parts = append(parts, fmt.Sprintf("return_date = $%d", argIdx))
		args = append(args, *returnDate)
		argIdx++
	}
	if extendDateTo != nil && !extendDateTo.IsZero() {
		parts = append(parts, fmt.Sprintf("extend_date_to = $%d", argIdx))
		args = append(args, *extendDateTo)
		argIdx++
	}

	setClause := strings.Join(parts, ", ")
	_, _ = db.Exec(ctx,
		fmt.Sprintf("UPDATE contracts SET %s WHERE id = $1 AND deleted_at IS NULL", setClause),
		args...,
	)
}

// RestoreAdditionalAgreement sets the agreement back to 'active' status.
func (r *additionalAgreementRepo) RestoreAdditionalAgreement(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx,
		`UPDATE additional_agreements SET status = 'active', archived_at = NULL, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	return err
}