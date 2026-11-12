package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/errs"
	"CurrencyControl/internal/service/ports"

	"gorm.io/gorm"
)

type documentRepo struct {
	db *gorm.DB
}

func NewDocumentRepository(db *gorm.DB) ports.DocumentRepository {
	return &documentRepo{db: db}
}

func (r *documentRepo) GetDocumentFileInfo(ctx context.Context, entityType string, id int64) (*domain.DocumentFileInfo, error) {
	entityType = strings.TrimSpace(entityType)

	var docInfo domain.DocumentFileInfo
	docInfo.EntityType = entityType
	docInfo.EntityID = id

	type queryResult struct {
		DocumentPath   *string `gorm:"column:document_path"`
		Number         *string `gorm:"column:doc_number"`
		ApprovalStatus *string `gorm:"column:approval_status"`
	}
	var res queryResult
	var tx *gorm.DB
	switch entityType {
	case domain.EntityTypeContract:
		tx = r.db.WithContext(ctx).Table("contracts").
			Select("document_path, contract_number AS doc_number, 'approved' AS approval_status").
			Where("id = ? AND deleted_at IS NULL", id).
			Take(&res)
	case domain.EntityTypeInvoice:
		tx = r.db.WithContext(ctx).Table("invoices").
			Select("document_path, invoice_number AS doc_number, 'approved' AS approval_status").
			Where("id = ? AND deleted_at IS NULL", id).
			Take(&res)
	case domain.EntityTypeGTD:
		tx = r.db.WithContext(ctx).Table("gtd").
			Select("document_path, gtd_number AS doc_number, approval_status").
			Where("id = ? AND deleted_at IS NULL", id).
			Take(&res)
	case domain.EntityTypeAct:
		tx = r.db.WithContext(ctx).Table("acts").
			Select("document_path, act_number AS doc_number, 'approved' AS approval_status").
			Where("id = ? AND deleted_at IS NULL", id).
			Take(&res)
	case domain.EntityTypeAdditionalAgreement:
		tx = r.db.WithContext(ctx).Table("additional_agreements").
			Select("document_path, agreement_number AS doc_number, 'approved' AS approval_status").
			Where("id = ? AND deleted_at IS NULL", id).
			Take(&res)
	case domain.EntityTypePaymentOrder:
		tx = r.db.WithContext(ctx).Table("payment_orders").
			Select("document_path, id::text AS doc_number, approval_status").
			Where("id = ? AND deleted_at IS NULL", id).
			Take(&res)
	case "gtd_extension":
		tx = r.db.WithContext(ctx).Table("gtd_extension_requests").
			Select("document_path, '' AS doc_number, '' AS approval_status").
			Where("id = ? AND deleted_at IS NULL", id).
			Take(&res)
	default:
		return nil, fmt.Errorf("неизвестный тип сущности: %s (поддерживаются: contract, invoice, gtd, act, additional_agreement, payment_order, gtd_extension)", entityType)
	}

	if tx.Error != nil {
		if errors.Is(tx.Error, gorm.ErrRecordNotFound) {
			return nil, errs.ErrNotFound
		}
		return nil, tx.Error
	}
	if res.DocumentPath != nil {
		docInfo.DocumentPath = strings.TrimSpace(*res.DocumentPath)
	}
	if res.Number != nil {
		docInfo.DocumentNumber = strings.TrimSpace(*res.Number)
	}
	if res.ApprovalStatus != nil {
		docInfo.ApprovalStatus = strings.TrimSpace(*res.ApprovalStatus)
	}
	return &docInfo, nil
}
