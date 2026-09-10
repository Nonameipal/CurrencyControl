package ports

import (
	"context"

	"CurrencyControl/internal/abs"
	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
)

type ContractService interface {
	Create(ctx context.Context, login string, input domain.Contract) (domain.Contract, error)
	GetByID(ctx context.Context, login string, id int64) (domain.Contract, error)
	GetByClientID(ctx context.Context, login string, clientID int64) ([]domain.Contract, error)
	SearchDashboard(ctx context.Context, login string, req dto.DashboardSearchRequest) ([]dto.DashboardSearchResult, error)
	CheckCountry(ctx context.Context, name string) (bool, error)
	CheckCurrency(ctx context.Context, code string) (bool, error)
	GetExpiringContracts(ctx context.Context, branchID int) ([]dto.NotificationResponse, error)
	Update(ctx context.Context, id int64, c domain.Contract) (domain.Contract, error)
	SoftDelete(ctx context.Context, id int64) error
}

type CounterpartyService interface {
	Create(ctx context.Context, login string, input domain.Counterparty) (domain.Counterparty, error)
	CheckExistsInBranch(ctx context.Context, branchID int, name string) (bool, error)
	CheckExistsByINN(ctx context.Context, inn string) (bool, error)
	ABSLookup(ctx context.Context, inn string) (*abs.ABSClientInfo, error)
	GetByID(ctx context.Context, id int64) (domain.Counterparty, error)
	Update(ctx context.Context, id int64, input domain.Counterparty) (domain.Counterparty, error)
	SoftDelete(ctx context.Context, id int64) error
}
