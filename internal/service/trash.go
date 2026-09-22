package service

import (
	"context"
	"fmt"
	"os"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/service/ports"
)

type trashService struct {
	repo ports.TrashRepository
}

func NewTrashService(repo ports.TrashRepository) ports.TrashService {
	return &trashService{repo: repo}
}

func (s *trashService) GetTrashList(ctx context.Context, filter dto.TrashFilter) ([]dto.TrashItem, int, error) {
	return s.repo.GetTrashItems(ctx, filter)
}

func (s *trashService) GetTrashItem(ctx context.Context, entityType string, id int64) (*dto.TrashItem, error) {
	return s.repo.GetTrashItemByID(ctx, entityType, id)
}

func (s *trashService) GetFilePath(ctx context.Context, entityType string, id int64) (string, error) {
	item, err := s.repo.GetTrashItemByID(ctx, entityType, id)
	if err != nil {
		return "", err
	}
	if item.DocumentPath == "" {
		return "", fmt.Errorf("файл документа не прикреплен к данной записи")
	}
	if _, err := os.Stat(item.DocumentPath); os.IsNotExist(err) {
		return "", fmt.Errorf("файл документа отсутствует на сервере: %s", item.DocumentPath)
	}
	return item.DocumentPath, nil
}

func (s *trashService) RestoreItem(ctx context.Context, entityType string, id int64) error {
	return s.repo.RestoreItem(ctx, entityType, id)
}
