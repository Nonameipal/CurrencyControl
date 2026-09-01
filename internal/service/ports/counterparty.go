package ports

import (
	"context"

	"CurrencyControl/internal/domain"
)

type CounterpartyService interface {
	Create(ctx context.Context, login string, input domain.Counterparty) (domain.Counterparty, error)
	CheckExistsInBranch(ctx context.Context, branchID int, name string) (bool, error)
	GetByID(ctx context.Context, id int64) (domain.Counterparty, error)
	Update(ctx context.Context, id int64, input domain.Counterparty) (domain.Counterparty, error)
	SoftDelete(ctx context.Context, id int64) error
}

type CounterpartyRepository interface {
	Create(ctx context.Context, c domain.Counterparty) (domain.Counterparty, error)
	CheckExistsInBranch(ctx context.Context, branchID int, name string) (bool, error)
	GetByID(ctx context.Context, id int64) (domain.Counterparty, error)
	Update(ctx context.Context, id int64, c domain.Counterparty) (domain.Counterparty, error)
	SoftDelete(ctx context.Context, id int64) error
}
