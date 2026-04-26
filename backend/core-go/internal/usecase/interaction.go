package usecase

import (
	"time"

	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/infrastructure/repository/postgres"
)

// CacheInvalidator is anything that can drop cached entries that mention
// a given id. Used to evict CF similarity cache slots after an
// interaction (rating) update — the user's similarity to every other
// user, and the item's similarity to every other item, may have shifted.
type CacheInvalidator interface {
	Invalidate(id string)
}

// UpdateInteraction implements UpdateInteractionUseCase.
type UpdateInteraction struct {
	Interactions   postgres.InteractionRepository
	UserSimilarity CacheInvalidator
	ItemSimilarity CacheInvalidator
}

// NewUpdateInteraction wires the dependencies. Similarity caches are
// optional — wire them via WithCacheInvalidation when caching is enabled.
func NewUpdateInteraction(repo postgres.InteractionRepository) *UpdateInteraction {
	return &UpdateInteraction{Interactions: repo}
}

// WithCacheInvalidation hooks the similarity caches into the use case so
// they get evicted whenever a rating changes. Either argument may be nil.
func (u *UpdateInteraction) WithCacheInvalidation(userSim, itemSim CacheInvalidator) *UpdateInteraction {
	u.UserSimilarity = userSim
	u.ItemSimilarity = itemSim
	return u
}

// Execute persists a single user/item interaction. On success it
// invalidates the user's row and the item's column in the similarity
// caches so subsequent recommendation requests see fresh values.
func (u *UpdateInteraction) Execute(userID string, itemID string, data InteractionData) error {
	if err := u.Interactions.Save(entities.Interaction{
		UserID:     userID,
		ItemID:     itemID,
		Status:     data.Status,
		Rating:     data.Rating,
		IsFavorite: data.IsFavorite,
		ReviewText: data.ReviewText,
		UpdatedAt:  time.Now(),
	}); err != nil {
		return err
	}
	if u.UserSimilarity != nil {
		u.UserSimilarity.Invalidate(userID)
	}
	if u.ItemSimilarity != nil {
		u.ItemSimilarity.Invalidate(itemID)
	}
	return nil
}
