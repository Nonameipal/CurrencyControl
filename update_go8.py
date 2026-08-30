import re
with open('internal/repository/contract.go', 'r', encoding='utf-8') as f:
    text = f.read()

# Fix GetByID Scan
text = re.sub(r'(&result\.ContractEndDate,)\s+(&result\.CreatedAt)', r'\1\n\t\t&result.DocumentPath,\n\t\t&result.OriginalDocumentName,\n\t\t\2', text)
text = text.replace('contract_end_date, created_at, updated_at\n\t\tFROM contracts', 'contract_end_date, document_path, original_document_name, created_at, updated_at\n\t\tFROM contracts')
text = text.replace('contract_end_date, created_at, updated_at\n\t\tFROM contracts\n\t\tWHERE client_id = $1', 'contract_end_date, document_path, original_document_name, created_at, updated_at\n\t\tFROM contracts\n\t\tWHERE client_id = $1')

with open('internal/repository/contract.go', 'w', encoding='utf-8') as f:
    f.write(text)