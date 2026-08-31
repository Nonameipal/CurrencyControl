CREATE TABLE invoices (
    id             BIGSERIAL PRIMARY KEY,
    contract_id    BIGINT NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    invoice_number VARCHAR NOT NULL,
    invoice_name   VARCHAR NOT NULL,
    invoice_date   DATE NOT NULL,
    amount         DECIMAL(18,2) NOT NULL,
    currency       VARCHAR NOT NULL,
    foreign_amount DECIMAL(18, 2),
    deduct_amount  DECIMAL(18, 2),
    document_path VARCHAR,
    original_document_name VARCHAR,
    created_at     TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMP NOT NULL DEFAULT NOW()
);

ALTER TABLE gtd ADD CONSTRAINT fk_gtd_invoice FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE;
