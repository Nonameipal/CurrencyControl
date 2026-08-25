CREATE TABLE IF NOT EXISTS countries (
    id SERIAL PRIMARY KEY,
    country_code CHAR(2) NOT NULL UNIQUE,
    name_ru  TEXT NOT NULL
);

INSERT INTO countries (country_code, name_ru) VALUES
    ('CN', 'Китай'),
    ('DE', 'Германия'),
    ('GB', 'Великобритания'),
    ('GE', 'Грузия'),
    ('JP', 'Япония'),
    ('KG', 'Кыргызстан'),
    ('KZ', 'Казахстан'),
    ('RU', 'Россия'),
    ('TJ', 'Таджикистан'),
    ('US', 'США'),
    ('UZ', 'Узбекистан');
