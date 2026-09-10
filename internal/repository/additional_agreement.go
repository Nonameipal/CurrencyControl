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

	if ag.DocType == "" {
		ag.DocType = domain.DocTypeAdditionalAgreement
	}

	query := `
		INSERT INTO additional_agreements (
			contract_id, doc_type, agreement_number, agreement_date,
			new_delivery_conditions, new_delivery_term_days, new_return_term_days, subject,
			extend_date_to, foreign_amount, currency, amount_in_contract_currency,
			document_path, original_document_name, created_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15)
		RETURNING id, contract_id, COALESCE(doc_type, 'additional_agreement'), agreement_number, agreement_date,
			new_delivery_conditions, new_delivery_term_days, new_return_term_days, subject,
			extend_date_to, foreign_amount, currency, amount_in_contract_currency,
			document_path, original_document_name, COALESCE(created_by, ''), created_at`

	var result domain.AdditionalAgreement
	err = r.db.QueryRow(ctx, query,
		ag.ContractID, ag.DocType, ag.AgreementNumber, ag.AgreementDate,
		ag.DeliveryConditions, ag.DeliveryTermDays, ag.ReturnTermDays, ag.Subject,
		ag.ExtendDateTo, ag.ForeignAmount, ag.ForeignCurrency, ag.AmountInContractCurrency,
		ag.DocumentPath, ag.OriginalDocumentName, ag.CreatedBy,
	).Scan(
		&result.ID, &result.ContractID, &result.DocType, &result.AgreementNumber, &result.AgreementDate,
		&result.DeliveryConditions, &result.DeliveryTermDays, &result.ReturnTermDays, &result.Subject,
		&result.ExtendDateTo, &result.ForeignAmount, &result.ForeignCurrency,
		&result.AmountInContractCurrency, &result.DocumentPath, &result.OriginalDocumentName,
		&result.CreatedBy, &result.CreatedAt,
	)
	if err == nil {
		syncContractRemaining(ctx, r.db, result.ContractID)
	}
	return result, err
}

func (r *additionalAgreementRepo) GetByContractID(ctx context.Context, contractID int64) ([]domain.AdditionalAgreement, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, contract_id, COALESCE(doc_type, 'additional_agreement'), agreement_number, agreement_date,
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
			&ag.ID, &ag.ContractID, &ag.DocType, &ag.AgreementNumber, &ag.AgreementDate,
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
		SELECT id, contract_id, COALESCE(doc_type, 'additional_agreement'), agreement_number, agreement_date,
			new_delivery_conditions, new_delivery_term_days, new_return_term_days, subject,
			extend_date_to, foreign_amount, currency, amount_in_contract_currency,
			document_path, original_document_name, COALESCE(created_by, ''), created_at
		FROM additional_agreements WHERE id = $1 AND deleted_at IS NULL`
	var ag domain.AdditionalAgreement
	err := r.db.QueryRow(ctx, query, id).Scan(
		&ag.ID, &ag.ContractID, &ag.DocType, &ag.AgreementNumber, &ag.AgreementDate,
		&ag.DeliveryConditions, &ag.DeliveryTermDays, &ag.ReturnTermDays, &ag.Subject,
		&ag.ExtendDateTo, &ag.ForeignAmount, &ag.ForeignCurrency,
		&ag.AmountInContractCurrency, &ag.DocumentPath, &ag.OriginalDocumentName,
		&ag.CreatedBy, &ag.CreatedAt,
	)
	return ag, err
}

func (r *additionalAgreementRepo) SoftDelete(ctx context.Context, id int64) error {
	var contractID int64
	_ = r.db.QueryRow(ctx, `SELECT contract_id FROM additional_agreements WHERE id = $1`, id).Scan(&contractID)
	_, err := r.db.Exec(ctx, `UPDATE additional_agreements SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	if err == nil && contractID > 0 {
		syncContractRemaining(ctx, r.db, contractID)
	}
	return err
}

func (r *additionalAgreementRepo) Update(ctx context.Context, id int64, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error) {
	if ag.DocType == "" {
		ag.DocType = domain.DocTypeAdditionalAgreement
	}
	query := `
		UPDATE additional_agreements SET
			doc_type = $2, agreement_number = $3, agreement_date = $4,
			new_delivery_conditions = $5, new_delivery_term_days = $6, new_return_term_days = $7, subject = $8,
			extend_date_to = $9, foreign_amount = $10, currency = $11,
			amount_in_contract_currency = $12, document_path = $13, original_document_name = $14
		WHERE id = $1
		RETURNING id, contract_id, COALESCE(doc_type, 'additional_agreement'), agreement_number, agreement_date,
			new_delivery_conditions, new_delivery_term_days, new_return_term_days, subject,
			extend_date_to, foreign_amount, currency, amount_in_contract_currency,
			document_path, original_document_name, COALESCE(created_by, ''), created_at`
	var result domain.AdditionalAgreement
	err := r.db.QueryRow(ctx, query,
		id, ag.DocType, ag.AgreementNumber, ag.AgreementDate,
		ag.DeliveryConditions, ag.DeliveryTermDays, ag.ReturnTermDays, ag.Subject,
		ag.ExtendDateTo, ag.ForeignAmount, ag.ForeignCurrency,
		ag.AmountInContractCurrency, ag.DocumentPath, ag.OriginalDocumentName,
	).Scan(
		&result.ID, &result.ContractID, &result.DocType, &result.AgreementNumber, &result.AgreementDate,
		&result.DeliveryConditions, &result.DeliveryTermDays, &result.ReturnTermDays, &result.Subject,
		&result.ExtendDateTo, &result.ForeignAmount, &result.ForeignCurrency,
		&result.AmountInContractCurrency, &result.DocumentPath, &result.OriginalDocumentName,
		&result.CreatedBy, &result.CreatedAt,
	)
	if err == nil {
		syncContractRemaining(ctx, r.db, result.ContractID)
	}
	return result, err
}