package service

import (
	"context"
	"fmt"
	"strings"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"
)

type branchService struct {
	repo ports.BranchRepository
}

func NewBranchService(repo ports.BranchRepository) ports.BranchService {
	return &branchService{repo: repo}
}

func (s *branchService) Create(ctx context.Context, login string, req dto.CreateBranchRequest) (domain.Branch, error) {
	if req.ID <= 0 {
		return domain.Branch{}, fmt.Errorf("код (ID) филиала обязателен и должен быть положительным числом")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return domain.Branch{}, fmt.Errorf("название филиала обязательно")
	}

	exists, err := s.repo.CheckExists(ctx, req.ID)
	if err != nil {
		return domain.Branch{}, err
	}
	if exists {
		return domain.Branch{}, fmt.Errorf("филиал с кодом %d уже существует", req.ID)
	}

	branch := domain.Branch{
		ID:        req.ID,
		Name:      name,
		CreatedBy: login,
	}

	return s.repo.Create(ctx, branch)
}

func (s *branchService) GetByID(ctx context.Context, id int) (*domain.Branch, error) {
	if id <= 0 {
		return nil, fmt.Errorf("некорректный ID филиала")
	}
	b, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("филиал с ID %d не найден", id)
	}
	return b, nil
}

func (s *branchService) GetAll(ctx context.Context) ([]domain.Branch, error) {
	return s.repo.GetAll(ctx)
}

func (s *branchService) Update(ctx context.Context, id int, req dto.UpdateBranchRequest) (*domain.Branch, error) {
	if id <= 0 {
		return nil, fmt.Errorf("некорректный ID филиала")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, fmt.Errorf("название филиала не может быть пустым")
	}

	exists, err := s.repo.CheckExists(ctx, id)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, fmt.Errorf("филиал с ID %d не найден", id)
	}

	return s.repo.Update(ctx, id, name)
}

func (s *branchService) Delete(ctx context.Context, id int) error {
	if id <= 0 {
		return fmt.Errorf("некорректный ID филиала")
	}

	exists, err := s.repo.CheckExists(ctx, id)
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("филиал с ID %d не найден", id)
	}

	hasRelations, err := s.repo.CheckHasRelations(ctx, id)
	if err != nil {
		return err
	}
	if hasRelations {
		return fmt.Errorf("невозможно удалить филиал: к нему привязаны пользователи, компании или заявки")
	}

	return s.repo.SoftDelete(ctx, id)
}
