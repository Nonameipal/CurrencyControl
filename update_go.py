import re

# 1. Clean app.go
with open('internal/app/app.go', 'r', encoding='utf-8') as f:
    text = f.read()
text = re.sub(r'docRepo := repository\.NewDocumentRepository\(db\)\s+', '', text)
text = text.replace('service.NewContractService(contractRepo, docRepo)', 'service.NewContractService(contractRepo)')
with open('internal/app/app.go', 'w', encoding='utf-8') as f:
    f.write(text)

# 2. Clean invoice repo (remove documents table usage)
with open('internal/repository/invoice.go', 'r', encoding='utf-8') as f:
    text = f.read()
text = re.sub(r'\s*if inv\.DocumentPath == nil \{.*?\}\s*\}', '\n\t\t}', text, flags=re.DOTALL)
with open('internal/repository/invoice.go', 'w', encoding='utf-8') as f:
    f.write(text)

# 3. Clean invoice domain (remove Document fallback)
with open('internal/domain/invoice.go', 'r', encoding='utf-8') as f:
    text = f.read()
text = re.sub(r'\s*Document\s+\*Document\s+json:"document,omitempty"', '', text)
with open('internal/domain/invoice.go', 'w', encoding='utf-8') as f:
    f.write(text)

# 4. Clean contract domain (remove third party & old Document)
with open('internal/domain/contract.go', 'r', encoding='utf-8') as f:
    text = f.read()
text = re.sub(r'\s*IsThirdParty\s+bool\s+db:"is_third_party" json:"is_third_party"', '', text)
text = re.sub(r'\s*ThirdPartyName\s+string\s+db:"third_party_name" json:"third_party_name"', '', text)
text = re.sub(r'\s*ThirdPartyAccount\s+string\s+db:"third_party_account" json:"third_party_account"', '', text)
text = re.sub(r'\s*ThirdPartyCountry\s+string\s+db:"third_party_country" json:"third_party_country"', '', text)
text = re.sub(r'\s*AdditionalAgreement\s+string\s+db:"additional_agreement" json:"additional_agreement"', '', text)
text = re.sub(r'\s*Document\s+\*Document\s+json:"document,omitempty"', '', text)
with open('internal/domain/contract.go', 'w', encoding='utf-8') as f:
    f.write(text)
