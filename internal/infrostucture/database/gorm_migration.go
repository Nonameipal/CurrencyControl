package database

import (
	"fmt"
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

	db.Exec(`ALTER TABLE gtd ADD COLUMN IF NOT EXISTS currency varchar(10) DEFAULT 'USD';`)
	db.Exec(`UPDATE gtd SET currency = 'USD' WHERE currency IS NULL OR currency = '';`)

	err = db.AutoMigrate(
		&domain.Branch{},
		&domain.Country{},
		&domain.Currency{},
		&domain.Counterparty{},
		&domain.Contract{},
		&domain.Invoice{},
		&domain.GTD{},
		&domain.Act{},
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

	db.Exec(`DELETE FROM currency_control_permissions WHERE login = '*';`)

	db.Exec(`ALTER TABLE contracts DROP COLUMN IF EXISTS sender_bank;`)
	db.Exec(`ALTER TABLE contracts DROP COLUMN IF EXISTS sender_country;`)
	db.Exec(`DROP TRIGGER IF EXISTS trg_invoices_addl_balance ON invoices;`)
	db.Exec(`DROP TRIGGER IF EXISTS trg_invoices_contract_balance ON invoices;`)
	db.Exec(`ALTER TABLE invoices DROP COLUMN IF EXISTS sender_name CASCADE;`)
	db.Exec(`ALTER TABLE invoices DROP COLUMN IF EXISTS sender_bank CASCADE;`)
	db.Exec(`ALTER TABLE invoices DROP COLUMN IF EXISTS sender_country CASCADE;`)
	db.Exec(`ALTER TABLE invoices DROP COLUMN IF EXISTS hs_code CASCADE;`)
	db.Exec(`ALTER TABLE invoices DROP COLUMN IF EXISTS deduct_amount CASCADE;`)

	for _, col := range []string{
		"approval_status",
		"currency_control_decision",
		"currency_control_comment",
		"currency_control_reviewed_by",
		"currency_control_reviewed_at",
		"compliance_decision",
		"compliance_comment",
		"compliance_reviewed_by",
		"compliance_reviewed_at",
		"rejection_reason",
	} {
		db.Exec(fmt.Sprintf(`ALTER TABLE contracts DROP COLUMN IF EXISTS %s CASCADE;`, col))
		db.Exec(fmt.Sprintf(`ALTER TABLE invoices DROP COLUMN IF EXISTS %s CASCADE;`, col))
		db.Exec(fmt.Sprintf(`ALTER TABLE additional_agreements DROP COLUMN IF EXISTS %s CASCADE;`, col))
		db.Exec(fmt.Sprintf(`ALTER TABLE acts DROP COLUMN IF EXISTS %s CASCADE;`, col))
	}

	for _, col := range []string{
		"operation_date",
		"payment_order_number",
		"payer",
		"sender_bank",
		"sender_country",
		"contract_number",
		"invoice_number",
	} {
		db.Exec(fmt.Sprintf(`ALTER TABLE payment_orders DROP COLUMN IF EXISTS %s CASCADE;`, col))
	}

	for _, col := range []string{
		"sender_name",
		"sender_bank",
		"sender_country",
		"receiver_name",
		"receiver_bank",
		"receiver_country",
	} {
		db.Exec(fmt.Sprintf(`ALTER TABLE additional_agreements DROP COLUMN IF EXISTS %s CASCADE;`, col))
	}

	db.Exec(`ALTER TABLE gtd ADD COLUMN IF NOT EXISTS currency varchar(10) DEFAULT 'USD';`)
	db.Exec(`DO $$ 
	BEGIN 
		IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='gtd' AND column_name='gtd_currency') THEN 
			UPDATE gtd SET currency = COALESCE(NULLIF(gtd_currency, ''), 'USD') WHERE currency = '' OR currency IS NULL; 
		END IF; 
	END $$;`)

	db.Exec(`
		DO $$
		BEGIN
			IF EXISTS (SELECT 1 FROM information_schema.columns WHERE table_name='gtd' AND column_name='document_type') THEN
				INSERT INTO acts (contract_id, invoice_id, act_number, act_date, act_amount, currency, document_path, created_by, approval_status, currency_control_decision, currency_control_comment, currency_control_reviewed_by, currency_control_reviewed_at, compliance_decision, compliance_comment, compliance_reviewed_by, compliance_reviewed_at, rejection_reason, created_at, updated_at, deleted_at, updated_by, deleted_by)
				SELECT contract_id, invoice_id, gtd_number, COALESCE(gtd_date, created_at), gtd_amount, COALESCE(currency, 'USD'), document_path, created_by, approval_status, currency_control_decision, currency_control_comment, currency_control_reviewed_by, currency_control_reviewed_at, compliance_decision, compliance_comment, compliance_reviewed_by, compliance_reviewed_at, rejection_reason, created_at, updated_at, deleted_at, updated_by, deleted_by
				FROM gtd
				WHERE document_type = 'act';
				DELETE FROM gtd WHERE document_type = 'act';
			END IF;
		END $$;
	`)

	for _, col := range []string{
		"document_type",
		"hs_code",
		"destination_country",
		"sender_name",
		"sender_bank",
		"sender_country",
		"gtd_currency",
	} {
		db.Exec(fmt.Sprintf(`ALTER TABLE gtd DROP COLUMN IF EXISTS %s CASCADE;`, col))
	}

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
                SELECT SUM(amount)
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
    AFTER INSERT OR UPDATE OF amount, currency, deleted_at, contract_id OR DELETE ON invoices
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
        SELECT SUM(amount)
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
