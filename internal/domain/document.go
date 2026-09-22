package domain

type DocumentFileInfo struct {
	EntityType     string `json:"entity_type"`
	EntityID       int64  `json:"entity_id"`
	DocumentNumber string `json:"document_number"`
	DocumentPath   string `json:"document_path"`
	ApprovalStatus string `json:"approval_status"`
}
