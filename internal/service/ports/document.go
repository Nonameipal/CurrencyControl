package ports

import (
	"context"

	"CurrencyControl/internal/domain"
)

type DocumentRepository interface {
	Create(ctx context.Context, d domain.Document) (domain.Document, error)
}
