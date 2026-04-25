package usecase

import "aura/backend/core-go/internal/domain/entities"

type GetContentUseCase interface {
	Execute(itemID string) (entities.Item, error)
}

type GetContent struct {
	Metadata interface {
		GetItem(itemID string) (entities.Item, error)
	}
}

func NewGetContent(metadata interface {
	GetItem(itemID string) (entities.Item, error)
}) *GetContent {
	return &GetContent{Metadata: metadata}
}

func (u *GetContent) Execute(itemID string) (entities.Item, error) {
	return u.Metadata.GetItem(itemID)
}

