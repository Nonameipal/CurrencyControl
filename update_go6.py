import re

# Fix invoice.go
with open('internal/repository/invoice.go', 'r', encoding='utf-8') as f:
    text = f.read()

# I will just write a python script that cleans up the if inv.DocumentPath == nil manually better.
text = text.replace('\t\t}\n\t\t}\n\t\t}\n\n\t\tresult = append(result, inv)\n\t}', '\n\t\tresult = append(result, inv)\n\t}')
text = text.replace('\t\t}\n\n\t\tresult = append(result, inv)\n\t}', '\n\t\tresult = append(result, inv)\n\t}')
# Just replace any occurrence of too many brackets before result = append(result, inv)
text = re.sub(r'(\}\s*)+\s*result = append\(result, inv\)', '}\n\t\tresult = append(result, inv)', text)
with open('internal/repository/invoice.go', 'w', encoding='utf-8') as f:
    f.write(text)

# Fix contract ports
with open('internal/service/ports/contract.go', 'r', encoding='utf-8') as f:
    text = f.read()
text = re.sub(r'\s+GetAll\(.*?\).*', '', text)
text = re.sub(r'\s+Update\(.*?\).*', '', text)
text = re.sub(r'\s+Delete\(.*?\).*', '', text)
text = re.sub(r'\s+CreateWithDocument\(.*?\).*', '', text)
with open('internal/service/ports/contract.go', 'w', encoding='utf-8') as f:
    f.write(text)

# Fix contract service
with open('internal/service/contract.go', 'r', encoding='utf-8') as f:
    text = f.read()
text = re.sub(r'func \(s \*contractService\) GetAll\(ctx context\.Context, login string\) \(\[\]domain\.Contract, error\) \{.*?return s\.repo\.GetAll\(ctx\)\n\}', '', text, flags=re.DOTALL)
text = re.sub(r'func \(s \*contractService\) Update.*?return updated, nil\n\}', '', text, flags=re.DOTALL)
text = re.sub(r'func \(s \*contractService\) Delete.*?return s\.repo\.Delete\(ctx, id\)\n\}', '', text, flags=re.DOTALL)
with open('internal/service/contract.go', 'w', encoding='utf-8') as f:
    f.write(text)