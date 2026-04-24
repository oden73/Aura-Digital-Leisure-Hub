package usecase

import (
	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/infrastructure/repository/postgres"
)

// SearchContent implements SearchContentUseCase.
type SearchContent struct {
	Metadata postgres.MetadataRepository
}

// NewSearchContent wires the dependencies.
func NewSearchContent(metadata postgres.MetadataRepository) *SearchContent {
	return &SearchContent{Metadata: metadata}
}

// Execute performs a text search against the metadata repository.
func (u *SearchContent) Execute(query SearchQuery) ([]entities.Item, error) {
	limit := query.Limit
	if limit <= 0 {
		limit = 20
	}
	return u.Metadata.SearchByText(query.Text, limit)
}
