import re

# Fix invoice.go
with open('internal/domain/invoice.go', 'r', encoding='utf-8') as f:
    text = f.read()
text = re.sub(r'\s*Document\s+\*Document\s+`json:"document,omitempty"`', '', text)
with open('internal/domain/invoice.go', 'w', encoding='utf-8') as f:
    f.write(text)

# Fix contract.go
with open('internal/repository/contract.go', 'r', encoding='utf-8') as f:
    text = f.read()
text = re.sub(r'func \(r \*contractRepo\) \{\s*query := `\s*SELECT.*?return contracts, nil\n\}', '', text, flags=re.DOTALL)
text = re.sub(r'func \(r \*contractRepo\) \{\s*query := `\s*UPDATE.*?return result, nil\n\}', '', text, flags=re.DOTALL)
text = re.sub(r'func \(r \*contractRepo\) \{\s*query := `\s*DELETE.*?return nil\n\}', '', text, flags=re.DOTALL)
with open('internal/repository/contract.go', 'w', encoding='utf-8') as f:
    f.write(text)