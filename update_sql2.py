with open('migrations/0000_init_dicts.up.sql', 'r', encoding='utf-8') as f:
    text = f.read()

# Replace countries table definition
old_countries = '''CREATE TABLE IF NOT EXISTS countries (
    id SERIAL PRIMARY KEY,
    name_ru TEXT NOT NULL,
    name_en TEXT NOT NULL,
    code_2 CHAR(2) NOT NULL,
    code_3 CHAR(3) NOT NULL
);'''
new_countries = '''CREATE TABLE IF NOT EXISTS countries (
    id SERIAL PRIMARY KEY,
    name_ru TEXT NOT NULL UNIQUE
);'''
text = text.replace(old_countries, new_countries)

# Append inserts from 0020 (branches) and 0012 (countries)
import os
branches_insert = ''
if os.path.exists('migrations/0020_recreate_branches_list.up.sql'):
    with open('migrations/0020_recreate_branches_list.up.sql', 'r', encoding='utf-8') as f:
        branches_insert = f.read().replace('DROP TABLE IF EXISTS branches CASCADE;', '').replace('CREATE TABLE branches (\n    id SERIAL PRIMARY KEY,\n    name TEXT NOT NULL UNIQUE\n);', '').strip()

countries_insert = ''
if os.path.exists('migrations/0012_recreate_countries_list.up.sql'):
    with open('migrations/0012_recreate_countries_list.up.sql', 'r', encoding='utf-8') as f:
        countries_insert = f.read().replace('DROP TABLE IF EXISTS countries CASCADE;', '').replace('CREATE TABLE countries (\n    id SERIAL PRIMARY KEY,\n    name_ru TEXT NOT NULL UNIQUE\n);', '').strip()

# Combine
text += '\n\n' + branches_insert + '\n\n' + countries_insert

with open('migrations/0000_init_dicts.up.sql', 'w', encoding='utf-8') as f:
    f.write(text)

# Now delete the obsolete files
if os.path.exists('migrations/0012_recreate_countries_list.up.sql'): os.remove('migrations/0012_recreate_countries_list.up.sql')
if os.path.exists('migrations/0012_recreate_countries_list.down.sql'): os.remove('migrations/0012_recreate_countries_list.down.sql')
if os.path.exists('migrations/0020_recreate_branches_list.up.sql'): os.remove('migrations/0020_recreate_branches_list.up.sql')
if os.path.exists('migrations/0020_recreate_branches_list.down.sql'): os.remove('migrations/0020_recreate_branches_list.down.sql')