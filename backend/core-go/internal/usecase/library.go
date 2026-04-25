package usecase

import "aura/backend/core-go/internal/domain/entities"

type ListLibraryUseCase interface {
	Execute(userID string) ([]entities.Interaction, error)
}

type ListLibrary struct {
	Interactions interface {
		GetUserInteractions(userID string) ([]entities.Interaction, error)
	}
}

func NewListLibrary(interactions interface {
	GetUserInteractions(userID string) ([]entities.Interaction, error)
}) *ListLibrary {
	return &ListLibrary{Interactions: interactions}
}

func (u *ListLibrary) Execute(userID string) ([]entities.Interaction, error) {
	return u.Interactions.GetUserInteractions(userID)
}

