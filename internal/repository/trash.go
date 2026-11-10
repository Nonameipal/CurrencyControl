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
	includeAct := filter.EntityType == "" || filter.EntityType == "act"
	includePO := filter.EntityType == "" || filter.EntityType == "payment_order"

	var subQueries []*gorm.DB

	if includeContract {
		q := r.db.WithContext(ctx).Table("contracts c").
			Select(BuildTrashSelect("c", "contract", "'Контракт'", "c.contract_number", "c.contract_date", "c.total_amount", "c.contract_currency")).
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("c.deleted_at IS NOT NULL")
		subQueries = append(subQueries, q)
	}

	if includeAA {
		entityName := "CASE WHEN aa.doc_type = 'specification' THEN 'Спецификация' WHEN aa.doc_type = 'appendix' THEN 'Приложение' ELSE 'Доп. соглашение' END"
		q := r.db.WithContext(ctx).Table("additional_agreements aa").
			Select(BuildTrashSelect("aa", "additional_agreement", entityName, "aa.agreement_number", "aa.agreement_date", "aa.foreign_amount", "aa.currency")).
			Joins("JOIN contracts c ON c.id = aa.contract_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("aa.deleted_at IS NOT NULL")
		subQueries = append(subQueries, q)
	}

	if includeInvoice {
		q := r.db.WithContext(ctx).Table("invoices i").
			Select(BuildTrashSelect("i", "invoice", "'Инвойс'", "i.invoice_number", "i.invoice_date", "i.amount", "i.currency")).
			Joins("JOIN contracts c ON c.id = i.contract_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("i.deleted_at IS NOT NULL")
		subQueries = append(subQueries, q)
	}

	if includeGTD {
		q := r.db.WithContext(ctx).Table("gtd g").
			Select(BuildTrashSelect("g", "gtd", "'ГТД'", "g.gtd_number", "g.gtd_date", "g.gtd_amount", "g.currency")).
			Joins("JOIN contracts c ON c.id = g.contract_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("g.deleted_at IS NOT NULL")
		subQueries = append(subQueries, q)
	}

	if includeAct {
		q := r.db.WithContext(ctx).Table("acts a").
			Select(BuildTrashSelect("a", "act", "'Акт выполненных работ'", "a.act_number", "a.act_date", "a.act_amount", "a.currency")).
			Joins("JOIN contracts c ON c.id = a.contract_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("a.deleted_at IS NOT NULL")
		subQueries = append(subQueries, q)
	}

	if includePO {
		q := r.db.WithContext(ctx).Table("payment_orders po").
			Select(BuildTrashSelect("po", "payment_order", "'Платежное поручение'", "po.id::text", "po.value_date", "po.amount", "po.currency")).
			Joins("JOIN contracts c ON c.id = po.contract_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("po.deleted_at IS NOT NULL")
		subQueries = append(subQueries, q)
	}

	if len(subQueries) == 0 {
		return []dto.TrashItem{}, 0, nil
	}

	var combinedQuery *gorm.DB
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
	r.enrichTrashItems(ctx, items)

	return items, int(total), nil
}

func (r *trashRepo) GetTrashItemByID(ctx context.Context, entityType string, id int64) (*dto.TrashItem, error) {
	var item dto.TrashItem
	var err error

	switch entityType {
	case "contract":
		err = r.db.WithContext(ctx).Table("contracts c").
			Select(BuildTrashSelect("c", "contract", "'Контракт'", "c.contract_number", "c.contract_date", "c.total_amount", "c.contract_currency")).
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("c.id = ? AND c.deleted_at IS NOT NULL", id).
			Take(&item).Error

	case "additional_agreement":
		entityName := "CASE WHEN aa.doc_type = 'specification' THEN 'Спецификация' WHEN aa.doc_type = 'appendix' THEN 'Приложение' ELSE 'Доп. соглашение' END"
		err = r.db.WithContext(ctx).Table("additional_agreements aa").
			Select(BuildTrashSelect("aa", "additional_agreement", entityName, "aa.agreement_number", "aa.agreement_date", "aa.foreign_amount", "aa.currency")).
			Joins("JOIN contracts c ON c.id = aa.contract_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("aa.id = ? AND aa.deleted_at IS NOT NULL", id).
			Take(&item).Error

	case "invoice":
		err = r.db.WithContext(ctx).Table("invoices i").
			Select(BuildTrashSelect("i", "invoice", "'Инвойс'", "i.invoice_number", "i.invoice_date", "i.amount", "i.currency")).
			Joins("JOIN contracts c ON c.id = i.contract_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("i.id = ? AND i.deleted_at IS NOT NULL", id).
			Take(&item).Error

	case "gtd":
		err = r.db.WithContext(ctx).Table("gtd g").
			Select(BuildTrashSelect("g", "gtd", "'ГТД'", "g.gtd_number", "g.gtd_date", "g.gtd_amount", "g.currency")).
			Joins("JOIN contracts c ON c.id = g.contract_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("g.id = ? AND g.deleted_at IS NOT NULL", id).
			Take(&item).Error

	case "act":
		err = r.db.WithContext(ctx).Table("acts a").
			Select(BuildTrashSelect("a", "act", "'Акт выполненных работ'", "a.act_number", "a.act_date", "a.act_amount", "a.currency")).
			Joins("JOIN contracts c ON c.id = a.contract_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("a.id = ? AND a.deleted_at IS NOT NULL", id).
			Take(&item).Error

	case "payment_order":
		err = r.db.WithContext(ctx).Table("payment_orders po").
			Select(BuildTrashSelect("po", "payment_order", "'Платежное поручение'", "po.id::text", "po.value_date", "po.amount", "po.currency")).
			Joins("JOIN contracts c ON c.id = po.contract_id").
			Joins("LEFT JOIN counterparties cp ON cp.id = c.client_id").
			Where("po.id = ? AND po.deleted_at IS NOT NULL", id).
			Take(&item).Error

	default:
		return nil, fmt.Errorf("неизвестный тип сущности: %s", entityType)
	}
	if err != nil {
		return nil, fmt.Errorf("документ не найден в корзине: %w", err)
	}
	single := []dto.TrashItem{item}
	r.enrichTrashItems(ctx, single)
	return &single[0], nil
}

func (r *trashRepo) enrichTrashItems(ctx context.Context, items []dto.TrashItem) {
	logins := make([]string, 0, len(items)*2)
	for _, it := range items {
		logins = append(logins, it.CreatedBy, it.DeletedBy)
	}
	userMap := fetchUserBriefs(ctx, r.db, logins)
	for i := range items {
		if items[i].CreatedBy != "" {
			b := userMap[items[i].CreatedBy]
			items[i].Creator = &b
		}
		if items[i].DeletedBy != "" {
			b := userMap[items[i].DeletedBy]
			items[i].Deleter = &b
		}
	}
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
		_ = syncContractBalances(ctx, r.db, aa.ContractID)
		return nil
	case "invoice":
		var inv domain.Invoice
		err := r.db.WithContext(ctx).Unscoped().
			Select("contract_id").
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
		if inv.ContractID > 0 {
			_ = syncContractBalances(ctx, r.db, inv.ContractID)
			tryArchiveContractGorm(ctx, r.db, inv.ContractID)
		}
		return nil
	case "gtd":
		var g domain.GTD
		err := r.db.WithContext(ctx).Unscoped().
			Select("contract_id").
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
		if g.ContractID > 0 {
			tryArchiveContractGorm(ctx, r.db, g.ContractID)
		}
		return nil
	case "act":
		var a domain.Act
		err := r.db.WithContext(ctx).Unscoped().
			Select("contract_id").
			Where("id = ? AND deleted_at IS NOT NULL", id).
			First(&a).Error
		if err != nil {
			return fmt.Errorf("акт не найден в корзине: %w", err)
		}
		res := r.db.WithContext(ctx).Unscoped().Model(&domain.Act{}).
			Where("id = ? AND deleted_at IS NOT NULL", id).
			Updates(restoreUpdates)
		if res.Error != nil {
			return res.Error
		}
		if a.ContractID > 0 {
			tryArchiveContractGorm(ctx, r.db, a.ContractID)
		}
		return nil
	case "payment_order":
		var po domain.PaymentOrder
		err := r.db.WithContext(ctx).Unscoped().
			Select("contract_id").
			Where("id = ? AND deleted_at IS NOT NULL", id).
			First(&po).Error
		if err != nil {
			return fmt.Errorf("платежное поручение не найдено в корзине: %w", err)
		}
		res := r.db.WithContext(ctx).Unscoped().Model(&domain.PaymentOrder{}).
			Where("id = ? AND deleted_at IS NOT NULL", id).
			Updates(restoreUpdates)
		if res.Error != nil {
			return res.Error
		}
		if po.ContractID > 0 {
			tryArchiveContractGorm(ctx, r.db, po.ContractID)
		}
		return nil
	default:
		return fmt.Errorf("неизвестный тип сущности: %s", entityType)
	}
}
