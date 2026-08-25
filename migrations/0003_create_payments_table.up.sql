CREATE OR REPLACE FUNCTION calc_overdue_days()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.refund_date IS NOT NULL AND NEW.refund_date < CURRENT_DATE THEN
        NEW.overdue_days := (CURRENT_DATE - NEW.refund_date);
    ELSE
        NEW.overdue_days := 0;
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS payments (
    id BIGSERIAL PRIMARY KEY,
    contract_id BIGINT NOT NULL REFERENCES contracts(id) ON DELETE RESTRICT,
    payer_id BIGINT NOT NULL REFERENCES counterparties(id) ON DELETE RESTRICT,
    receiver_id BIGINT NOT NULL REFERENCES counterparties(id) ON DELETE RESTRICT,
    payment_number VARCHAR NOT NULL,
    payment_date DATE NOT NULL,
    currency_code CHAR NOT NULL,
    amount DECIMAL(18, 2) NOT NULL,
    delivery_date DATE,
    delivery_conditions TEXT,
    refund_date DATE,
    overdue_days INTEGER NOT NULL DEFAULT 0,
    receiver_country_code VARCHAR,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trg_payments_overdue_days
    BEFORE INSERT OR UPDATE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION calc_overdue_days();
