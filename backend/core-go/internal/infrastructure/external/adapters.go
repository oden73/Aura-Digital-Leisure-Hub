package external

import "aura/backend/core-go/internal/domain/entities"

type Adapter interface {
	FetchMetadata(externalID string) (entities.Item, error)
}

type SteamAdapter struct{}
type TMDBAdapter struct{}
type BooksAdapter struct{}

func (SteamAdapter) FetchMetadata(_ string) (entities.Item, error) {
	return entities.Item{}, nil
}

func (TMDBAdapter) FetchMetadata(_ string) (entities.Item, error) {
	return entities.Item{}, nil
}

func (BooksAdapter) FetchMetadata(_ string) (entities.Item, error) {
	return entities.Item{}, nil
}
