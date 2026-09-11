package database

import (
	"CurrencyControl/internal/domain"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"log"
	"os"
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
		&domain.Payment{},
		&domain.Invoice{},
		&domain.GTD{},
		&domain.AdditionalAgreement{},
		&domain.User{},
		&domain.AccessRequest{},
		&domain.Session{},
		&domain.AuditLog{},
	)
	if err != nil {
		return nil, err
	}

	initDictsSQL, err := os.ReadFile("internal/infrostucture/database/init_data.sql")
	if err == nil {
		db.Exec(string(initDictsSQL))
	}

	db.Exec(`ALTER TABLE contracts ADD COLUMN IF NOT EXISTS return_date DATE;`)
	db.Exec(`ALTER TABLE gtd ADD COLUMN IF NOT EXISTS document_type VARCHAR(50) DEFAULT 'gtd';`)
	db.Exec(`ALTER TABLE additional_agreements ADD COLUMN IF NOT EXISTS doc_type VARCHAR(50) DEFAULT 'additional_agreement';`)
	db.Exec(`ALTER TABLE additional_agreements ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP NOT NULL DEFAULT NOW();`)
	db.Exec(`ALTER TABLE additional_agreements ADD COLUMN IF NOT EXISTS delivery_date DATE;`)
	db.Exec(`ALTER TABLE additional_agreements ADD COLUMN IF NOT EXISTS return_date DATE;`)
	db.Exec(`ALTER TABLE invoices ADD COLUMN IF NOT EXISTS hs_code VARCHAR(50) DEFAULT '';`)
	db.Exec(`ALTER TABLE gtd ADD COLUMN IF NOT EXISTS hs_code VARCHAR(50) DEFAULT '';`)
	db.Exec(`ALTER TABLE gtd ADD COLUMN IF NOT EXISTS destination_country VARCHAR(255) DEFAULT '';`)
	db.Exec(`ALTER TABLE gtd ADD COLUMN IF NOT EXISTS submission_date DATE;`)
	db.Exec(`ALTER TABLE gtd ADD COLUMN IF NOT EXISTS delivery_deadline DATE;`)
	db.Exec(`ALTER TABLE gtd ADD COLUMN IF NOT EXISTS days_difference INT DEFAULT 0;`)
	db.Exec(`ALTER TABLE gtd ADD COLUMN IF NOT EXISTS delivery_status VARCHAR(50) DEFAULT '';`)
	db.Exec(`ALTER TABLE gtd ADD COLUMN IF NOT EXISTS delivery_notice TEXT DEFAULT '';`)


	db.Exec(`
CREATE OR REPLACE FUNCTION calc_overdue_days()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.delivery_date IS NOT NULL THEN
        NEW.swift_deadline := NEW.delivery_date + INTERVAL '5 days';
    ELSE
        NEW.swift_deadline := NULL;
    END IF;

    IF NEW.swift_deadline IS NOT NULL AND NEW.swift_deadline < CURRENT_DATE THEN
        NEW.overdue_days := (CURRENT_DATE - NEW.swift_deadline);
    ELSE
        NEW.overdue_days := 0;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
`)
	db.Exec(`DROP TRIGGER IF EXISTS trg_payments_overdue_days ON payments;`)
	db.Exec(`
CREATE TRIGGER trg_payments_overdue_days
    BEFORE INSERT OR UPDATE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION calc_overdue_days();
`)


	db.Exec(`
CREATE OR REPLACE FUNCTION manage_contract_balance()
RETURNS TRIGGER AS $$
DECLARE
    v_remaining DECIMAL(18, 2);
    v_old_amount DECIMAL(18, 2) := 0;
BEGIN
    IF TG_OP = 'DELETE' THEN
        UPDATE contracts
        SET remaining_amount = remaining_amount + OLD.amount,
            updated_at = NOW()
        WHERE id = OLD.contract_id;
        RETURN OLD;
    END IF;

    IF TG_OP = 'UPDATE' THEN
        v_old_amount := OLD.amount;
    END IF;

    SELECT remaining_amount INTO v_remaining
    FROM contracts
    WHERE id = NEW.contract_id
    FOR UPDATE;

    IF (NEW.amount - v_old_amount) > v_remaining THEN
        RAISE EXCEPTION
            'Сумма платежа превышает остаток по контракту.';
    END IF;

    UPDATE contracts
    SET remaining_amount = remaining_amount - (NEW.amount - v_old_amount),
        updated_at = NOW()
    WHERE id = NEW.contract_id;

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;
`)
	db.Exec(`DROP TRIGGER IF EXISTS trg_payments_contract_balance ON payments;`)
	db.Exec(`
CREATE TRIGGER trg_payments_contract_balance
    AFTER INSERT OR UPDATE OR DELETE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION manage_contract_balance();
`)

	db.Exec(`
CREATE OR REPLACE FUNCTION sync_contract_balance()
RETURNS TRIGGER AS $$
DECLARE
    target_id BIGINT;
BEGIN
    IF TG_OP = 'DELETE' THEN
        target_id := OLD.contract_id;
    ELSE
        target_id := NEW.contract_id;
    END IF;

    IF target_id IS NOT NULL THEN
        UPDATE contracts
        SET remaining_amount = total_amount 
            + COALESCE((SELECT SUM(amount_in_contract_currency) FROM additional_agreements WHERE contract_id = contracts.id AND deleted_at IS NULL), 0)
            - COALESCE((SELECT SUM(deduct_amount) FROM invoices WHERE contract_id = contracts.id AND deleted_at IS NULL), 0),
            updated_at = NOW()
        WHERE id = target_id;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_invoices_contract_balance ON invoices;
CREATE TRIGGER trg_invoices_contract_balance
    AFTER INSERT OR UPDATE OR DELETE ON invoices
    FOR EACH ROW
    EXECUTE FUNCTION sync_contract_balance();

DROP TRIGGER IF EXISTS trg_addl_contract_balance ON additional_agreements;
CREATE TRIGGER trg_addl_contract_balance
    AFTER INSERT OR UPDATE OR DELETE ON additional_agreements
    FOR EACH ROW
    EXECUTE FUNCTION sync_contract_balance();

UPDATE contracts
SET remaining_amount = total_amount 
    + COALESCE((SELECT SUM(amount_in_contract_currency) FROM additional_agreements WHERE contract_id = contracts.id AND deleted_at IS NULL), 0)
    - COALESCE((SELECT SUM(deduct_amount) FROM invoices WHERE contract_id = contracts.id AND deleted_at IS NULL), 0)
WHERE deleted_at IS NULL;
`)

	log.Println("GORM AutoMigrate and Triggers applied successfully")
	return db, nil
}
