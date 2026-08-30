import re

with open('internal/delivery/http/contract_handler.go', 'r', encoding='utf-8') as f:
    text = f.read()

# Replace the doc creation and CreateWithDocument call
old_code = """	doc := &domain.Document{
		EntityType:   "contract",
		OriginalName: handler.Filename,
		FilePath:     filePath,
		FileSize:     handler.Size,
		MimeType:     handler.Header.Get("Content-Type"),
	}

	created, err := h.service.CreateWithDocument(r.Context(), login, contract, doc)"""

new_code = """	
	pathStr := filePath
	nameStr := handler.Filename
	contract.DocumentPath = &pathStr
	contract.OriginalDocumentName = &nameStr

	created, err := h.service.Create(r.Context(), login, contract)"""

text = text.replace(old_code, new_code)

# Remove unused imports if any, but go build does that? No, it will fail if unused.
text = text.replace('io\n', 'io\n\t"path/filepath"\n') # Just to be safe, but it's probably there

with open('internal/delivery/http/contract_handler.go', 'w', encoding='utf-8') as f:
    f.write(text)