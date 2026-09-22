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
	if ag.DocType == "" {
		ag.DocType = domain.DocTypeAdditionalAgreement
	}
	ag.Normalize()

	if ag.RemainingAmount == 0 && ag.ForeignAmount != nil {
		ag.RemainingAmount = *ag.ForeignAmount
	}

	var contract domain.Contract
	if err := r.db.WithContext(ctx).First(&contract, ag.ContractID).Error; err != nil {
		return domain.AdditionalAgreement{}, fmt.Errorf("контракт не найден: %w", err)
	}

	if ag.AgreementDate != nil {
		dContract := time.Date(contract.ContractDate.Year(), contract.ContractDate.Month(), contract.ContractDate.Day(), 0, 0, 0, 0, time.UTC)
		dAgreement := time.Date(ag.AgreementDate.Year(), ag.AgreementDate.Month(), ag.AgreementDate.Day(), 0, 0, 0, 0, time.UTC)
		if dAgreement.Before(dContract) {
			return domain.AdditionalAgreement{}, fmt.Errorf("дата доп. соглашения не может быть раньше даты контракта (%s)", contract.ContractDate.Format("02.01.2006"))
		}

		if ag.DeliveryDate != nil {
			dDelivery := time.Date(ag.DeliveryDate.Year(), ag.DeliveryDate.Month(), ag.DeliveryDate.Day(), 0, 0, 0, 0, time.UTC)
			if dDelivery.Before(dAgreement) {
				return domain.AdditionalAgreement{}, fmt.Errorf("срок поставки товара не может быть раньше даты доп. соглашения")
			}
			if ag.ExtendDateTo != nil {
				dEnd := time.Date(ag.ExtendDateTo.Year(), ag.ExtendDateTo.Month(), ag.ExtendDateTo.Day(), 0, 0, 0, 0, time.UTC)
				if dDelivery.After(dEnd) {
					return domain.AdditionalAgreement{}, fmt.Errorf("срок поставки товара не может быть позже даты окончания доп. соглашения")
				}
			}
		}

		if ag.ExtendDateTo != nil {
			dEnd := time.Date(ag.ExtendDateTo.Year(), ag.ExtendDateTo.Month(), ag.ExtendDateTo.Day(), 0, 0, 0, 0, time.UTC)
			if dEnd.Before(dAgreement) {
				return domain.AdditionalAgreement{}, fmt.Errorf("дата окончания доп. соглашения не может быть раньше даты доп. соглашения")
			}
			if ag.DeliveryDate != nil {
				dDelivery := time.Date(ag.DeliveryDate.Year(), ag.DeliveryDate.Month(), ag.DeliveryDate.Day(), 0, 0, 0, 0, time.UTC)
				if dEnd.Before(dDelivery) {
					return domain.AdditionalAgreement{}, fmt.Errorf("дата окончания доп. соглашения не может быть раньше срока поставки товара")
				}
			}
		}
	}

	today := time.Now().UTC().Truncate(24 * time.Hour)
	var expiryDate *time.Time
	if contract.ExtendDateTo != nil && !contract.ExtendDateTo.IsZero() {
		expiryDate = contract.ExtendDateTo
	} else if !contract.DeliveryDate.IsZero() {
		expiryDate = &contract.DeliveryDate
	}
	if expiryDate != nil && !expiryDate.IsZero() {
		exp := time.Date(expiryDate.Year(), expiryDate.Month(), expiryDate.Day(), 0, 0, 0, 0, time.UTC)
		if today.After(exp) {
			return domain.AdditionalAgreement{}, fmt.Errorf("срок действия контракта истёк (%s). Добавление доп. соглашений запрещено", exp.Format("02.01.2006"))
		}
	}

	if ag.ApprovalStatus == "" {
		ag.ApprovalStatus = domain.ApprovalStatusPendingCurrencyControl
	}

	if err := r.db.WithContext(ctx).Create(&ag).Error; err != nil {
		return domain.AdditionalAgreement{}, err
	}

	ag.Normalize()
	propagateAADatesToContractGorm(ctx, r.db, ag.ContractID, ag.DeliveryDate, ag.ExtendDateTo, ag.ReturnDays)
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
	return ag, nil
}

func (r *additionalAgreementRepo) SoftDelete(ctx context.Context, id int64) error {
	res := r.db.WithContext(ctx).Delete(&domain.AdditionalAgreement{}, id)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return errors.New("Дополнительное соглашение не найдено")
	}
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

	if ag.DocType == "" {
		ag.DocType = domain.DocTypeAdditionalAgreement
	}
	ag.Normalize()

	var contract domain.Contract
	if err := r.db.WithContext(ctx).First(&contract, existing.ContractID).Error; err == nil {
		checkAgreementDate := existing.AgreementDate
		if ag.AgreementDate != nil {
			checkAgreementDate = ag.AgreementDate
		}
		checkDeliveryDate := existing.DeliveryDate
		if ag.DeliveryDate != nil {
			checkDeliveryDate = ag.DeliveryDate
		}
		checkEndDate := existing.ExtendDateTo
		if ag.ExtendDateTo != nil {
			checkEndDate = ag.ExtendDateTo
		}

		if checkAgreementDate != nil {
			dContract := time.Date(contract.ContractDate.Year(), contract.ContractDate.Month(), contract.ContractDate.Day(), 0, 0, 0, 0, time.UTC)
			dAgreement := time.Date(checkAgreementDate.Year(), checkAgreementDate.Month(), checkAgreementDate.Day(), 0, 0, 0, 0, time.UTC)
			if dAgreement.Before(dContract) {
				return domain.AdditionalAgreement{}, fmt.Errorf("дата доп. соглашения не может быть раньше даты контракта (%s)", contract.ContractDate.Format("02.01.2006"))
			}

			if checkDeliveryDate != nil {
				dDelivery := time.Date(checkDeliveryDate.Year(), checkDeliveryDate.Month(), checkDeliveryDate.Day(), 0, 0, 0, 0, time.UTC)
				if dDelivery.Before(dAgreement) {
					return domain.AdditionalAgreement{}, fmt.Errorf("срок поставки товара не может быть раньше даты доп. соглашения")
				}
				if checkEndDate != nil {
					dEnd := time.Date(checkEndDate.Year(), checkEndDate.Month(), checkEndDate.Day(), 0, 0, 0, 0, time.UTC)
					if dDelivery.After(dEnd) {
						return domain.AdditionalAgreement{}, fmt.Errorf("срок поставки товара не может быть позже даты окончания доп. соглашения")
					}
				}
			}

			if checkEndDate != nil {
				dEnd := time.Date(checkEndDate.Year(), checkEndDate.Month(), checkEndDate.Day(), 0, 0, 0, 0, time.UTC)
				if dEnd.Before(dAgreement) {
					return domain.AdditionalAgreement{}, fmt.Errorf("дата окончания доп. соглашения не может быть раньше даты доп. соглашения")
				}
				if checkDeliveryDate != nil {
					dDelivery := time.Date(checkDeliveryDate.Year(), checkDeliveryDate.Month(), checkDeliveryDate.Day(), 0, 0, 0, 0, time.UTC)
					if dEnd.Before(dDelivery) {
						return domain.AdditionalAgreement{}, fmt.Errorf("дата окончания доп. соглашения не может быть раньше срока поставки товара")
					}
				}
			}
		}
	}

	updates := map[string]interface{}{
		"doc_type":         ag.DocType,
		"agreement_number": ag.AgreementNumber,
		"agreement_date":   ag.AgreementDate,
		"delivery_date":    ag.DeliveryDate,
		"return_days":      ag.ReturnDays,
		"subject":          ag.Subject,
		"extend_date_to":   ag.ExtendDateTo,
		"foreign_amount":   ag.ForeignAmount,
		"currency":         ag.ForeignCurrency,
		"document_path":    ag.DocumentPath,
		"receiver_name":    ag.ReceiverName,
		"receiver_bank":    ag.ReceiverBank,
		"receiver_country": ag.ReceiverCountry,
		"approval_status":  domain.ApprovalStatusPendingCurrencyControl,
		"rejection_reason": "",
	}

	if err := r.db.WithContext(ctx).Model(&existing).Updates(updates).Error; err != nil {
		return domain.AdditionalAgreement{}, err
	}

	if err := r.db.WithContext(ctx).First(&existing, id).Error; err != nil {
		return domain.AdditionalAgreement{}, err
	}
	existing.Normalize()
	syncAdditionalAgreementRemainingGorm(ctx, r.db, existing.ID)
	propagateAADatesToContractGorm(ctx, r.db, existing.ContractID, existing.DeliveryDate, existing.ExtendDateTo, existing.ReturnDays)
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

	return r.db.WithContext(ctx).Model(&existing).Updates(map[string]interface{}{
		"status":      "active",
		"archived_at": nil,
	}).Error
}
