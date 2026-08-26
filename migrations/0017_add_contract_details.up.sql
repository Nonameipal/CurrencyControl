ALTER TABLE contracts 
    ADD COLUMN IF NOT EXISTS contract_name VARCHAR,
    ADD COLUMN IF NOT EXISTS delivery_date DATE,
    ADD COLUMN IF NOT EXISTS delivery_conditions TEXT,
    ADD COLUMN IF NOT EXISTS delivery_term_days INT,
    ADD COLUMN IF NOT EXISTS return_term_days INT,
    ADD COLUMN IF NOT EXISTS receiver_name VARCHAR,
    ADD COLUMN IF NOT EXISTS receiver_account VARCHAR,
    ADD COLUMN IF NOT EXISTS receiver_country VARCHAR;
