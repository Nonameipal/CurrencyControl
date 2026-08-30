import os

# Delete the bad migration files
bad_files = [
    'migrations/0005_create_invoice_specs_table.up.sql',
    'migrations/0005_create_invoice_specs_table.down.sql',
    'migrations/0008_create_client_accounts_table.up.sql',
    'migrations/0008_create_client_accounts_table.down.sql',
    'migrations/0009_create_documents_table.up.sql',
    'migrations/0009_create_documents_table.down.sql',
    'migrations/0011_add_inn_and_branches.up.sql',
    'migrations/0011_add_inn_and_branches.down.sql',
    'migrations/0022_gtd_payment_id_nullable.up.sql',
    'migrations/0024_additional_agreement_contract_fields.up.sql',
    'migrations/0025_invoice_gtd_documents.up.sql'
]
for f in bad_files:
    if os.path.exists(f):
        os.remove(f)

# Write 0000_init_dicts.up.sql to create branches, countries, currencies before counterparties
with open('migrations/0000_init_dicts.up.sql', 'w', encoding='utf-8') as f:
    f.write('''CREATE TABLE IF NOT EXISTS branches (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS countries (
    id SERIAL PRIMARY KEY,
    name_ru TEXT NOT NULL,
    name_en TEXT NOT NULL,
    code_2 CHAR(2) NOT NULL,
    code_3 CHAR(3) NOT NULL
);

CREATE TABLE IF NOT EXISTS currencies (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    code CHAR(3) NOT NULL,
    number CHAR(3) NOT NULL
);
''')

with open('migrations/0000_init_dicts.down.sql', 'w', encoding='utf-8') as f:
    f.write('''DROP TABLE IF EXISTS branches CASCADE;
DROP TABLE IF EXISTS countries CASCADE;
DROP TABLE IF EXISTS currencies CASCADE;
''')

# We can remove 0006 and 0007 since they are now in 0000
if os.path.exists('migrations/0006_create_currencies_table.up.sql'): os.remove('migrations/0006_create_currencies_table.up.sql')
if os.path.exists('migrations/0006_create_currencies_table.down.sql'): os.remove('migrations/0006_create_currencies_table.down.sql')
if os.path.exists('migrations/0007_create_countries_table.up.sql'): os.remove('migrations/0007_create_countries_table.up.sql')
if os.path.exists('migrations/0007_create_countries_table.down.sql'): os.remove('migrations/0007_create_countries_table.down.sql')

# Fix 0001
with open('migrations/0001_create_counterparties_table.up.sql', 'w', encoding='utf-8') as f:
    f.write('''CREATE TABLE IF NOT EXISTS counterparties (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    email TEXT,
    branch_id INT REFERENCES branches(id),
    inn VARCHAR(22),
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
''')

# Fix 0002
with open('migrations/0002_create_contracts_table.up.sql', 'w', encoding='utf-8') as f:
    f.write('''CREATE TABLE IF NOT EXISTS contracts (
    id BIGSERIAL PRIMARY KEY,
    client_id BIGINT REFERENCES counterparties(id) ON DELETE RESTRICT,
    branch_id INT REFERENCES branches(id) ON DELETE RESTRICT,
    contract_number VARCHAR NOT NULL,
    contract_name VARCHAR,
    contract_date DATE NOT NULL,
    delivery_date DATE,
    delivery_conditions TEXT,
    delivery_term_days INT,
    return_term_days INT,
    total_amount DECIMAL(18, 2) NOT NULL DEFAULT 0,
    remaining_amount DECIMAL(18, 2) NOT NULL DEFAULT 0,
    contract_currency CHAR(3) NOT NULL,
    sender_account VARCHAR,
    receiver_name VARCHAR,
    receiver_account VARCHAR,
    receiver_country VARCHAR,
    subject TEXT NOT NULL,
    contract_end_date DATE,
    document_path VARCHAR,
    original_document_name VARCHAR,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
''')

# Fix 0004
with open('migrations/0004_create_gtd_table.up.sql', 'w', encoding='utf-8') as f:
    f.write('''CREATE TABLE IF NOT EXISTS gtd (
    id BIGSERIAL PRIMARY KEY,
    contract_id BIGINT NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    payment_id BIGINT REFERENCES payments(id) ON DELETE SET NULL,
    invoice_id BIGINT, -- Will reference invoices(id) later
    gtd_number VARCHAR NOT NULL,
    gtd_date DATE,
    amount DECIMAL(18, 2) NOT NULL,
    foreign_amount DECIMAL(18, 2),
    closes_amount DECIMAL(18,2) NOT NULL DEFAULT 0,
    document_path VARCHAR,
    original_document_name VARCHAR,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
''')

# Fix 0021
with open('migrations/0021_create_invoices_alter_gtd.up.sql', 'w', encoding='utf-8') as f:
    f.write('''CREATE TABLE invoices (
    id             BIGSERIAL PRIMARY KEY,
    contract_id    BIGINT NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    invoice_number VARCHAR NOT NULL,
    invoice_name   VARCHAR NOT NULL,
    invoice_date   DATE NOT NULL,
    amount         DECIMAL(18,2) NOT NULL,
    currency       VARCHAR NOT NULL,
    foreign_amount DECIMAL(18, 2),
    amount_in_contract_currency DECIMAL(18, 2),
    document_path VARCHAR,
    original_document_name VARCHAR,
    created_at     TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMP NOT NULL DEFAULT NOW()
);

ALTER TABLE gtd ADD CONSTRAINT fk_gtd_invoice FOREIGN KEY (invoice_id) REFERENCES invoices(id) ON DELETE CASCADE;
''')

# Fix 0023
with open('migrations/0023_additional_agreements.up.sql', 'w', encoding='utf-8') as f:
    f.write('''CREATE TABLE IF NOT EXISTS additional_agreements (
    id BIGSERIAL PRIMARY KEY,
    contract_id BIGINT NOT NULL REFERENCES contracts(id) ON DELETE CASCADE,
    agreement_number VARCHAR NOT NULL,
    agreement_date DATE NOT NULL,
    
    extend_date_to DATE,
    increase_amount DECIMAL(18, 2),
    new_delivery_conditions TEXT,
    new_delivery_term_days INT,
    new_return_term_days INT,
    
    currency VARCHAR NOT NULL,
    foreign_amount DECIMAL(18, 2),
    amount_in_contract_currency DECIMAL(18, 2),

    document_path VARCHAR,
    original_document_name VARCHAR,
    
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW()
);
''')