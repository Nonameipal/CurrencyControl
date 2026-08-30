with open('migrations/0000_init_dicts.up.sql', 'r', encoding='utf-8') as f:
    text = f.read()
text = text.replace('name TEXT NOT NULL\n);', 'name TEXT NOT NULL UNIQUE\n);')
with open('migrations/0000_init_dicts.up.sql', 'w', encoding='utf-8') as f:
    f.write(text)