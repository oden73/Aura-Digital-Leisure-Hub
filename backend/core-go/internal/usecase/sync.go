package usecase

import (
	"fmt"

	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/infrastructure/external"
	"aura/backend/core-go/internal/infrastructure/repository/postgres"
)

// SyncExternalContent implements SyncExternalContentUseCase.
type SyncExternalContent struct {
	Adapters map[entities.ExternalService]external.Adapter
	Metadata postgres.MetadataRepository
}

// NewSyncExternalContent wires the dependencies.
func NewSyncExternalContent(
	adapters map[entities.ExternalService]external.Adapter,
	metadata postgres.MetadataRepository,
) *SyncExternalContent {
	return &SyncExternalContent{
		Adapters: adapters,
		Metadata: metadata,
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
	return item, nil
}
