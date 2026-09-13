package ports

import (
	"context"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
)

type ContractService interface {
	Create(ctx context.Context, login string, input domain.Contract) (domain.Contract, error)
	GetByID(ctx context.Context, login string, id int64) (domain.Contract, error)
	GetByClientID(ctx context.Context, login string, clientID int64) ([]domain.Contract, error)
	GetArchived(ctx context.Context, branchID int, page, pageSize int) ([]domain.Contract, int, error)
	GetArchivedByClientID(ctx context.Context, clientID int64) ([]domain.Contract, error)
	RestoreContract(ctx context.Context, id int64) error
	SearchDashboard(ctx context.Context, login string, req dto.DashboardSearchRequest) ([]dto.DashboardSearchResult, error)
	CheckCountry(ctx context.Context, name string) (bool, error)
	CheckCurrency(ctx context.Context, code string) (bool, error)
	GetExpiringContracts(ctx context.Context, branchID int) ([]dto.NotificationResponse, error)
	Update(ctx context.Context, id int64, c domain.Contract) (domain.Contract, error)
	SoftDelete(ctx context.Context, id int64) error
}

type CounterpartyService interface {
	// CreateFromABS — единый маршрут создания ЧДММ: сначала обязательный запрос в АБС по ИНН,
	// при успехе — автосоздание карточки контрагента с данными из АБС + название из запроса.
	CreateFromABS(ctx context.Context, login, llc, inn string, branchID int) (domain.Counterparty, error)
	CheckExistsInBranch(ctx context.Context, branchID int, name string) (bool, error)
	CheckExistsByINN(ctx context.Context, inn string) (bool, error)
	GetByID(ctx context.Context, id int64) (domain.Counterparty, error)
	Update(ctx context.Context, id int64, input domain.Counterparty) (domain.Counterparty, error)
	SoftDelete(ctx context.Context, id int64) error
}
