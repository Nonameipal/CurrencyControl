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

func normalizeEntityType(raw string) string {
	raw = strings.ToLower(strings.TrimSpace(raw))
	raw = strings.ReplaceAll(raw, "-", "_")
	switch raw {
	case "contract", "contracts":
		return "contract"
	case "invoice", "invoices":
		return "invoice"
	case "gtd", "gtds":
		return "gtd"
	case "additional_agreement", "additional_agreements", "additionalagreement", "additionalagreements":
		return "additional_agreement"
	case "payment_order", "payment_orders", "paymentorder", "paymentorders":
		return "payment_order"
	case "gtd_extension", "gtd_extensions", "gtdextension", "gtdextensions":
		return "gtd_extension"
	default:
		return raw
	}
}

func (r *documentRepo) GetDocumentFileInfo(ctx context.Context, entityType string, id int64) (*domain.DocumentFileInfo, error) {
	normType := normalizeEntityType(entityType)

	var docInfo domain.DocumentFileInfo
	docInfo.EntityType = normType
	docInfo.EntityID = id

	type queryResult struct {
		DocumentPath *string `gorm:"column:document_path"`
		Number       *string `gorm:"column:doc_number"`
	}
	var res queryResult

	var tx *gorm.DB
	switch normType {
	case "contract":
		tx = r.db.WithContext(ctx).Table("contracts").
			Select("document_path, contract_number AS doc_number").
			Where("id = ? AND deleted_at IS NULL", id).
			Take(&res)

	case "invoice":
		tx = r.db.WithContext(ctx).Table("invoices").
			Select("document_path, invoice_number AS doc_number").
			Where("id = ? AND deleted_at IS NULL", id).
			Take(&res)

	case "gtd":
		tx = r.db.WithContext(ctx).Table("gtd").
			Select("document_path, gtd_number AS doc_number").
			Where("id = ? AND deleted_at IS NULL", id).
			Take(&res)

	case "additional_agreement":
		tx = r.db.WithContext(ctx).Table("additional_agreements").
			Select("document_path, agreement_number AS doc_number").
			Where("id = ? AND deleted_at IS NULL", id).
			Take(&res)

	case "payment_order":
		tx = r.db.WithContext(ctx).Table("payment_orders").
			Select("document_path, payment_order_number AS doc_number").
			Where("id = ? AND deleted_at IS NULL", id).
			Take(&res)

	case "gtd_extension":
		tx = r.db.WithContext(ctx).Table("gtd_extension_requests").
			Select("document_path, '' AS doc_number").
			Where("id = ? AND deleted_at IS NULL", id).
			Take(&res)

	default:
		return nil, fmt.Errorf("неизвестный тип сущности: %s (поддерживаются: contract, invoice, gtd, additional_agreement, payment_order, gtd_extension)", entityType)
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

	return &docInfo, nil
}
