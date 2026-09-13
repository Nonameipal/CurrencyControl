package service

import (
	"context"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/repository"
)

type AdditionalAgreementService interface {
	Create(ctx context.Context, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error)
	GetByContractID(ctx context.Context, contractID int64) ([]domain.AdditionalAgreement, error)
	GetByID(ctx context.Context, id int64) (domain.AdditionalAgreement, error)
	Update(ctx context.Context, id int64, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error)
	SoftDelete(ctx context.Context, id int64) error
	RestoreAdditionalAgreement(ctx context.Context, id int64) error
}

type additionalAgreementService struct{ repo repository.AdditionalAgreementRepository }

func NewAdditionalAgreementService(repo repository.AdditionalAgreementRepository) AdditionalAgreementService {
	return &additionalAgreementService{repo: repo}
}

func (s *additionalAgreementService) Create(ctx context.Context, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error) {
	return s.repo.Create(ctx, ag)
}
func (s *additionalAgreementService) GetByContractID(ctx context.Context, contractID int64) ([]domain.AdditionalAgreement, error) {
	return s.repo.GetByContractID(ctx, contractID)
}
func (s *additionalAgreementService) GetByID(ctx context.Context, id int64) (domain.AdditionalAgreement, error) {
	return s.repo.GetByID(ctx, id)
}
func (s *additionalAgreementService) Update(ctx context.Context, id int64, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error) {
	return s.repo.Update(ctx, id, ag)
}
func (s *additionalAgreementService) SoftDelete(ctx context.Context, id int64) error {
	return s.repo.SoftDelete(ctx, id)
}
func (s *additionalAgreementService) RestoreAdditionalAgreement(ctx context.Context, id int64) error {
	return s.repo.RestoreAdditionalAgreement(ctx, id)
}