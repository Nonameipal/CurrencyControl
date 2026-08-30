import re

with open('internal/repository/contract.go', 'r', encoding='utf-8') as f:
    text = f.read()

# Interface
text = re.sub(r'\s+GetAll\(ctx context\.Context\) \(\[\]domain\.Contract, error\)', '', text)
text = re.sub(r'\s+Update\(ctx context\.Context, c domain\.Contract\) \(domain\.Contract, error\)', '', text)
text = re.sub(r'\s+Delete\(ctx context\.Context, id int64\) error', '', text)

# Delete functions
text = re.sub(r'func \(r \*contractRepo\) GetAll.*?return contracts, nil\n\}', '', text, flags=re.DOTALL)
text = re.sub(r'func \(r \*contractRepo\) Update.*?return result, nil\n\}', '', text, flags=re.DOTALL)
text = re.sub(r'func \(r \*contractRepo\) Delete.*?return nil\n\}', '', text, flags=re.DOTALL)

# Create
text = text.replace('additional_agreement, subject, contract_end_date', 'subject, contract_end_date, document_path, original_document_name')
text = text.replace('VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19)', 'VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)')
text = text.replace('COALESCE(additional_agreement, \'\'), COALESCE(subject, \'\')', 'COALESCE(subject, \'\')')
text = text.replace('contract_end_date, created_at, updated_at', 'contract_end_date, document_path, original_document_name, created_at, updated_at')
text = text.replace('c.AdditionalAgreement, c.Subject', 'c.Subject')
text = text.replace('c.ContractEndDate,', 'c.ContractEndDate, c.DocumentPath, c.OriginalDocumentName,')
text = text.replace('&result.AdditionalAgreement, &result.Subject', '&result.Subject')
text = text.replace('&result.ContractEndDate, &result.CreatedAt', '&result.ContractEndDate, &result.DocumentPath, &result.OriginalDocumentName, &result.CreatedAt')

# Remove Third party from Create
text = text.replace('third_party_name, third_party_account, third_party_country, ', '')
text = text.replace('VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20, $21, $22)', 'VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)')
text = text.replace('COALESCE(third_party_name, \'\'), COALESCE(third_party_account, \'\'), COALESCE(third_party_country, \'\'), ', '')
text = text.replace('c.ThirdPartyName, c.ThirdPartyAccount, c.ThirdPartyCountry, ', '')
text = text.replace('&result.ThirdPartyName, &result.ThirdPartyAccount, &result.ThirdPartyCountry, ', '')

# Remove Third party from GetByID / GetByClientID
text = text.replace('COALESCE(third_party_name, \'\'), COALESCE(third_party_account, \'\'), COALESCE(third_party_country, \'\'), ', '')

with open('internal/repository/contract.go', 'w', encoding='utf-8') as f:
    f.write(text)