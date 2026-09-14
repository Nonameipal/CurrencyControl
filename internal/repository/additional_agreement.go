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

func NewAdditionalAgreementRepository(db *pgxpool.Pool) ports.AdditionalAgreementRepository {
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

	if ag.ApprovalStatus == "" {
		ag.ApprovalStatus = domain.ApprovalStatusPendingCurrencyControl
	}

	query := `
		INSERT INTO additional_agreements (
			contract_id, doc_type, agreement_number, agreement_date,
			delivery_date, return_date, subject,
			extend_date_to, foreign_amount, currency, remaining_amount,
			document_path, created_by, receiver_name, receiver_bank, receiver_country, approval_status
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)
		RETURNING id, contract_id, COALESCE(doc_type, 'additional_agreement'), agreement_number, agreement_date,
			delivery_date, return_date, subject,
			extend_date_to, foreign_amount, currency, remaining_amount,
			document_path, COALESCE(created_by, ''), COALESCE(receiver_name, ''), COALESCE(receiver_bank, ''), COALESCE(receiver_country, ''), COALESCE(approval_status, 'pending_currency_control'), created_at, COALESCE(updated_at, created_at)`

	var result domain.AdditionalAgreement
	err := r.db.QueryRow(ctx, query,
		ag.ContractID, ag.DocType, ag.AgreementNumber, ag.AgreementDate,
		ag.DeliveryDate, ag.ReturnDate, ag.Subject,
		ag.ExtendDateTo, ag.ForeignAmount, ag.ForeignCurrency, ag.RemainingAmount,
		ag.DocumentPath, ag.CreatedBy, ag.ReceiverName, ag.ReceiverBank, ag.ReceiverCountry, ag.ApprovalStatus,
	).Scan(
		&result.ID, &result.ContractID, &result.DocType, &result.AgreementNumber, &result.AgreementDate,
		&result.DeliveryDate, &result.ReturnDate, &result.Subject,
		&result.ExtendDateTo, &result.ForeignAmount, &result.ForeignCurrency,
		&result.RemainingAmount, &result.DocumentPath,
		&result.CreatedBy, &result.ReceiverName, &result.ReceiverBank, &result.ReceiverCountry, &result.ApprovalStatus, &result.CreatedAt, &result.UpdatedAt,
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
			extend_date_to, foreign_amount, currency, remaining_amount,
			document_path, COALESCE(created_by, ''), COALESCE(receiver_name, ''), COALESCE(receiver_bank, ''), COALESCE(receiver_country, ''), COALESCE(approval_status, 'pending_currency_control'), created_at, COALESCE(updated_at, created_at)
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
			&ag.RemainingAmount, &ag.DocumentPath,
			&ag.CreatedBy, &ag.ReceiverName, &ag.ReceiverBank, &ag.ReceiverCountry, &ag.ApprovalStatus, &ag.CreatedAt, &ag.UpdatedAt,
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
			extend_date_to, foreign_amount, currency, remaining_amount,
			document_path, COALESCE(created_by, ''), COALESCE(receiver_name, ''), COALESCE(receiver_bank, ''), COALESCE(receiver_country, ''), COALESCE(approval_status, 'pending_currency_control'), created_at, COALESCE(updated_at, created_at)
		FROM additional_agreements WHERE id = $1 AND deleted_at IS NULL`
	var ag domain.AdditionalAgreement
	err := r.db.QueryRow(ctx, query, id).Scan(
		&ag.ID, &ag.ContractID, &ag.DocType, &ag.AgreementNumber, &ag.AgreementDate,
		&ag.DeliveryDate, &ag.ReturnDate, &ag.Subject,
		&ag.ExtendDateTo, &ag.ForeignAmount, &ag.ForeignCurrency,
		&ag.RemainingAmount, &ag.DocumentPath,
		&ag.CreatedBy, &ag.ReceiverName, &ag.ReceiverBank, &ag.ReceiverCountry, &ag.ApprovalStatus, &ag.CreatedAt, &ag.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.AdditionalAgreement{}, errors.New("Дополнительное соглашение не найдено")
		}
		return ag, err
	}
	ag.Normalize()
	return ag, nil
}

func (r *additionalAgreementRepo) SoftDelete(ctx context.Context, id int64) error {
	cmdTag, err := r.db.Exec(ctx, `UPDATE additional_agreements SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err != nil {
		return err
	}
	if cmdTag.RowsAffected() == 0 {
		return errors.New("Дополнительное соглашение не найдено")
	}
	return nil
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
			document_path = $11, receiver_name = $12, receiver_bank = $13, receiver_country = $14,
			approval_status = 'pending_currency_control', rejection_reason = '',
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING id, contract_id, COALESCE(doc_type, 'additional_agreement'), agreement_number, agreement_date,
			delivery_date, return_date, subject,
			extend_date_to, foreign_amount, currency, remaining_amount,
			document_path, COALESCE(created_by, ''), COALESCE(receiver_name, ''), COALESCE(receiver_bank, ''), COALESCE(receiver_country, ''), COALESCE(approval_status, 'pending_currency_control'), created_at, COALESCE(updated_at, created_at)`
	var result domain.AdditionalAgreement
	err := r.db.QueryRow(ctx, query,
		id, ag.DocType, ag.AgreementNumber, ag.AgreementDate,
		ag.DeliveryDate, ag.ReturnDate, ag.Subject,
		ag.ExtendDateTo, ag.ForeignAmount, ag.ForeignCurrency,
		ag.DocumentPath, ag.ReceiverName, ag.ReceiverBank, ag.ReceiverCountry,
	).Scan(
		&result.ID, &result.ContractID, &result.DocType, &result.AgreementNumber, &result.AgreementDate,
		&result.DeliveryDate, &result.ReturnDate, &result.Subject,
		&result.ExtendDateTo, &result.ForeignAmount, &result.ForeignCurrency,
		&result.RemainingAmount, &result.DocumentPath,
		&result.CreatedBy, &result.ReceiverName, &result.ReceiverBank, &result.ReceiverCountry, &result.ApprovalStatus, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.AdditionalAgreement{}, errors.New("Дополнительное соглашение не найдено")
		}
		return result, err
	}
	result.Normalize()
	syncAdditionalAgreementRemaining(ctx, r.db, result.ID)
	propagateAADatesToContract(ctx, r.db, result.ContractID, result.DeliveryDate, result.ReturnDate, result.ExtendDateTo)
	return result, nil
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
	var currentStatus string
	err := r.db.QueryRow(ctx,
		`SELECT COALESCE(status, 'active') FROM additional_agreements WHERE id = $1 AND deleted_at IS NULL`,
		id,
	).Scan(&currentStatus)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return errors.New("Дополнительное соглашение не найдено")
		}
		return err
	}

	if currentStatus != domain.ContractStatusArchived {
		return errors.New("Дополнительное соглашение не находится в архиве")
	}

	_, err = r.db.Exec(ctx,
		`UPDATE additional_agreements SET status = 'active', archived_at = NULL, updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		id,
	)
	return err
}