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
    swift_deadline DATE,
    overdue_days INTEGER  NOT NULL DEFAULT 0,
    receiver_country_code VARCHAR(2),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE TRIGGER trg_payments_overdue_days
    BEFORE INSERT OR UPDATE ON payments
    FOR EACH ROW
    EXECUTE FUNCTION calc_overdue_days();
