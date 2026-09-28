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
		&domain.Payment{},
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

	db.Exec("ALTER TABLE contracts DROP COLUMN IF EXISTS return_term_days;")
	db.Exec("ALTER TABLE counterparties DROP COLUMN IF EXISTS name;")

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
            - COALESCE((SELECT SUM(deduct_amount) FROM invoices WHERE contract_id = contracts.id AND additional_agreement_id IS NULL AND deleted_at IS NULL), 0),
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

-- Доп. соглашения больше не влияют на баланс контракта
DROP TRIGGER IF EXISTS trg_addl_contract_balance ON additional_agreements;

-- Функция и триггер пересчета баланса доп. соглашения
CREATE OR REPLACE FUNCTION sync_additional_agreement_balance()
RETURNS TRIGGER AS $$
DECLARE
    target_id BIGINT;
BEGIN
    IF TG_OP = 'DELETE' THEN
        target_id := OLD.additional_agreement_id;
    ELSE
        target_id := NEW.additional_agreement_id;
    END IF;

    IF target_id IS NOT NULL THEN
        UPDATE additional_agreements
        SET remaining_amount = COALESCE(foreign_amount, 0)
            - COALESCE((SELECT SUM(amount) FROM invoices WHERE additional_agreement_id = additional_agreements.id AND deleted_at IS NULL), 0),
            updated_at = NOW()
        WHERE id = target_id;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_invoices_addl_balance ON invoices;
CREATE TRIGGER trg_invoices_addl_balance
    AFTER INSERT OR UPDATE OR DELETE ON invoices
    FOR EACH ROW
    EXECUTE FUNCTION sync_additional_agreement_balance();

UPDATE contracts
SET remaining_amount = total_amount 
    - COALESCE((SELECT SUM(deduct_amount) FROM invoices WHERE contract_id = contracts.id AND additional_agreement_id IS NULL AND deleted_at IS NULL), 0)
WHERE deleted_at IS NULL;

UPDATE additional_agreements
SET remaining_amount = COALESCE(foreign_amount, 0)
    - COALESCE((SELECT SUM(amount) FROM invoices WHERE additional_agreement_id = additional_agreements.id AND deleted_at IS NULL), 0)
WHERE deleted_at IS NULL;
`)

	log.Println("GORM AutoMigrate and Triggers applied successfully")
	return db, nil
}
