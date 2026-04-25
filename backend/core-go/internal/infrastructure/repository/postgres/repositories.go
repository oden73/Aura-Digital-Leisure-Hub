package postgres

import "aura/backend/core-go/internal/domain/entities"

// UserRepository handles persistence of users and their external links.
type UserRepository interface {
	Create(user entities.User) (entities.User, error)
	GetByID(userID string) (entities.User, error)
	GetByEmail(email string) (entities.User, error)
	GetProfile(userID string) (entities.UserProfile, error)
	LinkExternalAccount(account entities.ExternalAccount) error
}

// InteractionRepository persists the Rui matrix.
type InteractionRepository interface {
	Save(interaction entities.Interaction) error
	GetUserInteractions(userID string) ([]entities.Interaction, error)
}

// MetadataRepository stores and retrieves catalog items and their details.
type MetadataRepository interface {
	GetItem(itemID string) (entities.Item, error)
	SaveItem(item entities.Item) error
	SearchByText(query string, limit int) ([]entities.Item, error)
}

// UserStatisticsRepository aggregates per-user rating statistics used by
// the CF predictors (see internal/domain/services/cf).
type UserStatisticsRepository interface {
	GetMeanRating(userID string) (float64, error)
	GetVariance(userID string) (float64, error)
}
