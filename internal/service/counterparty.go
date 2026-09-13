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

// CreateFromABS — единая точка создания контрагента.
// 1. Проверяет, что клиент с таким ИНН ещё не зарегистрирован.
// 2. Обязательно запрашивает АБС по ИНН: если не найден — возвращает ошибку.
// 3. Автоматически подставляет из АБС: ФИО/наименование , тип клиента, телефоны, счета.
// 4. Создаёт карточку ЧДММ в БД.
func (s *counterpartyService) CreateFromABS(ctx context.Context, login, llc, inn string, branchID int) (domain.Counterparty, error) {
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

	llc = strings.TrimSpace(llc)
	if llc == "" {
		return domain.Counterparty{}, fmt.Errorf("поле llc (название ЧДММ) обязательно")
	}

	// Автоматически добавляем префикс "ЧДММ " если не написано
	llcUpper := strings.ToUpper(llc)
	if !strings.HasPrefix(llcUpper, "ЧДММ") && !strings.HasPrefix(llcUpper, "ҶДММ") {
		llc = "ЧДММ " + llc
	}

	// name = ФИО из АБС; llc = название ЧДММ от операциониста
	c := domain.Counterparty{
		Name:       absInfo.FullName, // ФИО из АБС
		LLC:        llc,             // Название ЧДММ (с автопрефиксом)
		INN:        &cleanINN,
		BranchID:   branchID,
		ClientType: absInfo.ClientType,
		CreatedBy:  login,
	}
	c.SetPhones(absInfo.Phones)
	c.SetAccounts(absInfo.Accounts)

	if c.ClientType == "" {
		c.ClientType = domain.ClientTypeLegalEntity
	}

	return s.repo.Create(ctx, c)
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
