package external

import "aura/backend/core-go/internal/domain/entities"

// ExternalData is the DTO exchanged between external adapters and the
// transformation layer (see docs/predone/diagrams/class/infrastructure_layer.puml).
type ExternalData struct {
	ExternalID string
	Source     entities.ExternalService
	Title      string
	RawData    map[string]any
}

// ToItemMetadata turns ExternalData into a domain Item. Concrete adapters are
// expected to populate the catalog-specific details via RawData.
func (d ExternalData) ToItemMetadata() entities.Item {
	return entities.Item{
		Title: d.Title,
	}
}

// Adapter is implemented by every external data source (Steam, TMDB, etc.).
type Adapter interface {
	FetchMetadata(externalID string) (ExternalData, error)
	Search(query string, limit int) ([]ExternalData, error)
	ValidateConnection() bool
}

// SteamAdapter fetches games from the Steam Web API.
type SteamAdapter struct {
	APIKey  string
	BaseURL string
}

func (SteamAdapter) FetchMetadata(externalID string) (ExternalData, error) {
	_ = externalID
	return ExternalData{Source: entities.ExternalServiceSteam}, nil
}
func (SteamAdapter) Search(query string, limit int) ([]ExternalData, error) {
	_ = query
	_ = limit
	return nil, nil
}
func (SteamAdapter) ValidateConnection() bool { return true }

// TMDBAdapter fetches cinema metadata from TMDB.
type TMDBAdapter struct {
	APIKey string
}

func (TMDBAdapter) FetchMetadata(externalID string) (ExternalData, error) {
	_ = externalID
	return ExternalData{Source: entities.ExternalServiceKinopoisk}, nil
}
func (TMDBAdapter) Search(query string, limit int) ([]ExternalData, error) {
	_ = query
	_ = limit
	return nil, nil
}
func (TMDBAdapter) ValidateConnection() bool { return true }

// BooksAdapter fetches book metadata from an ISBN/book provider.
type BooksAdapter struct{}

func (BooksAdapter) FetchMetadata(externalID string) (ExternalData, error) {
	_ = externalID
	return ExternalData{Source: entities.ExternalServiceGoodreads}, nil
}
func (BooksAdapter) Search(query string, limit int) ([]ExternalData, error) {
	_ = query
	_ = limit
	return nil, nil
}
func (BooksAdapter) ValidateConnection() bool { return true }
