package repository

import (
	"context"

	"CurrencyControl/internal/domain"
	"CurrencyControl/internal/service/ports"

	"github.com/jackc/pgx/v5/pgxpool"
)

type documentRepo struct {
	db *pgxpool.Pool
}

func NewDocumentRepository(db *pgxpool.Pool) ports.DocumentRepository {
	return &documentRepo{db: db}
}

func (r *documentRepo) Create(ctx context.Context, d domain.Document) (domain.Document, error) {
	query := `
		INSERT INTO documents (entity_type, entity_id, original_name, file_path, file_size, mime_type)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, entity_type, entity_id, original_name, file_path, file_size, mime_type, uploaded_at`

	var result domain.Document
	err := r.db.QueryRow(ctx, query,
		d.EntityType,
		d.EntityID,
		d.OriginalName,
		d.FilePath,
		d.FileSize,
		d.MimeType,
	).Scan(
		&result.ID,
		&result.EntityType,
		&result.EntityID,
		&result.OriginalName,
		&result.FilePath,
		&result.FileSize,
		&result.MimeType,
		&result.UploadedAt,
	)

	if err != nil {
		return domain.Document{}, err
	}

	return result, nil
}
