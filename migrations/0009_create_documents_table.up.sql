CREATE TABLE IF NOT EXISTS documents (
    id BIGSERIAL PRIMARY KEY,
    entity_type VARCHAR NOT NULL
                  CHECK (entity_type IN ('payment', 'gtd', 'contract', 'invoice_spec')),
    entity_id BIGINT NOT NULL,
    original_name TEXT NOT NULL,
    file_path TEXT NOT NULL,
    file_size BIGINT NOT NULL DEFAULT 0,
    mime_type VARCHAR NOT NULL,
    uploaded_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_documents_entity ON documents (entity_type, entity_id);
