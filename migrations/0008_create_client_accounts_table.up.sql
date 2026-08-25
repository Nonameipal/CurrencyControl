CREATE TABLE IF NOT EXISTS client_accounts (
    id BIGSERIAL PRIMARY KEY,
    counterparty_id BIGINT NOT NULL REFERENCES counterparties(id) ON DELETE CASCADE,
    account_number TEXT NOT NULL,
    bank_name TEXT NOT NULL,
    bank_swift VARCHAR,
    currency_id INT NOT NULL REFERENCES currencies(id) ON DELETE RESTRICT,
    country_id INT NOT NULL REFERENCES countries(id) ON DELETE RESTRICT,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
