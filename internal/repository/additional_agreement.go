package repository
import (
	"context"
	"fmt"
	"strings"

	"CurrencyControl/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)
type AdditionalAgreementRepository interface {
	Create(ctx context.Context, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error)
	GetByContractID(ctx context.Context, contractID int64) ([]domain.AdditionalAgreement, error)
	GetByID(ctx context.Context, id int64) (domain.AdditionalAgreement, error)
	Update(ctx context.Context, id int64, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error)
	SoftDelete(ctx context.Context, id int64) error
}
func NewAdditionalAgreementRepository(db *pgxpool.Pool) AdditionalAgreementRepository {
	return &additionalAgreementRepo{db: db}
}
type additionalAgreementRepo struct{ db *pgxpool.Pool }

func (r *additionalAgreementRepo) Create(ctx context.Context, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error) {
	var contractCurrency string
	err := r.db.QueryRow(ctx, "SELECT contract_currency FROM contracts WHERE id = $1", ag.ContractID).Scan(&contractCurrency)
	if err != nil {
		return domain.AdditionalAgreement{}, fmt.Errorf("ошибка получения контракта: %w", err)
	}

	if ag.ForeignCurrency != nil && ag.ForeignAmount != nil {
		if strings.EqualFold(*ag.ForeignCurrency, contractCurrency) && ag.AmountInContractCurrency == 0 {
			ag.AmountInContractCurrency = *ag.ForeignAmount
		} else if !strings.EqualFold(*ag.ForeignCurrency, contractCurrency) && ag.AmountInContractCurrency == 0 {
			return domain.AdditionalAgreement{}, fmt.Errorf(
				"поле amount_in_contract_currency обязательно: валюта платежа (%s) отличается от валюты контракта (%s). Укажите сумму в %s для увеличения лимита контракта",
				*ag.ForeignCurrency, contractCurrency, contractCurrency,
			)
		}
	}

	query := `
		INSERT INTO additional_agreements (
			contract_id, agreement_number, agreement_date,
			new_delivery_conditions, new_delivery_term_days, new_return_term_days, subject,
			extend_date_to, foreign_amount, currency, amount_in_contract_currency,
			document_path, original_document_name, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id, contract_id, agreement_number, agreement_date,
			new_delivery_conditions, new_delivery_term_days, new_return_term_days, subject,
			extend_date_to, foreign_amount, currency, amount_in_contract_currency,
			document_path, original_document_name, COALESCE(created_by, ''), created_at`

	var result domain.AdditionalAgreement
	err = r.db.QueryRow(ctx, query,
		ag.ContractID, ag.AgreementNumber, ag.AgreementDate,
		ag.DeliveryConditions, ag.DeliveryTermDays, ag.ReturnTermDays, ag.Subject,
		ag.ExtendDateTo, ag.ForeignAmount, ag.ForeignCurrency, ag.AmountInContractCurrency,
		ag.DocumentPath, ag.OriginalDocumentName, ag.CreatedBy,
	).Scan(
		&result.ID, &result.ContractID, &result.AgreementNumber, &result.AgreementDate,
		&result.DeliveryConditions, &result.DeliveryTermDays, &result.ReturnTermDays, &result.Subject,
		&result.ExtendDateTo, &result.ForeignAmount, &result.ForeignCurrency,
		&result.AmountInContractCurrency, &result.DocumentPath, &result.OriginalDocumentName,
		&result.CreatedBy, &result.CreatedAt,
	)
	return result, err
}

func (r *additionalAgreementRepo) GetByContractID(ctx context.Context, contractID int64) ([]domain.AdditionalAgreement, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, contract_id, agreement_number, agreement_date,
			new_delivery_conditions, new_delivery_term_days, new_return_term_days, subject,
			extend_date_to, foreign_amount, currency, amount_in_contract_currency,
			document_path, original_document_name, COALESCE(created_by, ''), created_at
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
			&ag.ID, &ag.ContractID, &ag.AgreementNumber, &ag.AgreementDate,
			&ag.DeliveryConditions, &ag.DeliveryTermDays, &ag.ReturnTermDays, &ag.Subject,
			&ag.ExtendDateTo, &ag.ForeignAmount, &ag.ForeignCurrency,
			&ag.AmountInContractCurrency, &ag.DocumentPath, &ag.OriginalDocumentName,
			&ag.CreatedBy, &ag.CreatedAt,
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

func (r *additionalAgreementRepo) GetByID(ctx context.Context, id int64) (domain.AdditionalAgreement, error) {
	query := `
		SELECT id, contract_id, agreement_number, agreement_date,
			new_delivery_conditions, new_delivery_term_days, new_return_term_days, subject,
			extend_date_to, foreign_amount, currency, amount_in_contract_currency,
			document_path, original_document_name, COALESCE(created_by, ''), created_at
		FROM additional_agreements WHERE id = $1 AND deleted_at IS NULL`
	var ag domain.AdditionalAgreement
	err := r.db.QueryRow(ctx, query, id).Scan(
		&ag.ID, &ag.ContractID, &ag.AgreementNumber, &ag.AgreementDate,
		&ag.DeliveryConditions, &ag.DeliveryTermDays, &ag.ReturnTermDays, &ag.Subject,
		&ag.ExtendDateTo, &ag.ForeignAmount, &ag.ForeignCurrency,
		&ag.AmountInContractCurrency, &ag.DocumentPath, &ag.OriginalDocumentName,
		&ag.CreatedBy, &ag.CreatedAt,
	)
	return ag, err
}

func (r *additionalAgreementRepo) SoftDelete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `UPDATE additional_agreements SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}

func (r *additionalAgreementRepo) Update(ctx context.Context, id int64, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error) {
	query := `
		UPDATE additional_agreements SET
			agreement_number = $2, agreement_date = $3,
			new_delivery_conditions = $4, new_delivery_term_days = $5, new_return_term_days = $6, subject = $7,
			extend_date_to = $8, foreign_amount = $9, currency = $10,
			amount_in_contract_currency = $11, document_path = $12, original_document_name = $13
		WHERE id = $1
		RETURNING id, contract_id, agreement_number, agreement_date,
			new_delivery_conditions, new_delivery_term_days, new_return_term_days, subject,
			extend_date_to, foreign_amount, currency, amount_in_contract_currency,
			document_path, original_document_name, COALESCE(created_by, ''), created_at`
	var result domain.AdditionalAgreement
	err := r.db.QueryRow(ctx, query,
		id, ag.AgreementNumber, ag.AgreementDate,
		ag.DeliveryConditions, ag.DeliveryTermDays, ag.ReturnTermDays, ag.Subject,
		ag.ExtendDateTo, ag.ForeignAmount, ag.ForeignCurrency,
		ag.AmountInContractCurrency, ag.DocumentPath, ag.OriginalDocumentName,
	).Scan(
		&result.ID, &result.ContractID, &result.AgreementNumber, &result.AgreementDate,
		&result.DeliveryConditions, &result.DeliveryTermDays, &result.ReturnTermDays, &result.Subject,
		&result.ExtendDateTo, &result.ForeignAmount, &result.ForeignCurrency,
		&result.AmountInContractCurrency, &result.DocumentPath, &result.OriginalDocumentName,
		&result.CreatedBy, &result.CreatedAt,
	)
	return result, err
}