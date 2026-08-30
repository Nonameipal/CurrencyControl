CREATE TABLE IF NOT EXISTS gtd (
    id BIGSERIAL PRIMARY KEY,
    gtd_number VARCHAR NOT NULL,
    gtd_amount DECIMAL(18, 2) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    document_path VARCHAR,
    original_document_name VARCHAR
);
