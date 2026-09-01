CREATE TABLE IF NOT EXISTS additional_agreements (
    id BIGSERIAL PRIMARY KEY,
    contract_id BIGINT NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    agreement_number VARCHAR,
    agreement_date DATE,
    
    extend_date_to DATE,
    increase_amount DECIMAL(18, 2),
    new_delivery_conditions TEXT,
    new_delivery_term_days INT,
    new_return_term_days INT,
    subject TEXT,
    
    currency VARCHAR,
    foreign_amount DECIMAL(18, 2),
    amount_in_contract_currency DECIMAL(18, 2),

    document_path VARCHAR,
    original_document_name VARCHAR,
    created_by VARCHAR(255) DEFAULT '',
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMP
);

