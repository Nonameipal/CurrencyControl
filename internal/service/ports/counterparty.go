package ports

import (
	"context"

	"CurrencyControl/internal/domain"
)

type CounterpartyService interface {
	Create(ctx context.Context, login string, input domain.Counterparty) (domain.Counterparty, error)
	CheckExistsInBranch(ctx context.Context, branchID int, name string) (bool, error)
}

type CounterpartyRepository interface {
	Create(ctx context.Context, c domain.Counterparty) (domain.Counterparty, error)
	CheckExistsInBranch(ctx context.Context, branchID int, name string) (bool, error)
}
