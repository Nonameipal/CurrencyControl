package service

import (
	"context"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/repository"
	"CurrencyControl/internal/service/ports"
)

type contractService struct {
	repo    repository.ContractRepository
	docRepo ports.DocumentRepository
}

func NewContractService(repo repository.ContractRepository, docRepo ports.DocumentRepository) ports.ContractService {
	return &contractService{repo: repo, docRepo: docRepo}
}

func (s *contractService) Create(ctx context.Context, login string, input domain.Contract) (domain.Contract, error) {
	return s.repo.Create(ctx, input)
}

func (s *contractService) GetByID(ctx context.Context, login string, id int64) (domain.Contract, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *contractService) GetAll(ctx context.Context, login string) ([]domain.Contract, error) {
	return s.repo.GetAll(ctx)
}

func (s *contractService) Update(ctx context.Context, login string, id int64, input domain.Contract) (domain.Contract, error) {
	input.ID = id
	return s.repo.Update(ctx, input)
}

func (s *contractService) Delete(ctx context.Context, login string, id int64) error {
	return s.repo.Delete(ctx, id)
}

func (s *contractService) SearchDashboard(ctx context.Context, login string, req dto.DashboardSearchRequest) ([]dto.DashboardSearchResult, error) {
	return s.repo.SearchDashboard(ctx, req)
}
func (s *contractService) CreateWithDocument(ctx context.Context, login string, input domain.Contract, doc *domain.Document) (domain.Contract, error) {
	created, err := s.repo.Create(ctx, input)
	if err != nil {
		return domain.Contract{}, err
	}
	
	if doc != nil {
		doc.EntityID = created.ID
		_, err = s.docRepo.Create(ctx, *doc)
		// We could rollback contract creation on failure if we had transactions here, 
		// but since repo handles individual statements, we'll just log or return error.
		if err != nil {
			return created, err
		}
	}
	
	return created, nil
}
