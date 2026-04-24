package app

import "aura/backend/core-go/internal/domain/entities"

// These no-op repositories let the skeleton boot without a real database.
// They will be replaced by PostgreSQL-backed implementations once the driver
// is wired in.

type stubUserRepo struct{}

func (stubUserRepo) GetByID(string) (entities.User, error) { return entities.User{}, nil }
func (stubUserRepo) GetProfile(string) (entities.UserProfile, error) {
	return entities.UserProfile{}, nil
}
func (stubUserRepo) LinkExternalAccount(entities.ExternalAccount) error { return nil }

type stubInteractionRepo struct{}

func (stubInteractionRepo) Save(entities.Interaction) error { return nil }
func (stubInteractionRepo) GetUserInteractions(string) ([]entities.Interaction, error) {
	return nil, nil
}

type stubMetadataRepo struct{}

func (stubMetadataRepo) GetItem(string) (entities.Item, error)             { return entities.Item{}, nil }
func (stubMetadataRepo) SaveItem(entities.Item) error                      { return nil }
func (stubMetadataRepo) SearchByText(string, int) ([]entities.Item, error) { return nil, nil }
