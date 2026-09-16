package repository

import (
	"context"
	"errors"
	"strings"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"

	"gorm.io/gorm"
)

type counterpartyRepo struct {
	db *gorm.DB
}

func NewCounterpartyRepository(db *gorm.DB) ports.CounterpartyRepository {
	return &counterpartyRepo{db: db}
}

func (r *counterpartyRepo) Create(ctx context.Context, c domain.Counterparty) (domain.Counterparty, error) {
	if c.ClientType == "" {
		c.ClientType = domain.ClientTypeLegalEntity
	}
	if c.Phones == "" {
		c.Phones = "[]"
	}
	if c.Accounts == "" {
		c.Accounts = "[]"
	}

	if err := r.db.WithContext(ctx).Create(&c).Error; err != nil {
		return domain.Counterparty{}, err
	}

	c.PhonesList = c.GetPhones()
	c.AccountsList = c.GetAccounts()
	return c, nil
}

func (r *counterpartyRepo) CheckExistsInBranch(ctx context.Context, branchID int, name string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Counterparty{}).
		Where("branch_id = ? AND name = ?", branchID, name).
		Count(&count).Error
	return count > 0, err
}

func (r *counterpartyRepo) CheckExistsByINN(ctx context.Context, inn string) (bool, error) {
	var count int64
	err := r.db.WithContext(ctx).Model(&domain.Counterparty{}).
		Where("inn = ?", strings.TrimSpace(inn)).
		Count(&count).Error
	return count > 0, err
}

func (r *counterpartyRepo) GetByID(ctx context.Context, id int64) (domain.Counterparty, error) {
	var result domain.Counterparty
	err := r.db.WithContext(ctx).First(&result, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Counterparty{}, errors.New("Компания не найдена")
		}
		return domain.Counterparty{}, err
	}

	result.PhonesList = result.GetPhones()
	result.AccountsList = result.GetAccounts()
	return result, nil
}

func (r *counterpartyRepo) Update(ctx context.Context, id int64, c domain.Counterparty) (domain.Counterparty, error) {
	var existing domain.Counterparty
	if err := r.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.Counterparty{}, errors.New("Компания не найдена")
		}
		return domain.Counterparty{}, err
	}

	clientType := c.ClientType
	if clientType == "" {
		clientType = domain.ClientTypeLegalEntity
	}

	updates := map[string]interface{}{
		"name":        c.Name,
		"llc":         c.LLC,
		"inn":         c.INN,
		"email":       c.Email,
		"client_type": clientType,
		"phones":      c.Phones,
		"accounts":    c.Accounts,
	}

	if err := r.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
		return domain.Counterparty{}, err
	}

	_ = r.db.WithContext(ctx).First(&existing, id)
	existing.PhonesList = existing.GetPhones()
	existing.AccountsList = existing.GetAccounts()
	return existing, nil
}

func (r *counterpartyRepo) SoftDelete(ctx context.Context, id int64) error {
	result := r.db.WithContext(ctx).Delete(&domain.Counterparty{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("Компания не найдена")
	}
	return nil
}
