with open('internal/domain/contract.go', 'r', encoding='utf-8') as f:
    text = f.read()

text = text.replace('ContractEndDate     *time.Time `db:"contract_end_date"`', 'ContractEndDate     *time.Time `db:"contract_end_date"`\n\tDocumentPath        *string    `db:"document_path" json:"document_path,omitempty"`\n\tOriginalDocumentName *string    `db:"original_document_name" json:"original_document_name,omitempty"`')

with open('internal/domain/contract.go', 'w', encoding='utf-8') as f:
    f.write(text)