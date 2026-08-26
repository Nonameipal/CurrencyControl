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
		INSERT INTO counterparties (name, inn, branch_id, is_third_party)
		VALUES ($1, $2, $3, $4)
		RETURNING id, name, inn, branch_id, COALESCE(email, ''), is_third_party, created_at, updated_at`

	var result domain.Counterparty
	err := r.db.QueryRow(ctx, query,
		c.Name,
		c.INN,
		c.BranchID,
		c.IsThirdParty,
	).Scan(
		&result.ID,
		&result.Name,
		&result.INN,
		&result.BranchID,
		&result.Email,
		&result.IsThirdParty,
		&result.CreatedAt,
		&result.UpdatedAt,
	)

	if err != nil {
		return domain.Counterparty{}, err
	}

	return result, nil
}
