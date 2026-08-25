package ports

import (
	"context"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
)

type ContractService interface {
	Create(ctx context.Context, login string, input domain.Contract) (domain.Contract, error)
	GetByID(ctx context.Context, login string, id int64) (domain.Contract, error)
	GetAll(ctx context.Context, login string) ([]domain.Contract, error)
	Update(ctx context.Context, login string, id int64, input domain.Contract) (domain.Contract, error)
	Delete(ctx context.Context, login string, id int64) error
	SearchDashboard(ctx context.Context, login string, req dto.DashboardSearchRequest) ([]dto.DashboardSearchResult, error)
}
type ContractRepository interface {
	Create(ctx context.Context, c domain.Contract) (domain.Contract, error)
	GetByID(ctx context.Context, id int64) (domain.Contract, error)
	GetAll(ctx context.Context) ([]domain.Contract, error)
	Update(ctx context.Context, c domain.Contract) (domain.Contract, error)
	Delete(ctx context.Context, id int64) error
}
