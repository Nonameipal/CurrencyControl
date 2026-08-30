import re

with open('internal/delivery/http/contract_handler.go', 'r', encoding='utf-8') as f:
    text = f.read()

text = re.sub(r'func \(h \*ContractHandler\) GetAll.*?\}\n\n', '', text, flags=re.DOTALL)
text = re.sub(r'func \(h \*ContractHandler\) Update.*?\}\n\n', '', text, flags=re.DOTALL)
text = re.sub(r'func \(h \*ContractHandler\) Delete.*?\}\n\n', '', text, flags=re.DOTALL)

# In Create method, remove ThirdParty logic
text = re.sub(r'c\.ThirdPartyName = r\.FormValue\("third_party_name"\)\s*c\.ThirdPartyAccount = r\.FormValue\("third_party_account"\)\s*c\.ThirdPartyCountry = r\.FormValue\("third_party_country"\)\s*', '', text)

# Remove the endpoints from router registration if they are there, wait, router registration is in routes.go
with open('internal/delivery/http/contract_handler.go', 'w', encoding='utf-8') as f:
    f.write(text)

with open('internal/delivery/http/routes.go', 'r', encoding='utf-8') as f:
    text = f.read()
text = re.sub(r'\s*router\.GET\("/contracts", contractHandler\.GetAll\)', '', text)
text = re.sub(r'\s*router\.PUT\("/contracts/:id", contractHandler\.Update\)', '', text)
text = re.sub(r'\s*router\.DELETE\("/contracts/:id", contractHandler\.Delete\)', '', text)
with open('internal/delivery/http/routes.go', 'w', encoding='utf-8') as f:
    f.write(text)