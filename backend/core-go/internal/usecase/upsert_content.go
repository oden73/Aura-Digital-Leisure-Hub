package usecase

import "aura/backend/core-go/internal/domain/entities"

type UpsertContentUseCase interface {
	Execute(item entities.Item) error
}

type UpsertContent struct {
	Metadata interface {
		SaveItem(item entities.Item) error
	}
}

func NewUpsertContent(metadata interface {
	SaveItem(item entities.Item) error
}) *UpsertContent {
	return &UpsertContent{Metadata: metadata}
}

func (u *UpsertContent) Execute(item entities.Item) error {
	return u.Metadata.SaveItem(item)
}

