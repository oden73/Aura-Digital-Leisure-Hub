package postgres

import "aura/backend/core-go/internal/domain/entities"

type UserRepository interface {
	GetByID(userID string) (entities.User, error)
}

type InteractionRepository interface {
	Save(userID string, itemID string, rating int) error
}

type MetadataRepository interface {
	GetItem(itemID string) (entities.Item, error)
	Search(query string) ([]entities.Item, error)
}
