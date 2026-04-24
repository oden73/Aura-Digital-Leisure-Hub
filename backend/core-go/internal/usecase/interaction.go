package usecase

import (
	"time"

	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/infrastructure/repository/postgres"
)

// UpdateInteraction implements UpdateInteractionUseCase.
type UpdateInteraction struct {
	Interactions postgres.InteractionRepository
}

// NewUpdateInteraction wires the dependencies.
func NewUpdateInteraction(repo postgres.InteractionRepository) *UpdateInteraction {
	return &UpdateInteraction{Interactions: repo}
}

// Execute persists a single user/item interaction.
func (u *UpdateInteraction) Execute(userID string, itemID string, data InteractionData) error {
	return u.Interactions.Save(entities.Interaction{
		UserID:     userID,
		ItemID:     itemID,
		Status:     data.Status,
		Rating:     data.Rating,
		IsFavorite: data.IsFavorite,
		ReviewText: data.ReviewText,
		UpdatedAt:  time.Now(),
	})
}
