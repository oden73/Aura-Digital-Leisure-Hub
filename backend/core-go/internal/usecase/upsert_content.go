package usecase

import (
	"errors"
	"log"

	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/domain/services/embeddings"
)

// UpsertContentUseCase is the API surface used by transports.
type UpsertContentUseCase interface {
	Execute(item entities.Item) error
}

// UpsertContent persists an item and pushes its textual representation to
// the AI engine for embedding. The embedding step is best-effort: failures
// are logged but never propagated, so a transient AI engine outage does not
// block catalog updates.
type UpsertContent struct {
	Metadata interface {
		SaveItem(item entities.Item) error
	}
	Publisher *embeddings.Publisher
}

// NewUpsertContent wires the metadata repository; publisher may be nil in
// tests or environments where the AI engine is not yet available.
func NewUpsertContent(
	metadata interface {
		SaveItem(item entities.Item) error
	},
	publisher *embeddings.Publisher,
) *UpsertContent {
	return &UpsertContent{Metadata: metadata, Publisher: publisher}
}

// Execute saves the item and then triggers embedding generation.
func (u *UpsertContent) Execute(item entities.Item) error {
	if err := u.Metadata.SaveItem(item); err != nil {
		return err
	}
	if u.Publisher != nil {
		if err := u.Publisher.Publish(item); err != nil && !errors.Is(err, embeddings.ErrNoText) {
			log.Printf("upsert: embedding publish failed for %q: %v", item.ID, err)
		}
	}
	return nil
}
