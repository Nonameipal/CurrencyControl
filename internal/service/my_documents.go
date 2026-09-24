package service

import (
	"context"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/service/ports"
)

type myDocumentsService struct {
	repo ports.MyDocumentsRepository
}

func NewMyDocumentsService(repo ports.MyDocumentsRepository) ports.MyDocumentsService {
	return &myDocumentsService{repo: repo}
}

func (s *myDocumentsService) GetMyDocuments(ctx context.Context, login string, filter dto.MyDocumentsFilter) (*dto.MyDocumentsResponse, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.PageSize < 1 || filter.PageSize > 100 {
		filter.PageSize = 20
	}
	items, total, err := s.repo.GetMyDocuments(ctx, login, filter)
	if err != nil {
		return nil, err
	}
	return &dto.MyDocumentsResponse{
		Items:    items,
		Total:    total,
		Page:     filter.Page,
		PageSize: filter.PageSize,
	}, nil
}
