package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"

	"gorm.io/gorm"
)

func NewAdditionalAgreementRepository(db *gorm.DB) ports.AdditionalAgreementRepository {
	return &additionalAgreementRepo{db: db}
}

type additionalAgreementRepo struct{ db *gorm.DB }

func (r *additionalAgreementRepo) Create(ctx context.Context, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error) {
	ag.Normalize()

	if ag.RemainingAmount == 0 && ag.ForeignAmount != nil {
		ag.RemainingAmount = *ag.ForeignAmount
	}

	var contract domain.Contract
	if err := r.db.WithContext(ctx).First(&contract, ag.ContractID).Error; err != nil {
		return domain.AdditionalAgreement{}, fmt.Errorf("контракт не найден: %w", err)
	}

	if isExpired, exp := contract.IsExpired(time.Now()); isExpired {
		return domain.AdditionalAgreement{}, fmt.Errorf("срок действия контракта истёк (%s). Добавление доп. соглашений запрещено", exp.Format("02.01.2006"))
	}

	ag.Status = "active"

	if err := r.db.WithContext(ctx).Create(&ag).Error; err != nil {
		return domain.AdditionalAgreement{}, err
	}

	ag.Normalize()
	propagateAADatesToContractGorm(ctx, r.db, ag.ContractID, ag.DeliveryDate, ag.ExtendDateTo, ag.ReturnDays)
	_ = syncContractBalances(ctx, r.db, ag.ContractID)
	return ag, nil
}

func (r *additionalAgreementRepo) GetByContractID(ctx context.Context, contractID int64) ([]domain.AdditionalAgreement, error) {
	var list []domain.AdditionalAgreement
	if err := r.db.WithContext(ctx).
		Where("contract_id = ?", contractID).
		Order("created_at ASC").
		Find(&list).Error; err != nil {
		return nil, err
	}
	for i := range list {
		list[i].Normalize()
	}
	enrichAdditionalAgreements(ctx, r.db, list)
	return list, nil
}

func (r *additionalAgreementRepo) GetByID(ctx context.Context, id int64) (domain.AdditionalAgreement, error) {
	var ag domain.AdditionalAgreement
	if err := r.db.WithContext(ctx).First(&ag, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.AdditionalAgreement{}, errors.New("Дополнительное соглашение не найдено")
		}
		return ag, err
	}
	ag.Normalize()
	enrichAdditionalAgreement(ctx, r.db, &ag)
	return ag, nil
}

func (r *additionalAgreementRepo) SoftDelete(ctx context.Context, id int64) error {
	var existing domain.AdditionalAgreement
	if err := r.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("Дополнительное соглашение не найдено")
		}
		return err
	}

	userLogin := domain.GetLoginFromCtx(ctx)
	if userLogin != "" {
		_ = r.db.WithContext(ctx).Model(&domain.AdditionalAgreement{}).Where("id = ?", id).Update("deleted_by", userLogin).Error
	}

	res := r.db.WithContext(ctx).Delete(&domain.AdditionalAgreement{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("Дополнительное соглашение не найдено")
	}
	_ = syncContractBalances(ctx, r.db, existing.ContractID)
	return nil
}

func (r *additionalAgreementRepo) Update(ctx context.Context, id int64, ag domain.AdditionalAgreement) (domain.AdditionalAgreement, error) {
	var existing domain.AdditionalAgreement
	if err := r.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return domain.AdditionalAgreement{}, errors.New("Дополнительное соглашение не найдено")
		}
		return domain.AdditionalAgreement{}, err
	}

	ag.Normalize()

	updates := map[string]interface{}{
		"agreement_number": ag.AgreementNumber,
		"agreement_date":   ag.AgreementDate,
		"delivery_date":    ag.DeliveryDate,
		"return_days":      ag.ReturnDays,
		"subject":          ag.Subject,
		"extend_date_to":   ag.ExtendDateTo,
		"foreign_amount":   ag.ForeignAmount,
		"currency":         ag.ForeignCurrency,
		"document_path":    ag.DocumentPath,
	}

	userLogin := domain.GetLoginFromCtx(ctx)
	if userLogin == "" {
		userLogin = ag.UpdatedBy
	}
	if userLogin != "" {
		updates["updated_by"] = userLogin
	}

	if err := r.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
		return domain.AdditionalAgreement{}, err
	}

	if err := r.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		return domain.AdditionalAgreement{}, err
	}
	existing.Normalize()
	propagateAADatesToContractGorm(ctx, r.db, existing.ContractID, existing.DeliveryDate, existing.ExtendDateTo, existing.ReturnDays)
	_ = syncContractBalances(ctx, r.db, existing.ContractID)
	enrichAdditionalAgreement(ctx, r.db, &existing)
	return existing, nil
}

func (r *additionalAgreementRepo) RestoreAdditionalAgreement(ctx context.Context, id int64) error {
	var existing domain.AdditionalAgreement
	if err := r.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("Дополнительное соглашение не найдено")
		}
		return err
	}

	if existing.Status != domain.ContractStatusArchived {
		return errors.New("Дополнительное соглашение не находится в архиве")
	}

	err := r.db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
		"status":      "active",
		"archived_at": nil,
	}).Error
	if err != nil {
		return err
	}
	_ = syncContractBalances(ctx, r.db, existing.ContractID)
	return nil
}
