package repository

import (
	"context"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"

	"github.com/jackc/pgx/v5/pgxpool"
)

type counterpartyRepo struct {
	db *pgxpool.Pool
}

func NewCounterpartyRepository(db *pgxpool.Pool) ports.CounterpartyRepository {
	return &counterpartyRepo{db: db}
}

func (r *counterpartyRepo) Create(ctx context.Context, c domain.Counterparty) (domain.Counterparty, error) {
	query := `
		INSERT INTO counterparties (name, inn, branch_id)
		VALUES ($1, $2, $3)
		RETURNING id, name, inn, branch_id, COALESCE(email, ''), created_at, updated_at`

	var result domain.Counterparty
	err := r.db.QueryRow(ctx, query,
		c.Name,
		c.INN,
		c.BranchID,
	).Scan(
		&result.ID,
		&result.Name,
		&result.INN,
		&result.BranchID,
		&result.Email,
		&result.CreatedAt,
		&result.UpdatedAt,
	)

	if err != nil {
		return domain.Counterparty{}, err
	}

	return result, nil
}

func (r *counterpartyRepo) CheckExistsInBranch(ctx context.Context, branchID int, name string) (bool, error) {
	var exists bool
	err := r.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM counterparties WHERE branch_id = $1 AND name = $2)", branchID, name).Scan(&exists)
	return exists, err
}

func (r *counterpartyRepo) Update(ctx context.Context, id int64, c domain.Counterparty) (domain.Counterparty, error) {
	query := `
		UPDATE counterparties
		SET 
			name = COALESCE(NULLIF($2, ''), name),
			inn = COALESCE(NULLIF($3, ''), inn),
			email = COALESCE(NULLIF($4, ''), email),
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, inn, branch_id, COALESCE(email, ''), created_at, updated_at`

	var result domain.Counterparty
	err := r.db.QueryRow(ctx, query, id, c.Name, c.INN, c.Email).Scan(
		&result.ID,
		&result.Name,
		&result.INN,
		&result.BranchID,
		&result.Email,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return domain.Counterparty{}, err
	}
	return result, nil
}
