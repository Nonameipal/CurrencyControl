CREATE TABLE IF NOT EXISTS contracts (
    id BIGSERIAL PRIMARY KEY,
    contract_number VARCHAR NOT NULL,
    contract_date DATE NOT NULL,
    additional_agreement TEXT,
    subject TEXT NOT NULL,
    remaining_amount DECIMAL(18, 2) NOT NULL DEFAULT 0,
    contract_currency CHAR NOT NULL,
    contract_end_date DATE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
