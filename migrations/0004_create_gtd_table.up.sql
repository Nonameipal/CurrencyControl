CREATE TABLE IF NOT EXISTS gtd (
    id BIGSERIAL PRIMARY KEY,
    contract_id BIGINT NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    payment_id BIGINT REFERENCES payments(id) ON DELETE SET NULL,
    invoice_id BIGINT, 
    gtd_number VARCHAR NOT NULL,
    gtd_currency VARCHAR NOT NULL,
    gtd_date DATE,
    gtd_amount DECIMAL(18, 2) NOT NULL,
    gtd_currency VARCHAR,
    foreign_amount DECIMAL(18, 2),
    closes_amount DECIMAL(18,2) NOT NULL DEFAULT 0,
    document_path VARCHAR,
    original_document_name VARCHAR,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);
