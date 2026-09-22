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

func (s *counterpartyService) CreateLegalEntityFromABS(ctx context.Context, login, inn string, branchID int) (domain.Counterparty, error) {
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

	if isIndividualClient(absInfo.ClientType) {
		return domain.Counterparty{}, fmt.Errorf("клиент с ИНН '%s' является физическим лицом, используйте форму создания физического лица", cleanINN)
	}

	c := domain.Counterparty{
		Name:       "",
		LLC:        absInfo.FullName,
		INN:        &cleanINN,
		BranchID:   branchID,
		ClientType: domain.ClientTypeLegalEntity,
		CreatedBy:  login,
	}
	c.SetPhones(absInfo.Phones)

	return s.repo.Create(ctx, c)
}

func (s *counterpartyService) CreateIndividualFromABS(ctx context.Context, login, inn, llc string, branchID int) (domain.Counterparty, error) {
	return s.createPersonFromABS(ctx, login, inn, llc, branchID, domain.ClientTypeIndividual, "физического лица")
}

func (s *counterpartyService) CreateSoleProprietorFromABS(ctx context.Context, login, inn, llc string, branchID int) (domain.Counterparty, error) {
	return s.createPersonFromABS(ctx, login, inn, llc, branchID, domain.ClientTypeSoleProprietor, "индивидуального предпринимателя")
}

func (s *counterpartyService) createPersonFromABS(ctx context.Context, login, inn, llc string, branchID int, clientType, typeDesc string) (domain.Counterparty, error) {
	cleanINN := strings.TrimSpace(inn)
	if cleanINN == "" {
		return domain.Counterparty{}, fmt.Errorf("ИНН обязателен")
	}
	cleanLLC := strings.TrimSpace(llc)
	if cleanLLC == "" {
		return domain.Counterparty{}, fmt.Errorf("название компании обязательно для %s", typeDesc)
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

	if !isIndividualClient(absInfo.ClientType) {
		return domain.Counterparty{}, fmt.Errorf("клиент с ИНН '%s' является юридическим лицом, используйте форму создания юридического лица", cleanINN)
	}

	c := domain.Counterparty{
		Name:       absInfo.FullName,
		LLC:        cleanLLC,
		INN:        &cleanINN,
		BranchID:   branchID,
		ClientType: clientType,
		CreatedBy:  login,
	}
	c.SetPhones(absInfo.Phones)

	return s.repo.Create(ctx, c)
}

func isIndividualClient(clientType string) bool {
	lower := strings.ToLower(strings.TrimSpace(clientType))
	return lower == "individual" || strings.Contains(lower, "физ")
}

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

	isIndividual := strings.Contains(strings.ToLower(absInfo.ClientType), "физ") || strings.EqualFold(absInfo.ClientType, "individual")

	var nameField, llcField, clientType string
	if isIndividual {
		clientType = domain.ClientTypeIndividual
		nameField = absInfo.FullName
		llcField = strings.TrimSpace(llc)
	} else {
		clientType = domain.ClientTypeLegalEntity
		nameField = ""
		llcField = strings.TrimSpace(llc)
		if llcField == "" {
			llcField = absInfo.FullName
		}
	}

	c := domain.Counterparty{
		Name:       nameField,
		LLC:        llcField,
		INN:        &cleanINN,
		BranchID:   branchID,
		ClientType: clientType,
		CreatedBy:  login,
	}
	c.SetPhones(absInfo.Phones)

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
