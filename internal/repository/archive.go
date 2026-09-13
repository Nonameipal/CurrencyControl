package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"
)

// tryArchiveAdditionalAgreement проверяет условия архивации доп. соглашения:
//  1. remaining_amount <= 0 (все инвойсы загружены)
//  2. Все инвойсы закрыты ГТД (SUM(closes_amount) >= invoice.amount по каждому)
//
// Если условия выполнены — переводит доп. соглашение в статус "archived",
// затем пробует архивировать родительский контракт.
func tryArchiveAdditionalAgreement(ctx context.Context, db *pgxpool.Pool, addlID int64) {
	var contractID int64
	var remainingAmount float64
	err := db.QueryRow(ctx,
		`SELECT contract_id, remaining_amount FROM additional_agreements WHERE id = $1 AND deleted_at IS NULL AND status = 'active'`,
		addlID,
	).Scan(&contractID, &remainingAmount)
	if err != nil {
		return
	}

	if remainingAmount > 0 {
		return
	}

	// Проверяем, что каждый инвойс полностью закрыт ГТД
	var unclosedCount int
	_ = db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM invoices i
		WHERE i.additional_agreement_id = $1
		  AND i.deleted_at IS NULL
		  AND i.amount > COALESCE(
		      (SELECT SUM(g.closes_amount) FROM gtd g WHERE g.invoice_id = i.id AND g.deleted_at IS NULL),
		      0)`,
		addlID,
	).Scan(&unclosedCount)

	if unclosedCount > 0 {
		return
	}

	// Все условия выполнены — архивируем доп. соглашение
	_, _ = db.Exec(ctx,
		`UPDATE additional_agreements SET status = 'archived', archived_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		addlID,
	)

	// Пробуем архивировать родительский контракт
	if contractID > 0 {
		tryArchiveContract(ctx, db, contractID)
	}
}

// tryArchiveContract проверяет условия архивации контракта.
//
// Случай А — нет доп. соглашений:
//  1. remaining_amount <= 0
//  2. Все инвойсы закрыты ГТД
//
// Случай Б — есть доп. соглашения:
//  1. Случай А выполнен
//  2. ВСЕ доп. соглашения имеют status = "archived"
func tryArchiveContract(ctx context.Context, db *pgxpool.Pool, contractID int64) {
	var remainingAmount float64
	err := db.QueryRow(ctx,
		`SELECT remaining_amount FROM contracts WHERE id = $1 AND deleted_at IS NULL AND status = 'active'`,
		contractID,
	).Scan(&remainingAmount)
	if err != nil {
		return
	}

	if remainingAmount > 0 {
		return
	}

	// Проверяем, что все прямые инвойсы контракта (без доп. соглашения) закрыты ГТД
	var unclosedContractInvoices int
	_ = db.QueryRow(ctx, `
		SELECT COUNT(*)
		FROM invoices i
		WHERE i.contract_id = $1
		  AND i.additional_agreement_id IS NULL
		  AND i.deleted_at IS NULL
		  AND i.amount > COALESCE(
		      (SELECT SUM(g.closes_amount) FROM gtd g WHERE g.invoice_id = i.id AND g.deleted_at IS NULL),
		      0)`,
		contractID,
	).Scan(&unclosedContractInvoices)

	if unclosedContractInvoices > 0 {
		return
	}

	// Проверяем наличие активных доп. соглашений
	var activeAgreements int
	_ = db.QueryRow(ctx,
		`SELECT COUNT(*) FROM additional_agreements WHERE contract_id = $1 AND deleted_at IS NULL AND status = 'active'`,
		contractID,
	).Scan(&activeAgreements)

	if activeAgreements > 0 {
		return
	}

	// Все условия выполнены — архивируем контракт
	_, _ = db.Exec(ctx,
		`UPDATE contracts SET status = 'archived', archived_at = NOW(), updated_at = NOW() WHERE id = $1 AND deleted_at IS NULL`,
		contractID,
	)
}
