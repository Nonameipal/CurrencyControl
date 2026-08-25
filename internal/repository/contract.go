package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/errs"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ContractRepository interface {
	Create(ctx context.Context, c domain.Contract) (domain.Contract, error)
	GetByID(ctx context.Context, id int64) (domain.Contract, error)
	GetAll(ctx context.Context) ([]domain.Contract, error)
	Update(ctx context.Context, c domain.Contract) (domain.Contract, error)
	Delete(ctx context.Context, id int64) error
	SearchDashboard(ctx context.Context, req dto.DashboardSearchRequest) ([]dto.DashboardSearchResult, error)
}

type contractRepo struct {
	db *pgxpool.Pool
}

func NewContractRepository(db *pgxpool.Pool) ContractRepository {
	return &contractRepo{db: db}
}

func (r *contractRepo) Create(ctx context.Context, c domain.Contract) (domain.Contract, error) {
	query := `
		INSERT INTO contracts
			(contract_number, contract_date, additional_agreement, subject, total_amount, remaining_amount, contract_currency, contract_end_date)
		VALUES ($1, $2, $3, $4, $5, $5, $6, $7)
		RETURNING id, contract_number, contract_date, additional_agreement, subject, total_amount, remaining_amount, contract_currency, contract_end_date, created_at, updated_at`

	var result domain.Contract
	err := r.db.QueryRow(ctx, query,
		c.ContractNumber,
		c.ContractDate,
		c.AdditionalAgreement,
		c.Subject,
		c.TotalAmount,
		c.ContractCurrency,
		c.ContractEndDate,
	).Scan(
		&result.ID,
		&result.ContractNumber,
		&result.ContractDate,
		&result.AdditionalAgreement,
		&result.Subject,
		&result.TotalAmount,
		&result.RemainingAmount,
		&result.ContractCurrency,
		&result.ContractEndDate,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return domain.Contract{}, err
	}
	return result, nil
}

func (r *contractRepo) GetByID(ctx context.Context, id int64) (domain.Contract, error) {
	query := `
		SELECT id, contract_number, contract_date, additional_agreement, subject, total_amount, remaining_amount, contract_currency, contract_end_date, created_at, updated_at
		FROM contracts
		WHERE id = $1`

	var result domain.Contract
	err := r.db.QueryRow(ctx, query, id).Scan(
		&result.ID,
		&result.ContractNumber,
		&result.ContractDate,
		&result.AdditionalAgreement,
		&result.Subject,
		&result.TotalAmount,
		&result.RemainingAmount,
		&result.ContractCurrency,
		&result.ContractEndDate,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Contract{}, errs.ErrContractNotFound
		}
		return domain.Contract{}, err
	}
	return result, nil
}

func (r *contractRepo) GetAll(ctx context.Context) ([]domain.Contract, error) {
	query := `
		SELECT id, contract_number, contract_date, additional_agreement, subject, total_amount, remaining_amount, contract_currency, contract_end_date, created_at, updated_at
		FROM contracts
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contracts []domain.Contract
	for rows.Next() {
		var c domain.Contract
		if err := rows.Scan(
			&c.ID,
			&c.ContractNumber,
			&c.ContractDate,
			&c.AdditionalAgreement,
			&c.Subject,
			&c.TotalAmount,
			&c.RemainingAmount,
			&c.ContractCurrency,
			&c.ContractEndDate,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		contracts = append(contracts, c)
	}
	return contracts, nil
}

func (r *contractRepo) Update(ctx context.Context, c domain.Contract) (domain.Contract, error) {
	query := `
		UPDATE contracts SET
			contract_number      = $2,
			contract_date        = $3,
			additional_agreement = $4,
			subject              = $5,
			contract_currency    = $6,
			contract_end_date    = $7,
			updated_at           = NOW()
		WHERE id = $1
		RETURNING id, contract_number, contract_date, additional_agreement, subject, total_amount, remaining_amount, contract_currency, contract_end_date, created_at, updated_at`

	var result domain.Contract
	err := r.db.QueryRow(ctx, query,
		c.ID,
		c.ContractNumber,
		c.ContractDate,
		c.AdditionalAgreement,
		c.Subject,
		c.ContractCurrency,
		c.ContractEndDate,
	).Scan(
		&result.ID,
		&result.ContractNumber,
		&result.ContractDate,
		&result.AdditionalAgreement,
		&result.Subject,
		&result.TotalAmount,
		&result.RemainingAmount,
		&result.ContractCurrency,
		&result.ContractEndDate,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Contract{}, errs.ErrContractNotFound
		}
		return domain.Contract{}, err
	}
	return result, nil
}

func (r *contractRepo) Delete(ctx context.Context, id int64) error {
	query := `DELETE FROM contracts WHERE id = $1`

	result, err := r.db.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return errs.ErrContractNotFound
	}
	return nil
}

func (r *contractRepo) SearchDashboard(ctx context.Context, req dto.DashboardSearchRequest) ([]dto.DashboardSearchResult, error) {
	query := `
		SELECT c.id, COALESCE(cp.client_number, ''), COALESCE(cp.name, '')
		FROM contracts c
		LEFT JOIN counterparties cp ON c.client_id = cp.id
		WHERE 1=1`

	var args []interface{}
	var conditions []string
	argId := 1

	if req.Amount > 0 {
		conditions = append(conditions, fmt.Sprintf(`c.total_amount = $%d`, argId))
		args = append(args, req.Amount)
		argId++
	}
	if req.INN != "" {
		conditions = append(conditions, fmt.Sprintf(`cp.inn = $%d`, argId))
		args = append(args, req.INN)
		argId++
	}
	if req.CompanyName != "" {
		conditions = append(conditions, fmt.Sprintf(`cp.name ILIKE $%d`, argId))
		args = append(args, req.CompanyName)
		argId++
	}
	if req.BranchID > 0 {
		conditions = append(conditions, fmt.Sprintf(`c.branch_id = $%d`, argId))
		args = append(args, req.BranchID)
		argId++
	}

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY c.created_at DESC LIMIT 100"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []dto.DashboardSearchResult
	for rows.Next() {
		var res dto.DashboardSearchResult
		if err := rows.Scan(&res.ContractID, &res.Number, &res.CompanyName); err != nil {
			return nil, err
		}
		results = append(results, res)
	}
	if results == nil {
		results = make([]dto.DashboardSearchResult, 0)
	}
	return results, nil
}
