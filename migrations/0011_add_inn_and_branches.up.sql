CREATE TABLE IF NOT EXISTS branches (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL
);

INSERT INTO branches (name) VALUES 
    ('Головной'),
    ('Душанбе "Садбарг"'),
    ('Душанбе "Симург"'),
    ('Филиал "Вахдат"');

