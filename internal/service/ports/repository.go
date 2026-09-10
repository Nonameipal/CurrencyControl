package ports

import (
	"context"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
)

type ContractRepository interface {
	Create(ctx context.Context, c domain.Contract) (domain.Contract, error)
	GetByID(ctx context.Context, id int64) (domain.Contract, error)
	GetByClientID(ctx context.Context, clientID int64) ([]domain.Contract, error)
	SearchDashboard(ctx context.Context, req dto.DashboardSearchRequest) ([]dto.DashboardSearchResult, error)
	CheckCountry(ctx context.Context, name string) (bool, error)
	CheckCurrency(ctx context.Context, code string) (bool, error)
	Update(ctx context.Context, id int64, c domain.Contract) (domain.Contract, error)
}

type CounterpartyRepository interface {
	Create(ctx context.Context, c domain.Counterparty) (domain.Counterparty, error)
	CheckExistsInBranch(ctx context.Context, branchID int, name string) (bool, error)
	CheckExistsByINN(ctx context.Context, inn string) (bool, error)
	GetByID(ctx context.Context, id int64) (domain.Counterparty, error)
	Update(ctx context.Context, id int64, c domain.Counterparty) (domain.Counterparty, error)
	SoftDelete(ctx context.Context, id int64) error
}

