package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"

	"gorm.io/gorm"
)

type trashRepo struct {
	db *gorm.DB
}

func NewTrashRepository(db *gorm.DB) ports.TrashRepository {
	return &trashRepo{db: db}
}

func (r *trashRepo) GetTrashItems(ctx context.Context, filter dto.TrashFilter) ([]dto.TrashItem, int, error) {
	includeContract := filter.EntityType == "" || filter.EntityType == "contract"
	includeAA := filter.EntityType == "" || filter.EntityType == "additional_agreement"
	includeInvoice := filter.EntityType == "" || filter.EntityType == "invoice"
	includeGTD := filter.EntityType == "" || filter.EntityType == "gtd"

	var subQueries []*gorm.DB

	if includeContract {
		q := r.db.WithContext(ctx).Table("contracts c").
			Select(`
				c.id,
				'contract' AS entity_type,
				'Контракт' AS entity_name,
				c.contract_number AS number,
				c.contract_date AS document_date,
				c.total_amount AS amount,
				c.contract_currency AS currency,
				COALESCE(c.document_path, '') AS document_path,
				c.client_id,
				COALESCE(cp.name, '') AS client_name,
				c.id AS contract_id,
				c.contract_number AS contract_number,
				COALESCE(c.created_by, '') AS created_by,
				c.deleted_at,
				COALESCE(c.branch_id, cp.branch_id, 0) AS branch_id
			`).
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("c.deleted_at IS NOT NULL")
		subQueries = append(subQueries, q)
	}

	if includeAA {
		q := r.db.WithContext(ctx).Table("additional_agreements aa").
			Select(`
				aa.id,
				'additional_agreement' AS entity_type,
				CASE 
					WHEN aa.doc_type = 'specification' THEN 'Спецификация'
					WHEN aa.doc_type = 'appendix' THEN 'Приложение'
					ELSE 'Доп. соглашение'
				END AS entity_name,
				COALESCE(aa.agreement_number, '') AS number,
				aa.agreement_date AS document_date,
				COALESCE(aa.foreign_amount, 0) AS amount,
				COALESCE(aa.currency, '') AS currency,
				COALESCE(aa.document_path, '') AS document_path,
				c.client_id,
				COALESCE(cp.name, '') AS client_name,
				aa.contract_id,
				COALESCE(c.contract_number, '') AS contract_number,
				COALESCE(aa.created_by, '') AS created_by,
				aa.deleted_at,
				COALESCE(c.branch_id, cp.branch_id, 0) AS branch_id
			`).
			Joins("JOIN contracts c ON c.id = aa.contract_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("aa.deleted_at IS NOT NULL")
		subQueries = append(subQueries, q)
	}

	if includeInvoice {
		q := r.db.WithContext(ctx).Table("invoices i").
			Select(`
				i.id,
				'invoice' AS entity_type,
				'Инвойс' AS entity_name,
				COALESCE(i.invoice_number, '') AS number,
				i.invoice_date AS document_date,
				i.amount AS amount,
				i.currency AS currency,
				COALESCE(i.document_path, '') AS document_path,
				c.client_id,
				COALESCE(cp.name, '') AS client_name,
				i.contract_id,
				COALESCE(c.contract_number, '') AS contract_number,
				COALESCE(i.created_by, '') AS created_by,
				i.deleted_at,
				COALESCE(c.branch_id, cp.branch_id, 0) AS branch_id
			`).
			Joins("JOIN contracts c ON c.id = i.contract_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("i.deleted_at IS NOT NULL")
		subQueries = append(subQueries, q)
	}

	if includeGTD {
		q := r.db.WithContext(ctx).Table("gtd g").
			Select(`
				g.id,
				'gtd' AS entity_type,
				CASE 
					WHEN g.document_type = 'act' THEN 'Акт выполненных работ'
					ELSE 'ГТД'
				END AS entity_name,
				COALESCE(g.gtd_number, '') AS number,
				g.gtd_date AS document_date,
				g.gtd_amount AS amount,
				COALESCE(g.gtd_currency, '') AS currency,
				COALESCE(g.document_path, '') AS document_path,
				c.client_id,
				COALESCE(cp.name, '') AS client_name,
				g.contract_id,
				COALESCE(c.contract_number, '') AS contract_number,
				COALESCE(g.created_by, '') AS created_by,
				g.deleted_at,
				COALESCE(c.branch_id, cp.branch_id, 0) AS branch_id
			`).
			Joins("JOIN contracts c ON c.id = g.contract_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("g.deleted_at IS NOT NULL")
		subQueries = append(subQueries, q)
	}

	if len(subQueries) == 0 {
		return []dto.TrashItem{}, 0, nil
	}

	var combinedQuery *gorm.DB
	if len(subQueries) == 1 {
		combinedQuery = subQueries[0]
	} else {
		unionClauses := make([]string, len(subQueries))
		for i := range subQueries {
			unionClauses[i] = "(?)"
		}
		unionSQL := strings.Join(unionClauses, " UNION ALL ")
		args := make([]interface{}, len(subQueries))
		for i, sq := range subQueries {
			args[i] = sq
		}
		combinedQuery = r.db.WithContext(ctx).Table(fmt.Sprintf("(%s) AS u", unionSQL), args...)
	}

	if filter.BranchID > 0 {
		combinedQuery = combinedQuery.Where("u.branch_id = ?", filter.BranchID)
	}

	if strings.TrimSpace(filter.Search) != "" {
		s := "%" + strings.TrimSpace(filter.Search) + "%"
		combinedQuery = combinedQuery.Where("u.number ILIKE ? OR u.client_name ILIKE ? OR u.contract_number ILIKE ?", s, s, s)
	}

	var total int64
	if err := combinedQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	page := filter.Page
	if page < 1 {
		page = 1
	}
	pageSize := filter.PageSize
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize

	var items []dto.TrashItem
	if err := combinedQuery.Order("u.deleted_at DESC").Limit(pageSize).Offset(offset).Scan(&items).Error; err != nil {
		return nil, 0, err
	}

	if items == nil {
		items = []dto.TrashItem{}
	}

	return items, int(total), nil
}

func (r *trashRepo) GetTrashItemByID(ctx context.Context, entityType string, id int64) (*dto.TrashItem, error) {
	var item dto.TrashItem
	var err error

	switch entityType {
	case "contract":
		err = r.db.WithContext(ctx).Table("contracts c").
			Select(`
				c.id, 'contract' AS entity_type, 'Контракт' AS entity_name,
				c.contract_number AS number, c.contract_date AS document_date,
				c.total_amount AS amount, c.contract_currency AS currency,
				COALESCE(c.document_path, '') AS document_path, c.client_id,
				COALESCE(cp.name, '') AS client_name, c.id AS contract_id,
				c.contract_number AS contract_number, COALESCE(c.created_by, '') AS created_by,
				c.deleted_at
			`).
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("c.id = ? AND c.deleted_at IS NOT NULL", id).
			Take(&item).Error

	case "additional_agreement":
		err = r.db.WithContext(ctx).Table("additional_agreements aa").
			Select(`
				aa.id, 'additional_agreement' AS entity_type,
				CASE 
					WHEN aa.doc_type = 'specification' THEN 'Спецификация'
					WHEN aa.doc_type = 'appendix' THEN 'Приложение'
					ELSE 'Доп. соглашение'
				END AS entity_name,
				COALESCE(aa.agreement_number, '') AS number, aa.agreement_date AS document_date,
				COALESCE(aa.foreign_amount, 0) AS amount, COALESCE(aa.currency, '') AS currency,
				COALESCE(aa.document_path, '') AS document_path, c.client_id,
				COALESCE(cp.name, '') AS client_name, aa.contract_id,
				COALESCE(c.contract_number, '') AS contract_number, COALESCE(aa.created_by, '') AS created_by,
				aa.deleted_at
			`).
			Joins("JOIN contracts c ON c.id = aa.contract_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("aa.id = ? AND aa.deleted_at IS NOT NULL", id).
			Take(&item).Error

	case "invoice":
		err = r.db.WithContext(ctx).Table("invoices i").
			Select(`
				i.id, 'invoice' AS entity_type, 'Инвойс' AS entity_name,
				COALESCE(i.invoice_number, '') AS number, i.invoice_date AS document_date,
				i.amount AS amount, i.currency AS currency,
				COALESCE(i.document_path, '') AS document_path, c.client_id,
				COALESCE(cp.name, '') AS client_name, i.contract_id,
				COALESCE(c.contract_number, '') AS contract_number, COALESCE(i.created_by, '') AS created_by,
				i.deleted_at
			`).
			Joins("JOIN contracts c ON c.id = i.contract_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("i.id = ? AND i.deleted_at IS NOT NULL", id).
			Take(&item).Error

	case "gtd":
		err = r.db.WithContext(ctx).Table("gtd g").
			Select(`
				g.id, 'gtd' AS entity_type,
				CASE 
					WHEN g.document_type = 'act' THEN 'Акт выполненных работ'
					ELSE 'ГТД'
				END AS entity_name,
				COALESCE(g.gtd_number, '') AS number, g.gtd_date AS document_date,
				g.gtd_amount AS amount, COALESCE(g.gtd_currency, '') AS currency,
				COALESCE(g.document_path, '') AS document_path, c.client_id,
				COALESCE(cp.name, '') AS client_name, g.contract_id,
				COALESCE(c.contract_number, '') AS contract_number, COALESCE(g.created_by, '') AS created_by,
				g.deleted_at
			`).
			Joins("JOIN contracts c ON c.id = g.contract_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("g.id = ? AND g.deleted_at IS NOT NULL", id).
			Take(&item).Error

	default:
		return nil, fmt.Errorf("неизвестный тип сущности: %s", entityType)
	}

	if err != nil {
		return nil, fmt.Errorf("документ не найден в корзине: %w", err)
	}

	return &item, nil
}

func (r *trashRepo) RestoreItem(ctx context.Context, entityType string, id int64) error {
	now := time.Now()
	restoreUpdates := map[string]interface{}{
		"deleted_at": nil,
		"updated_at": &now,
	}

	switch entityType {
	case "contract":
		res := r.db.WithContext(ctx).Unscoped().Model(&domain.Contract{}).
			Where("id = ? AND deleted_at IS NOT NULL", id).
			Updates(restoreUpdates)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return fmt.Errorf("контракт не найден в корзине или уже восстановлен")
		}
		return nil

	case "additional_agreement":
		var aa domain.AdditionalAgreement
		err := r.db.WithContext(ctx).Unscoped().
			Select("contract_id").
			Where("id = ? AND deleted_at IS NOT NULL", id).
			First(&aa).Error
		if err != nil {
			return fmt.Errorf("доп. соглашение не найдено в корзине: %w", err)
		}

		res := r.db.WithContext(ctx).Unscoped().Model(&domain.AdditionalAgreement{}).
			Where("id = ? AND deleted_at IS NOT NULL", id).
			Updates(restoreUpdates)
		if res.Error != nil {
			return res.Error
		}

		syncAdditionalAgreementRemainingGorm(ctx, r.db, id)
		return nil

	case "invoice":
		var inv domain.Invoice
		err := r.db.WithContext(ctx).Unscoped().
			Select("contract_id, additional_agreement_id").
			Where("id = ? AND deleted_at IS NOT NULL", id).
			First(&inv).Error
		if err != nil {
			return fmt.Errorf("инвойс не найден в корзине: %w", err)
		}

		res := r.db.WithContext(ctx).Unscoped().Model(&domain.Invoice{}).
			Where("id = ? AND deleted_at IS NOT NULL", id).
			Updates(restoreUpdates)
		if res.Error != nil {
			return res.Error
		}

		if inv.AdditionalAgreementID != nil {
			syncAdditionalAgreementRemainingGorm(ctx, r.db, *inv.AdditionalAgreementID)
			tryArchiveAdditionalAgreementGorm(ctx, r.db, *inv.AdditionalAgreementID)
		} else if inv.ContractID > 0 {
			syncContractRemainingGorm(ctx, r.db, inv.ContractID)
			tryArchiveContractGorm(ctx, r.db, inv.ContractID)
		}
		return nil

	case "gtd":
		var g domain.GTD
		err := r.db.WithContext(ctx).Unscoped().
			Select("contract_id, additional_agreement_id").
			Where("id = ? AND deleted_at IS NOT NULL", id).
			First(&g).Error
		if err != nil {
			return fmt.Errorf("ГТД не найдена в корзине: %w", err)
		}

		res := r.db.WithContext(ctx).Unscoped().Model(&domain.GTD{}).
			Where("id = ? AND deleted_at IS NOT NULL", id).
			Updates(restoreUpdates)
		if res.Error != nil {
			return res.Error
		}

		if g.AdditionalAgreementID != nil {
			tryArchiveAdditionalAgreementGorm(ctx, r.db, *g.AdditionalAgreementID)
		} else if g.ContractID > 0 {
			tryArchiveContractGorm(ctx, r.db, g.ContractID)
		}
		return nil

	default:
		return fmt.Errorf("неизвестный тип сущности: %s", entityType)
	}
}

