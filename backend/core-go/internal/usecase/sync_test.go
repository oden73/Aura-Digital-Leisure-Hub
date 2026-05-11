package usecase

import (
	"errors"
	"testing"

	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/infrastructure/external"
)

type syncMetadataSaver struct {
	saved []entities.Item
	err   error
}

func (s *syncMetadataSaver) GetItem(_ string) (entities.Item, error) {
	return entities.Item{}, errors.New("not implemented")
}

func (s *syncMetadataSaver) SaveItem(item *entities.Item) error {
	if s.err != nil {
		return s.err
	}
	if item.ID == "" {
		item.ID = "generated-id"
	}
	s.saved = append(s.saved, *item)
	return nil
}

func (s *syncMetadataSaver) SearchByText(_ string, _ int) ([]entities.Item, error) {
	return nil, errors.New("not implemented")
}

func TestSyncExternalContent_FetchesSavesAndReturnsItem(t *testing.T) {
	meta := &syncMetadataSaver{}
	adapter := fakeExternalAdapter{full: map[string]external.ExternalData{
		"steam-1": {
			ExternalID: "steam-1",
			Source:     entities.ExternalServiceSteam,
			Title:      "Portal",
			RawData: map[string]any{
				"media_type":      string(entities.MediaTypeGame),
				"description":     "Puzzle game",
				"cover_image_url": "https://example.com/portal.jpg",
			},
		},
	}}
	uc := NewSyncExternalContent(
		map[entities.ExternalService]external.Adapter{entities.ExternalServiceSteam: adapter},
		meta,
		nil,
	)

	got, err := uc.Execute("steam-1", entities.ExternalServiceSteam)
	if err != nil {
		t.Fatalf("sync external content: %v", err)
	}
	if got.ID != "generated-id" || got.Title != "Portal" || got.MediaType != entities.MediaTypeGame {
		t.Fatalf("unexpected item: %+v", got)
	}
	if len(meta.saved) != 1 || meta.saved[0].ID != "generated-id" {
		t.Fatalf("saved items = %+v", meta.saved)
	}
}

func TestSyncExternalContent_RejectsUnknownSource(t *testing.T) {
	uc := NewSyncExternalContent(nil, &syncMetadataSaver{}, nil)

	if _, err := uc.Execute("x", entities.ExternalServiceSteam); err == nil {
		t.Fatal("expected unknown source error")
	}
}

func TestSyncExternalContent_PropagatesAdapterError(t *testing.T) {
	uc := NewSyncExternalContent(
		map[entities.ExternalService]external.Adapter{
			entities.ExternalServiceSteam: fakeExternalAdapter{},
		},
		&syncMetadataSaver{},
		nil,
	)

	if _, err := uc.Execute("missing", entities.ExternalServiceSteam); err == nil {
		t.Fatal("expected adapter error")
	}
}

func TestSyncExternalContent_PropagatesSaveError(t *testing.T) {
	meta := &syncMetadataSaver{err: errors.New("save failed")}
	adapter := fakeExternalAdapter{full: map[string]external.ExternalData{
		"steam-1": {
			ExternalID: "steam-1",
			Source:     entities.ExternalServiceSteam,
			Title:      "Portal",
			RawData:    map[string]any{"media_type": string(entities.MediaTypeGame)},
		},
	}}
	uc := NewSyncExternalContent(
		map[entities.ExternalService]external.Adapter{entities.ExternalServiceSteam: adapter},
		meta,
		nil,
	)

	if _, err := uc.Execute("steam-1", entities.ExternalServiceSteam); err == nil {
		t.Fatal("expected save error")
	}
}
