CREATE TABLE IF NOT EXISTS currencies (
    id SERIAL PRIMARY KEY,
    code CHAR(3) NOT NULL UNIQUE,
    numeric_code SMALLINT NOT NULL UNIQUE,
    name_ru  TEXT NOT NULL
);

INSERT INTO currencies (code, numeric_code, name_ru) VALUES
    ('CNY', 156, 'Китайский юань (ренминби)'),
    ('EUR', 978, 'Евро'),
    ('GBP', 826, 'Английский фунт стерлингов'),
    ('GEL', 981, 'Грузинский лари'),
    ('JPY', 392, 'Японская иена'),
    ('KGS', 417, 'Кыргызский сом'),
    ('KZT', 398, 'Казахский тенге'),
    ('RUB', 643, 'Российский рубль'),
    ('SDR', 999, 'СДР (МВФ)'),
    ('TJS', 972, 'Таджикский сомони'),
    ('USD', 840, 'Доллар США'),
    ('UZS', 860, 'Узбекский сум');
