package usecase

import (
	"aura/backend/core-go/internal/domain/entities"
	repopostgres "aura/backend/core-go/internal/infrastructure/repository/postgres"
)

type LibraryItem struct {
	Interaction entities.Interaction `json:"interaction"`
	Item        entities.Item        `json:"item"`
}

type ListLibraryItemsUseCase interface {
	Execute(userID string, limit int) ([]LibraryItem, error)
}

type ListLibraryItems struct {
	Interactions interface {
		GetUserLibraryItems(userID string, limit int) ([]repopostgres.LibraryItem, error)
	}
}

func NewListLibraryItems(interactions interface {
	GetUserLibraryItems(userID string, limit int) ([]repopostgres.LibraryItem, error)
}) *ListLibraryItems {
	return &ListLibraryItems{Interactions: interactions}
}

func (u *ListLibraryItems) Execute(userID string, limit int) ([]LibraryItem, error) {
	raw, err := u.Interactions.GetUserLibraryItems(userID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]LibraryItem, 0, len(raw))
	for _, r := range raw {
		out = append(out, LibraryItem{Interaction: r.Interaction, Item: r.Item})
	}
	return out, nil
}

