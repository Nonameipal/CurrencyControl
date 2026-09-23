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

func (s *counterpartyService) CheckExistsInBranch(ctx context.Context, branchID int, llc string) (bool, error) {
	return s.repo.CheckExistsInBranch(ctx, branchID, llc)
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

func (s *counterpartyService) CreateByINNFromABS(ctx context.Context, login, inn string, branchID int) (domain.Counterparty, error) {
	cleanINN := strings.TrimSpace(inn)
	if cleanINN == "" {
		return domain.Counterparty{}, fmt.Errorf("ИНН обязателен")
	}

	exists, err := s.repo.CheckExistsByINN(ctx, cleanINN)
	if err != nil {
		return domain.Counterparty{}, err
	}
	if exists {
		return domain.Counterparty{}, fmt.Errorf("клиент с ИНН '%s' уже зарегистрирован в базе данных", cleanINN)
	}
	if s.abs == nil {
		return domain.Counterparty{}, fmt.Errorf("клиент АБС не инициализирован")
	}
	absInfo, err := s.abs.GetClientByINN(ctx, cleanINN)
	if err != nil {
		return domain.Counterparty{}, fmt.Errorf("ошибка обращения в АБС: %w", err)
	}
	if absInfo == nil {
		return domain.Counterparty{}, fmt.Errorf("клиент с ИНН '%s' не найден в АБС", cleanINN)
	}

	var clientType string
	if isSoleProprietorClient(absInfo.ClientType) {
		clientType = domain.ClientTypeSoleProprietor
	} else if isIndividualClient(absInfo.ClientType) {
		clientType = domain.ClientTypeIndividual
	} else {
		clientType = domain.ClientTypeLegalEntity
	}

	c := domain.Counterparty{
		LLC:        absInfo.FullName,
		INN:        &cleanINN,
		BranchID:   branchID,
		ClientType: clientType,
		CreatedBy:  login,
	}
	c.SetPhones(absInfo.Phones)

	return s.repo.Create(ctx, c)
}

func isSoleProprietorClient(clientType string) bool {
	lower := strings.ToLower(strings.TrimSpace(clientType))
	return strings.Contains(lower, "предприниматель") || strings.Contains(lower, "ип") || lower == domain.ClientTypeSoleProprietor
}

func isIndividualClient(clientType string) bool {
	lower := strings.ToLower(strings.TrimSpace(clientType))
	return lower == "individual" || strings.Contains(lower, "физ") || isSoleProprietorClient(clientType)
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
