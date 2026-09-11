package service

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"CurrencyControl/internal/abs"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"
)

type counterpartyService struct {
	repo ports.CounterpartyRepository
	abs  abs.ABSClient
}

func NewCounterpartyService(repo ports.CounterpartyRepository, absClient abs.ABSClient) ports.CounterpartyService {
	return &counterpartyService{repo: repo, abs: absClient}
}

func (s *counterpartyService) CheckExistsInBranch(ctx context.Context, branchID int, name string) (bool, error) {
	return s.repo.CheckExistsInBranch(ctx, branchID, name)
}

func (s *counterpartyService) CheckExistsByINN(ctx context.Context, inn string) (bool, error) {
	return s.repo.CheckExistsByINN(ctx, inn)
}

func (s *counterpartyService) ABSLookup(ctx context.Context, inn string) (*abs.ABSClientInfo, error) {
	if s.abs == nil {
		return nil, errors.New("клиент АБС не инициализирован")
	}
	return s.abs.GetClientByINN(ctx, inn)
}

func (s *counterpartyService) Create(ctx context.Context, login string, input domain.Counterparty) (domain.Counterparty, error) {
	if input.INN != nil && strings.TrimSpace(*input.INN) != "" {
		cleanINN := strings.TrimSpace(*input.INN)
		input.INN = &cleanINN

		exists, err := s.repo.CheckExistsByINN(ctx, cleanINN)
		if err != nil {
			return domain.Counterparty{}, err
		}
		if exists {
			return domain.Counterparty{}, fmt.Errorf("клиент с ИНН '%s' уже зарегистрирован в базе данных", cleanINN)
		}

		if (strings.TrimSpace(input.Name) == "" || strings.TrimSpace(input.ClientType) == "" || len(input.GetPhones()) == 0 || len(input.GetAccounts()) == 0) && s.abs != nil {
			absInfo, err := s.abs.GetClientByINN(ctx, cleanINN)
			if err == nil && absInfo != nil {
				if strings.TrimSpace(input.Name) == "" && absInfo.FullName != "" {
					input.Name = absInfo.FullName
				}
				if strings.TrimSpace(input.ClientType) == "" && absInfo.ClientType != "" {
					input.ClientType = absInfo.ClientType
				}
				if len(input.GetPhones()) == 0 && len(absInfo.Phones) > 0 {
					input.SetPhones(absInfo.Phones)
				}
				if len(input.GetAccounts()) == 0 && len(absInfo.Accounts) > 0 {
					input.SetAccounts(absInfo.Accounts)
				}
			}
		}
	}

	if input.ClientType == "" {
		input.ClientType = domain.ClientTypeLegalEntity
	}

	input.CreatedBy = login
	return s.repo.Create(ctx, input)
}

func (s *counterpartyService) GetByID(ctx context.Context, id int64) (domain.Counterparty, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *counterpartyService) Update(ctx context.Context, id int64, input domain.Counterparty) (domain.Counterparty, error) {
	return s.repo.Update(ctx, id, input)
}

func (s *counterpartyService) SoftDelete(ctx context.Context, id int64) error {
	return s.repo.SoftDelete(ctx, id)
}
