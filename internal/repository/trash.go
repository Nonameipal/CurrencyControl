package repository

import (
	"context"
	"fmt"
	"strings"

	"CurrencyControl/internal/delivery/dto"
	"CurrencyControl/internal/service/ports"

	"github.com/jackc/pgx/v5/pgxpool"
)

type trashRepo struct {
	db *pgxpool.Pool
}

func NewTrashRepository(db *pgxpool.Pool) ports.TrashRepository {
	return &trashRepo{db: db}
}

func (r *trashRepo) GetTrashItems(ctx context.Context, filter dto.TrashFilter) ([]dto.TrashItem, int, error) {
	var queries []string

	includeContract := filter.EntityType == "" || filter.EntityType == "contract"
	includeAA := filter.EntityType == "" || filter.EntityType == "additional_agreement"
	includeInvoice := filter.EntityType == "" || filter.EntityType == "invoice"
	includeGTD := filter.EntityType == "" || filter.EntityType == "gtd"

	if includeContract {
		q := `
			SELECT 
				c.id,
				'contract'::varchar AS entity_type,
				'Контракт'::varchar AS entity_name,
				c.contract_number::varchar AS number,
				c.contract_date::timestamp AS document_date,
				c.total_amount::numeric::float8 AS amount,
				c.contract_currency::varchar AS currency,
				COALESCE(c.document_path, '')::varchar AS document_path,
				c.client_id,
				COALESCE(cp.name, '')::varchar AS client_name,
				c.id AS contract_id,
				c.contract_number::varchar AS contract_number,
				COALESCE(c.created_by, '')::varchar AS created_by,
				c.deleted_at,
				COALESCE(c.branch_id, cp.branch_id, 0) AS branch_id
			FROM contracts c
			LEFT JOIN counterparties cp ON cp.id = c.client_id
			WHERE c.deleted_at IS NOT NULL`
		queries = append(queries, q)
	}

	if includeAA {
		q := `
			SELECT 
				aa.id,
				'additional_agreement'::varchar AS entity_type,
				CASE 
					WHEN aa.doc_type = 'specification' THEN 'Спецификация'
					WHEN aa.doc_type = 'appendix' THEN 'Приложение'
					ELSE 'Доп. соглашение'
				END::varchar AS entity_name,
				COALESCE(aa.agreement_number, '')::varchar AS number,
				aa.agreement_date::timestamp AS document_date,
				COALESCE(aa.foreign_amount, 0)::numeric::float8 AS amount,
				COALESCE(aa.currency, '')::varchar AS currency,
				COALESCE(aa.document_path, '')::varchar AS document_path,
				c.client_id,
				COALESCE(cp.name, '')::varchar AS client_name,
				aa.contract_id,
				COALESCE(c.contract_number, '')::varchar AS contract_number,
				COALESCE(aa.created_by, '')::varchar AS created_by,
				aa.deleted_at,
				COALESCE(c.branch_id, cp.branch_id, 0) AS branch_id
			FROM additional_agreements aa
			JOIN contracts c ON c.id = aa.contract_id
			LEFT JOIN counterparties cp ON cp.id = c.client_id
			WHERE aa.deleted_at IS NOT NULL`
		queries = append(queries, q)
	}

	if includeInvoice {
		q := `
			SELECT 
				i.id,
				'invoice'::varchar AS entity_type,
				'Инвойс'::varchar AS entity_name,
				COALESCE(i.invoice_number, '')::varchar AS number,
				i.invoice_date::timestamp AS document_date,
				i.amount::numeric::float8 AS amount,
				i.currency::varchar AS currency,
				COALESCE(i.document_path, '')::varchar AS document_path,
				c.client_id,
				COALESCE(cp.name, '')::varchar AS client_name,
				i.contract_id,
				COALESCE(c.contract_number, '')::varchar AS contract_number,
				COALESCE(i.created_by, '')::varchar AS created_by,
				i.deleted_at,
				COALESCE(c.branch_id, cp.branch_id, 0) AS branch_id
			FROM invoices i
			JOIN contracts c ON c.id = i.contract_id
			LEFT JOIN counterparties cp ON cp.id = c.client_id
			WHERE i.deleted_at IS NOT NULL`
		queries = append(queries, q)
	}

	if includeGTD {
		q := `
			SELECT 
				g.id,
				'gtd'::varchar AS entity_type,
				CASE 
					WHEN g.document_type = 'act' THEN 'Акт выполненных работ'
					ELSE 'ГТД'
				END::varchar AS entity_name,
				COALESCE(g.gtd_number, '')::varchar AS number,
				g.gtd_date::timestamp AS document_date,
				g.gtd_amount::numeric::float8 AS amount,
				COALESCE(g.gtd_currency, '')::varchar AS currency,
				COALESCE(g.document_path, '')::varchar AS document_path,
				c.client_id,
				COALESCE(cp.name, '')::varchar AS client_name,
				g.contract_id,
				COALESCE(c.contract_number, '')::varchar AS contract_number,
				COALESCE(g.created_by, '')::varchar AS created_by,
				g.deleted_at,
				COALESCE(c.branch_id, cp.branch_id, 0) AS branch_id
			FROM gtd g
			JOIN contracts c ON c.id = g.contract_id
			LEFT JOIN counterparties cp ON cp.id = c.client_id
			WHERE g.deleted_at IS NOT NULL`
		queries = append(queries, q)
	}

	if len(queries) == 0 {
		return []dto.TrashItem{}, 0, nil
	}

	unionQuery := strings.Join(queries, " UNION ALL ")

	var whereClauses []string
	var args []interface{}
	argIdx := 1

	if filter.BranchID > 0 {
		whereClauses = append(whereClauses, fmt.Sprintf("u.branch_id = $%d", argIdx))
		args = append(args, filter.BranchID)
		argIdx++
	}

	if strings.TrimSpace(filter.Search) != "" {
		whereClauses = append(whereClauses, fmt.Sprintf("(u.number ILIKE $%d OR u.client_name ILIKE $%d OR u.contract_number ILIKE $%d)", argIdx, argIdx, argIdx))
		args = append(args, "%"+strings.TrimSpace(filter.Search)+"%")
		argIdx++
	}

	whereSQL := ""
	if len(whereClauses) > 0 {
		whereSQL = " WHERE " + strings.Join(whereClauses, " AND ")
	}

	countSQL := fmt.Sprintf("SELECT COUNT(*) FROM (%s) u%s", unionQuery, whereSQL)
	var total int
	err := r.db.QueryRow(ctx, countSQL, args...).Scan(&total)
	if err != nil {
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

	selectSQL := fmt.Sprintf(`
		SELECT 
			u.id, u.entity_type, u.entity_name, u.number, u.document_date,
			u.amount, u.currency, u.document_path, u.client_id, u.client_name,
			u.contract_id, u.contract_number, u.created_by, u.deleted_at
		FROM (%s) u%s
		ORDER BY u.deleted_at DESC
		LIMIT $%d OFFSET $%d`, unionQuery, whereSQL, argIdx, argIdx+1)

	args = append(args, pageSize, offset)

	rows, err := r.db.Query(ctx, selectSQL, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var items []dto.TrashItem
	for rows.Next() {
		var item dto.TrashItem
		err := rows.Scan(
			&item.ID, &item.EntityType, &item.EntityName, &item.Number, &item.DocumentDate,
			&item.Amount, &item.Currency, &item.DocumentPath, &item.ClientID, &item.ClientName,
			&item.ContractID, &item.ContractNumber, &item.CreatedBy, &item.DeletedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, item)
	}

	if items == nil {
		items = []dto.TrashItem{}
	}

	return items, total, nil
}

func (r *trashRepo) GetTrashItemByID(ctx context.Context, entityType string, id int64) (*dto.TrashItem, error) {
	var query string
	switch entityType {
	case "contract":
		query = `
			SELECT 
				c.id, 'contract' AS entity_type, 'Контракт' AS entity_name,
				c.contract_number AS number, c.contract_date::timestamp AS document_date,
				c.total_amount::numeric::float8 AS amount, c.contract_currency AS currency,
				COALESCE(c.document_path, '') AS document_path, c.client_id,
				COALESCE(cp.name, '') AS client_name, c.id AS contract_id,
				c.contract_number AS contract_number, COALESCE(c.created_by, '') AS created_by,
				c.deleted_at
			FROM contracts c
			LEFT JOIN counterparties cp ON cp.id = c.client_id
			WHERE c.id = $1 AND c.deleted_at IS NOT NULL`
	case "additional_agreement":
		query = `
			SELECT 
				aa.id, 'additional_agreement' AS entity_type,
				CASE 
					WHEN aa.doc_type = 'specification' THEN 'Спецификация'
					WHEN aa.doc_type = 'appendix' THEN 'Приложение'
					ELSE 'Доп. соглашение'
				END AS entity_name,
				COALESCE(aa.agreement_number, '') AS number, aa.agreement_date::timestamp AS document_date,
				COALESCE(aa.foreign_amount, 0)::numeric::float8 AS amount, COALESCE(aa.currency, '') AS currency,
				COALESCE(aa.document_path, '') AS document_path, c.client_id,
				COALESCE(cp.name, '') AS client_name, aa.contract_id,
				COALESCE(c.contract_number, '') AS contract_number, COALESCE(aa.created_by, '') AS created_by,
				aa.deleted_at
			FROM additional_agreements aa
			JOIN contracts c ON c.id = aa.contract_id
			LEFT JOIN counterparties cp ON cp.id = c.client_id
			WHERE aa.id = $1 AND aa.deleted_at IS NOT NULL`
	case "invoice":
		query = `
			SELECT 
				i.id, 'invoice' AS entity_type, 'Инвойс' AS entity_name,
				COALESCE(i.invoice_number, '') AS number, i.invoice_date::timestamp AS document_date,
				i.amount::numeric::float8 AS amount, i.currency AS currency,
				COALESCE(i.document_path, '') AS document_path, c.client_id,
				COALESCE(cp.name, '') AS client_name, i.contract_id,
				COALESCE(c.contract_number, '') AS contract_number, COALESCE(i.created_by, '') AS created_by,
				i.deleted_at
			FROM invoices i
			JOIN contracts c ON c.id = i.contract_id
			LEFT JOIN counterparties cp ON cp.id = c.client_id
			WHERE i.id = $1 AND i.deleted_at IS NOT NULL`
	case "gtd":
		query = `
			SELECT 
				g.id, 'gtd' AS entity_type,
				CASE 
					WHEN g.document_type = 'act' THEN 'Акт выполненных работ'
					ELSE 'ГТД'
				END AS entity_name,
				COALESCE(g.gtd_number, '') AS number, g.gtd_date::timestamp AS document_date,
				g.gtd_amount::numeric::float8 AS amount, COALESCE(g.gtd_currency, '') AS currency,
				COALESCE(g.document_path, '') AS document_path, c.client_id,
				COALESCE(cp.name, '') AS client_name, g.contract_id,
				COALESCE(c.contract_number, '') AS contract_number, COALESCE(g.created_by, '') AS created_by,
				g.deleted_at
			FROM gtd g
			JOIN contracts c ON c.id = g.contract_id
			LEFT JOIN counterparties cp ON cp.id = c.client_id
			WHERE g.id = $1 AND g.deleted_at IS NOT NULL`
	default:
		return nil, fmt.Errorf("неизвестный тип сущности: %s", entityType)
	}

	var item dto.TrashItem
	err := r.db.QueryRow(ctx, query, id).Scan(
		&item.ID, &item.EntityType, &item.EntityName, &item.Number, &item.DocumentDate,
		&item.Amount, &item.Currency, &item.DocumentPath, &item.ClientID, &item.ClientName,
		&item.ContractID, &item.ContractNumber, &item.CreatedBy, &item.DeletedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("документ не найден в корзине: %w", err)
	}

	return &item, nil
}

func (r *trashRepo) RestoreItem(ctx context.Context, entityType string, id int64) error {
	switch entityType {
	case "contract":
		tag, err := r.db.Exec(ctx, `UPDATE contracts SET deleted_at = NULL, updated_at = NOW() WHERE id = $1 AND deleted_at IS NOT NULL`, id)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("контракт не найден в корзине или уже восстановлен")
		}
		return nil

	case "additional_agreement":
		var contractID int64
		err := r.db.QueryRow(ctx, `SELECT contract_id FROM additional_agreements WHERE id = $1 AND deleted_at IS NOT NULL`, id).Scan(&contractID)
		if err != nil {
			return fmt.Errorf("доп. соглашение не найдено в корзине: %w", err)
		}
		_, err = r.db.Exec(ctx, `UPDATE additional_agreements SET deleted_at = NULL, updated_at = NOW() WHERE id = $1 AND deleted_at IS NOT NULL`, id)
		if err != nil {
			return err
		}
		syncAdditionalAgreementRemaining(ctx, r.db, id)
		return nil

	case "invoice":
		var contractID int64
		var addlID *int64
		err := r.db.QueryRow(ctx, `SELECT contract_id, additional_agreement_id FROM invoices WHERE id = $1 AND deleted_at IS NOT NULL`, id).Scan(&contractID, &addlID)
		if err != nil {
			return fmt.Errorf("инвойс не найден в корзине: %w", err)
		}
		_, err = r.db.Exec(ctx, `UPDATE invoices SET deleted_at = NULL, updated_at = NOW() WHERE id = $1 AND deleted_at IS NOT NULL`, id)
		if err != nil {
			return err
		}
		if addlID != nil {
			syncAdditionalAgreementRemaining(ctx, r.db, *addlID)
			tryArchiveAdditionalAgreement(ctx, r.db, *addlID)
		} else if contractID > 0 {
			syncContractRemaining(ctx, r.db, contractID)
			tryArchiveContract(ctx, r.db, contractID)
		}
		return nil

	case "gtd":
		var contractID int64
		var addlID *int64
		err := r.db.QueryRow(ctx, `SELECT contract_id, additional_agreement_id FROM gtd WHERE id = $1 AND deleted_at IS NOT NULL`, id).Scan(&contractID, &addlID)
		if err != nil {
			return fmt.Errorf("ГТД не найдена в корзине: %w", err)
		}
		_, err = r.db.Exec(ctx, `UPDATE gtd SET deleted_at = NULL, updated_at = NOW() WHERE id = $1 AND deleted_at IS NOT NULL`, id)
		if err != nil {
			return err
		}
		if addlID != nil {
			tryArchiveAdditionalAgreement(ctx, r.db, *addlID)
		} else if contractID > 0 {
			tryArchiveContract(ctx, r.db, contractID)
		}
		return nil

	default:
		return fmt.Errorf("неизвестный тип сущности: %s", entityType)
	}
}
