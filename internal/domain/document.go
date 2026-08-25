package domain

import "time"

type Document struct {
	ID           int64     `db:"id"`
	EntityType   string    `db:"entity_type"`
	EntityID     int64     `db:"entity_id"`
	OriginalName string    `db:"original_name"`
	FilePath     string    `db:"file_path"`
	FileSize     int64     `db:"file_size"`
	MimeType     string    `db:"mime_type"`
	UploadedAt   time.Time `db:"uploaded_at"`
}
