CREATE TABLE IF NOT EXISTS invoice_specs (
    id BIGSERIAL PRIMARY KEY,
    payment_id  BIGINT NOT NULL REFERENCES payments(id) ON DELETE CASCADE,
    spec_name TEXT NOT NULL,
    spec_date DATE NOT NULL,
    spec_amount DECIMAL(18, 2) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
