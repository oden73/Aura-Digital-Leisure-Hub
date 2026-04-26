package usecase

import (
	"errors"
	"fmt"
	"log"

	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/domain/services/embeddings"
	"aura/backend/core-go/internal/infrastructure/external"
	"aura/backend/core-go/internal/infrastructure/repository/postgres"
)

// SyncExternalContent implements SyncExternalContentUseCase.
type SyncExternalContent struct {
	Adapters  map[entities.ExternalService]external.Adapter
	Metadata  postgres.MetadataRepository
	Publisher *embeddings.Publisher
}

// NewSyncExternalContent wires the dependencies.
func NewSyncExternalContent(
	adapters map[entities.ExternalService]external.Adapter,
	metadata postgres.MetadataRepository,
	publisher *embeddings.Publisher,
) *SyncExternalContent {
	return &SyncExternalContent{
		Adapters:  adapters,
		Metadata:  metadata,
		Publisher: publisher,
	}
}

// Execute fetches external metadata and stores the resulting item.
func (u *SyncExternalContent) Execute(
	externalID string,
	source entities.ExternalService,
) (entities.Item, error) {
	adapter, ok := u.Adapters[source]
	if !ok {
		return entities.Item{}, fmt.Errorf("no adapter registered for %q", source)
	}
	data, err := adapter.FetchMetadata(externalID)
	if err != nil {
		return entities.Item{}, err
	}
	item := data.ToItemMetadata()
	if err := u.Metadata.SaveItem(item); err != nil {
		return entities.Item{}, err
	}
	if u.Publisher != nil {
		if err := u.Publisher.Publish(item); err != nil && !errors.Is(err, embeddings.ErrNoText) {
			log.Printf("sync: embedding publish failed for %q: %v", item.ID, err)
		}
	}
	return item, nil
}
