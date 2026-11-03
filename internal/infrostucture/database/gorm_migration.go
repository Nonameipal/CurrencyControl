package database

import (
	"log"
	"os"
	"strings"

	"CurrencyControl/internal/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitGormDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(
		&domain.Branch{},
		&domain.Country{},
		&domain.Currency{},
		&domain.Counterparty{},
		&domain.Contract{},
		&domain.Invoice{},
		&domain.GTD{},
		&domain.GTDExtensionRequest{},
		&domain.PaymentOrder{},
		&domain.AdditionalAgreement{},
		&domain.User{},
		&domain.AccessRequest{},
		&domain.Session{},
		&domain.CurrencyControlPermission{}, 
		&domain.AuditLog{},
	)
	if err != nil {
		return nil, err
	}
	initDictsSQL, err := os.ReadFile("internal/infrostucture/database/init_data.sql")
	if err != nil {
		initDictsSQL, err = os.ReadFile("./internal/infrostucture/database/init_data.sql")
	}
	if err != nil {
		initDictsSQL, err = os.ReadFile("../../internal/infrostucture/database/init_data.sql")
	}
	if err == nil {
		cleanSQL := strings.ReplaceAll(string(initDictsSQL), "\ufeff", "")
		cleanSQL = strings.TrimSpace(cleanSQL)
		statements := strings.Split(cleanSQL, ";")
		for _, stmt := range statements {
			stmt = strings.TrimSpace(stmt)
			if stmt == "" {
				continue
			}
			if execErr := db.Exec(stmt).Error; execErr != nil {
				log.Printf("[MIGRATION WARNING] init_data.sql statement error: %v (stmt: %.50s...)", execErr, stmt)
			}
		}
	} else {
		log.Printf("[MIGRATION WARNING] could not read init_data.sql: %v", err)
	}

	db.Exec(`INSERT INTO currency_control_permissions (login, can_edit, can_delete, granted_by)
		VALUES ('*', true, true, 'compliance_system')
		ON CONFLICT (login) DO NOTHING;`)

	db.Exec(`UPDATE invoices SET hs_code = 'Нет кода' WHERE hs_code IS NULL OR TRIM(hs_code) = '';`)
	db.Exec(`UPDATE gtd SET hs_code = 'Нет кода' WHERE hs_code IS NULL OR TRIM(hs_code) = '';`)

	db.Exec(`
DROP TRIGGER IF EXISTS trg_invoices_addl_balance ON invoices;
DROP TRIGGER IF EXISTS trg_invoices_contract_balance ON invoices;
DROP TRIGGER IF EXISTS trg_addl_contract_balance ON additional_agreements;
DROP FUNCTION IF EXISTS sync_additional_agreement_balance();
DROP FUNCTION IF EXISTS sync_contract_balance();

CREATE OR REPLACE FUNCTION sync_contract_balance()
RETURNS TRIGGER AS $$
DECLARE
    target_id BIGINT;
BEGIN
    IF pg_trigger_depth() > 1 THEN
        RETURN NULL;
    END IF;

    IF TG_OP = 'DELETE' THEN
        target_id := OLD.contract_id;
    ELSE
        target_id := NEW.contract_id;
    END IF;

    IF target_id IS NOT NULL THEN
        UPDATE contracts
        SET total_amount = COALESCE(original_amount, total_amount) + COALESCE((
                SELECT SUM(COALESCE(foreign_amount, 0))
                FROM additional_agreements
                WHERE contract_id = contracts.id
                  AND deleted_at IS NULL
                  AND status = 'active'
                  AND currency = contracts.contract_currency
            ), 0),
            updated_at = NOW()
        WHERE id = target_id;

        UPDATE contracts
        SET remaining_amount = total_amount - COALESCE((
                SELECT SUM(COALESCE(NULLIF(deduct_amount, 0), amount))
                FROM invoices
                WHERE contract_id = contracts.id
                  AND currency = contracts.contract_currency
                  AND deleted_at IS NULL
            ), 0)
        WHERE id = target_id;

        UPDATE additional_agreements aa
        SET remaining_amount = CASE
            WHEN aa.status != 'active' THEN 0
            WHEN aa.currency = c.contract_currency THEN COALESCE(aa.foreign_amount, 0)
            ELSE COALESCE(aa.foreign_amount, 0) - COALESCE((
                SELECT SUM(inv.amount)
                FROM invoices inv
                WHERE inv.contract_id = aa.contract_id
                  AND inv.currency = aa.currency
                  AND inv.deleted_at IS NULL
            ), 0)
        END,
        updated_at = NOW()
        FROM contracts c
        WHERE aa.contract_id = c.id
          AND aa.contract_id = target_id
          AND aa.deleted_at IS NULL;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_invoices_contract_balance
    AFTER INSERT OR UPDATE OF amount, deduct_amount, currency, deleted_at, contract_id OR DELETE ON invoices
    FOR EACH ROW
    EXECUTE FUNCTION sync_contract_balance();

CREATE TRIGGER trg_addl_contract_balance
    AFTER INSERT OR UPDATE OF foreign_amount, currency, status, deleted_at, contract_id OR DELETE ON additional_agreements
    FOR EACH ROW
    EXECUTE FUNCTION sync_contract_balance();

UPDATE contracts SET original_amount = total_amount WHERE original_amount = 0 OR original_amount IS NULL;

UPDATE contracts
SET total_amount = COALESCE(original_amount, total_amount) + COALESCE((
        SELECT SUM(COALESCE(foreign_amount, 0))
        FROM additional_agreements
        WHERE contract_id = contracts.id
          AND deleted_at IS NULL
          AND status = 'active'
          AND currency = contracts.contract_currency
    ), 0)
WHERE deleted_at IS NULL;

UPDATE contracts
SET remaining_amount = total_amount - COALESCE((
        SELECT SUM(COALESCE(NULLIF(deduct_amount, 0), amount))
        FROM invoices
        WHERE contract_id = contracts.id
          AND currency = contracts.contract_currency
          AND deleted_at IS NULL
    ), 0)
WHERE deleted_at IS NULL;

UPDATE additional_agreements aa
SET remaining_amount = CASE
    WHEN aa.status != 'active' THEN 0
    WHEN aa.currency = c.contract_currency THEN COALESCE(aa.foreign_amount, 0)
    ELSE COALESCE(aa.foreign_amount, 0) - COALESCE((
        SELECT SUM(inv.amount)
        FROM invoices inv
        WHERE inv.contract_id = aa.contract_id
          AND inv.currency = aa.currency
          AND inv.deleted_at IS NULL
    ), 0)
END,
updated_at = NOW()
FROM contracts c
WHERE aa.contract_id = c.id
  AND aa.deleted_at IS NULL;
`)

	log.Println("GORM AutoMigrate and Triggers applied successfully")
	return db, nil
}
