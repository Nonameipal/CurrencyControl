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
		INSERT INTO counterparties (name, inn, branch_id, created_by)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, inn, branch_id, COALESCE(email, ''), COALESCE(created_by, ''), created_at, updated_at`

	var result domain.Counterparty
	err := r.db.QueryRow(ctx, query,
		c.Name,
		c.INN,
		c.BranchID,
		c.CreatedBy,
	).Scan(
		&result.ID,
		&result.Name,
		&result.INN,
		&result.BranchID,
		&result.Email,
		&result.CreatedBy,
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

func (r *counterpartyRepo) GetByID(ctx context.Context, id int64) (domain.Counterparty, error) {
	query := `SELECT id, name, inn, branch_id, COALESCE(email, ''), COALESCE(created_by, ''), created_at, updated_at FROM counterparties WHERE id = $1 AND deleted_at IS NULL`
	var result domain.Counterparty
	err := r.db.QueryRow(ctx, query, id).Scan(
		&result.ID,
		&result.Name,
		&result.INN,
		&result.BranchID,
		&result.Email,
		&result.CreatedBy,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return domain.Counterparty{}, err
	}
	return result, nil
}

func (r *counterpartyRepo) Update(ctx context.Context, id int64, c domain.Counterparty) (domain.Counterparty, error) {
	query := `
		UPDATE counterparties
		SET 
			name = $2,
			inn = $3,
			email = $4,
			updated_at = NOW()
		WHERE id = $1
		RETURNING id, name, inn, branch_id, COALESCE(email, ''), COALESCE(created_by, ''), created_at, updated_at`

	var result domain.Counterparty
	err := r.db.QueryRow(ctx, query, id, c.Name, c.INN, c.Email).Scan(
		&result.ID,
		&result.Name,
		&result.INN,
		&result.BranchID,
		&result.Email,
		&result.CreatedBy,
		&result.CreatedAt,
		&result.UpdatedAt,
	)
	if err != nil {
		return domain.Counterparty{}, err
	}
	return result, nil
}

func (r *counterpartyRepo) SoftDelete(ctx context.Context, id int64) error {
	_, err := r.db.Exec(ctx, `UPDATE counterparties SET deleted_at = NOW() WHERE id = $1 AND deleted_at IS NULL`, id)
	return err
}
