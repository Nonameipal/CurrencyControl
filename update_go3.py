import re

# Ports
with open('internal/service/ports/contract.go', 'r', encoding='utf-8') as f:
    text = f.read()
text = re.sub(r'\s+GetAll\(ctx context\.Context\) \(\[\]domain\.Contract, error\)', '', text)
text = re.sub(r'\s+Update\(ctx context\.Context, id int64, c domain\.Contract, doc \*multipart\.FileHeader\) \(domain\.Contract, error\)', '', text)
text = re.sub(r'\s+Delete\(ctx context\.Context, id int64\) error', '', text)
with open('internal/service/ports/contract.go', 'w', encoding='utf-8') as f:
    f.write(text)

# Service
with open('internal/service/contract.go', 'r', encoding='utf-8') as f:
    text = f.read()

text = text.replace('type contractService struct {\n\trepo    repository.ContractRepository\n\tdocRepo repository.DocumentRepository\n}', 'type contractService struct {\n\trepo    repository.ContractRepository\n}')
text = text.replace('func NewContractService(repo repository.ContractRepository, docRepo repository.DocumentRepository) ports.ContractService {\n\treturn &contractService{\n\t\trepo:    repo,\n\t\tdocRepo: docRepo,\n\t}\n}', 'func NewContractService(repo repository.ContractRepository) ports.ContractService {\n\treturn &contractService{\n\t\trepo:    repo,\n\t}\n}')

text = re.sub(r'func \(s \*contractService\) GetAll.*?return s\.repo\.GetAll\(ctx\)\n\}', '', text, flags=re.DOTALL)
text = re.sub(r'func \(s \*contractService\) Update.*?return updated, nil\n\}', '', text, flags=re.DOTALL)
text = re.sub(r'func \(s \*contractService\) Delete.*?return s\.repo\.Delete\(ctx, id\)\n\}', '', text, flags=re.DOTALL)

with open('internal/service/contract.go', 'w', encoding='utf-8') as f:
    f.write(text)