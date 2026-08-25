CREATE TABLE IF NOT EXISTS branches (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

INSERT INTO branches (name) VALUES 
    ('Головной'),
    ('Душанбе "Садбарг"'),
    ('Душанбе "Симург"'),
    ('Филиал "Вахдат"');

ALTER TABLE counterparties 
    ADD COLUMN IF NOT EXISTS inn VARCHAR;

ALTER TABLE contracts 
    ADD COLUMN IF NOT EXISTS client_id BIGINT REFERENCES counterparties(id),
    ADD COLUMN IF NOT EXISTS branch_id INT REFERENCES branches(id);
