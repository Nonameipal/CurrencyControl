CREATE TABLE IF NOT EXISTS additional_agreements (
    id                          BIGSERIAL PRIMARY KEY,
    contract_id                 BIGINT NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    extend_date_to              DATE,
    foreign_amount              DECIMAL(18,2),
    foreign_currency            CHAR(3),
    amount_in_contract_currency DECIMAL(18,2) DEFAULT 0,
    document_path               VARCHAR,
    original_document_name      VARCHAR,
    created_at                  TIMESTAMP NOT NULL DEFAULT NOW()
);

ALTER TABLE invoices ADD COLUMN IF NOT EXISTS deduct_amount DECIMAL(18,2) DEFAULT 0 NOT NULL;

ALTER TABLE gtd ADD COLUMN IF NOT EXISTS gtd_currency CHAR(3);
ALTER TABLE gtd ADD COLUMN IF NOT EXISTS closes_amount DECIMAL(18,2) DEFAULT 0 NOT NULL;