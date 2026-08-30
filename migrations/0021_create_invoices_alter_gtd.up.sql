CREATE TABLE invoices (
    id             BIGSERIAL PRIMARY KEY,
    contract_id    BIGINT NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    invoice_number VARCHAR NOT NULL,
    invoice_name   VARCHAR NOT NULL,
    invoice_date   DATE NOT NULL,
    amount         DECIMAL(18,2) NOT NULL,
    currency       VARCHAR NOT NULL,
    created_at     TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMP NOT NULL DEFAULT NOW()
    document_path VARCHAR,
    original_document_name VARCHAR
);

ALTER TABLE gtd ADD COLUMN IF NOT EXISTS invoice_id BIGINT REFERENCES invoices(id) ON DELETE CASCADE;
ALTER TABLE gtd ADD COLUMN IF NOT EXISTS gtd_date DATE;