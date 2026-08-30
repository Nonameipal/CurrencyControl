package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/errs"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ContractRepository interface {
	Create(ctx context.Context, c domain.Contract) (domain.Contract, error)
	GetByID(ctx context.Context, id int64) (domain.Contract, error)
	GetExpiringContracts(ctx context.Context, branchID int) ([]dto.NotificationResponse, error)
	GetByClientID(ctx context.Context, clientID int64) ([]domain.Contract, error)
	SearchDashboard(ctx context.Context, req dto.DashboardSearchRequest) ([]dto.DashboardSearchResult, error)
	CheckCountry(ctx context.Context, name string) (bool, error)
	CheckCurrency(ctx context.Context, code string) (bool, error)
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
			(client_id, branch_id, contract_number, contract_name, contract_date, delivery_date, delivery_conditions, delivery_term_days, return_term_days, total_amount, remaining_amount, contract_currency, sender_account, receiver_name, receiver_account, receiver_country, subject, contract_end_date, document_path, original_document_name)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $10, $11, $12, $13, $14, $15, $16, $17, $18)
		RETURNING id, client_id, branch_id, contract_number, contract_name, contract_date, delivery_date, delivery_conditions, delivery_term_days, return_term_days, total_amount, remaining_amount, contract_currency, sender_account, receiver_name, receiver_account, receiver_country, subject, contract_end_date, document_path, original_document_name, created_at, updated_at`

	var result domain.Contract
	err := r.db.QueryRow(ctx, query,
		c.ClientID, c.BranchID, c.ContractNumber, c.ContractName, c.ContractDate, c.DeliveryDate, c.DeliveryConditions, c.DeliveryTermDays, c.ReturnTermDays, c.TotalAmount, c.ContractCurrency, c.SenderAccount, c.ReceiverName, c.ReceiverAccount, c.ReceiverCountry, c.Subject, c.ContractEndDate, c.DocumentPath, c.OriginalDocumentName,
	).Scan(
		&result.ID, &result.ClientID, &result.BranchID, &result.ContractNumber, &result.ContractName, &result.ContractDate, &result.DeliveryDate, &result.DeliveryConditions, &result.DeliveryTermDays, &result.ReturnTermDays, &result.TotalAmount, &result.RemainingAmount, &result.ContractCurrency, &result.SenderAccount, &result.ReceiverName, &result.ReceiverAccount, &result.ReceiverCountry, &result.Subject, &result.ContractEndDate, &result.DocumentPath, &result.OriginalDocumentName, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		return domain.Contract{}, err
	}
	return result, nil
}

func (r *contractRepo) GetByID(ctx context.Context, id int64) (domain.Contract, error) {
	query := `
		SELECT id, contract_number, contract_date, COALESCE(subject, ''), total_amount, remaining_amount, contract_currency, COALESCE(sender_account, ''), contract_end_date, document_path, original_document_name, created_at, updated_at
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
		&result.SenderAccount,
		&result.ContractEndDate,
		&result.DocumentPath,
		&result.OriginalDocumentName,
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



func (r *contractRepo) GetByClientID(ctx context.Context, clientID int64) ([]domain.Contract, error) {
	query := `
		SELECT id, client_id, branch_id, contract_number, contract_date, COALESCE(subject, ''), total_amount, remaining_amount, contract_currency, COALESCE(sender_account, ''), contract_end_date, document_path, original_document_name, created_at, updated_at
		FROM contracts
		WHERE client_id = $1
		ORDER BY created_at DESC`

	rows, err := r.db.Query(ctx, query, clientID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var contracts []domain.Contract
	for rows.Next() {
		var c domain.Contract
		if err := rows.Scan(
			&c.ID,
			&c.ClientID,
			&c.BranchID,
			&c.ContractNumber,
			&c.ContractDate,
			&c.AdditionalAgreement,
			&c.Subject,
			&c.TotalAmount,
			&c.RemainingAmount,
			&c.ContractCurrency,
			&c.SenderAccount,
			&c.ContractEndDate, c.DocumentPath, c.OriginalDocumentName,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		contracts = append(contracts, c)
	}
	if contracts == nil {
		contracts = []domain.Contract{} 
	}
	return contracts, nil
}





func (r *contractRepo) SearchDashboard(ctx context.Context, req dto.DashboardSearchRequest) ([]dto.DashboardSearchResult, error) {
	query := `
		SELECT DISTINCT cp.id, COALESCE(cp.name, ''), COALESCE(cp.inn, '')
		FROM counterparties cp
		LEFT JOIN contracts c ON c.client_id = cp.id
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
		conditions = append(conditions, fmt.Sprintf(`cp.inn ILIKE $%d`, argId))
		args = append(args, "%"+req.INN+"%")
		argId++
	}
	if req.CompanyName != "" {
		conditions = append(conditions, fmt.Sprintf(`cp.name ILIKE $%d`, argId))
		args = append(args, "%"+req.CompanyName+"%")
		argId++
	}
	if req.BranchID > 0 {
		conditions = append(conditions, fmt.Sprintf(`cp.branch_id = $%d`, argId))
		args = append(args, req.BranchID)
		argId++
	}

	if len(conditions) > 0 {
		query += " AND " + strings.Join(conditions, " AND ")
	}
	query += " ORDER BY cp.id DESC LIMIT 100"

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []dto.DashboardSearchResult
	counter := 1 

	for rows.Next() {
		var res dto.DashboardSearchResult
		if err := rows.Scan(&res.CompanyID, &res.CompanyName, &res.INN); err != nil {
			return nil, err
		}
		res.Number = fmt.Sprintf("РІвЂћвЂ“%d", counter)
		counter++
		
		results = append(results, res)
	}
	if results == nil {
		results = make([]dto.DashboardSearchResult, 0)
	}
	return results, nil
}

func (r *contractRepo) CheckCountry(ctx context.Context, name string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM countries WHERE name_ru = $1)", name).Scan(&exists)
	return exists, err
}

func (r *contractRepo) CheckCurrency(ctx context.Context, code string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM currencies WHERE code = $1)", code).Scan(&exists)
	return exists, err
}
func (r *contractRepo) GetExpiringContracts(ctx context.Context, branchID int) ([]dto.NotificationResponse, error) {
	query := `
		SELECT 
			comp.id AS company_id,
			comp.name AS company_name,
			c.id AS contract_id,
			c.contract_number,
			GREATEST(c.contract_end_date, COALESCE(MAX(aa.extend_date_to), c.contract_end_date)) AS effective_end_date
		FROM contracts c
		JOIN counterparties comp ON c.client_id = comp.id
		LEFT JOIN additional_agreements aa ON aa.contract_id = c.id
		WHERE comp.branch_id = $1
		GROUP BY c.id, comp.id
		HAVING GREATEST(c.contract_end_date, COALESCE(MAX(aa.extend_date_to), c.contract_end_date)) <= CURRENT_DATE + INTERVAL '10 days'
		ORDER BY effective_end_date ASC
	`
	rows, err := r.db.Query(ctx, query, branchID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []dto.NotificationResponse
	now := time.Now().Truncate(24 * time.Hour)

	for rows.Next() {
		var n dto.NotificationResponse
		if err := rows.Scan(&n.CompanyID, &n.CompanyName, &n.ContractID, &n.ContractNumber, &n.EffectiveEndDate); err != nil {
			return nil, err
		}
		
		daysLeft := int(n.EffectiveEndDate.Sub(now).Hours() / 24)
		n.DaysLeft = daysLeft
		result = append(result, n)
	}

	if result == nil {
		result = []dto.NotificationResponse{}
	}
	return result, nil
}
