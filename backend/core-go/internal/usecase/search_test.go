package usecase

import (
	"errors"
	"testing"

	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/infrastructure/external"
)

type searchMetadata struct {
	local []entities.Item
	saved []entities.Item
	err   error
}

func (m *searchMetadata) GetItem(_ string) (entities.Item, error) {
	return entities.Item{}, errors.New("not implemented")
}

func (m *searchMetadata) SaveItem(item *entities.Item) error {
	if item.ID == "" {
		item.ID = "generated-id"
	}
	m.saved = append(m.saved, *item)
	return nil
}

func (m *searchMetadata) SearchByText(_ string, limit int) ([]entities.Item, error) {
	if m.err != nil {
		return nil, m.err
	}
	if limit > 0 && limit < len(m.local) {
		return m.local[:limit], nil
	}
	return m.local, nil
}

type fakeExternalAdapter struct {
	search []external.ExternalData
	full   map[string]external.ExternalData
}

func (a fakeExternalAdapter) FetchMetadata(id string) (external.ExternalData, error) {
	if data, ok := a.full[id]; ok {
		return data, nil
	}
	return external.ExternalData{}, errors.New("missing external data")
}

func (a fakeExternalAdapter) Search(_ string, limit int) ([]external.ExternalData, error) {
	if limit > 0 && limit < len(a.search) {
		return a.search[:limit], nil
	}
	return a.search, nil
}

func (fakeExternalAdapter) ValidateConnection() bool { return true }

func TestSearchContent_EnrichesLocalGameMissingCover(t *testing.T) {
	meta := &searchMetadata{local: []entities.Item{{
		ID:            "local-id",
		Title:         "Hades",
		MediaType:     entities.MediaTypeGame,
		AverageRating: 9.3,
		Criteria:      entities.BaseItemCriteria{Tonality: "fast-paced"},
	}}}
	steam := fakeExternalAdapter{
		search: []external.ExternalData{
			{ExternalID: "1145350", Title: "Hades II"},
			{ExternalID: "1145360", Title: "Hades"},
		},
		full: map[string]external.ExternalData{
			"1145360": {
				ExternalID: "1145360",
				Title:      "Hades",
				RawData: map[string]any{
					"media_type":      string(entities.MediaTypeGame),
					"cover_image_url": "https://cdn.akamai.steamstatic.com/steam/apps/1145360/library_600x900.jpg",
					"genre":           "Action",
				},
			},
		},
	}
	uc := NewSearchContent(meta).WithExternalSources(
		map[entities.ExternalService]external.Adapter{entities.ExternalServiceSteam: steam},
		nil,
	)

	got, err := uc.Execute(SearchQuery{Text: "", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Fatalf("expected one local result, got %d", len(got))
	}
	if got[0].ID != "local-id" {
		t.Fatalf("expected existing UUID to be preserved, got %q", got[0].ID)
	}
	if got[0].Title != "Hades" {
		t.Fatalf("expected local title to be preserved, got %q", got[0].Title)
	}
	if got[0].CoverImageURL == "" {
		t.Fatal("expected cover image to be enriched")
	}
	if got[0].Criteria.Tonality != "fast-paced" {
		t.Fatalf("expected existing criteria to be preserved, got %+v", got[0].Criteria)
	}
	if len(meta.saved) != 1 || meta.saved[0].ID != "local-id" {
		t.Fatalf("expected enriched local item to be saved, got %+v", meta.saved)
	}
}

func TestSearchContent_AppendsNewSteamResults(t *testing.T) {
	meta := &searchMetadata{local: []entities.Item{{
		ID:        "local-id",
		Title:     "Portal 2",
		MediaType: entities.MediaTypeGame,
	}}}
	steam := fakeExternalAdapter{
		search: []external.ExternalData{
			{ExternalID: "620", Title: "Portal 2"},
			{ExternalID: "400", Title: "Portal"},
		},
		full: map[string]external.ExternalData{
			"400": {
				ExternalID: "400",
				Title:      "Portal",
				RawData: map[string]any{
					"media_type":      string(entities.MediaTypeGame),
					"cover_image_url": "https://cdn.akamai.steamstatic.com/steam/apps/400/library_600x900.jpg",
				},
			},
		},
	}
	uc := NewSearchContent(meta).WithExternalSources(
		map[entities.ExternalService]external.Adapter{entities.ExternalServiceSteam: steam},
		nil,
	)

	got, err := uc.Execute(SearchQuery{Text: "portal", Limit: 3})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("expected local plus one new steam result, got %+v", got)
	}
	if got[1].Title != "Portal" || got[1].ID == "" || got[1].CoverImageURL == "" {
		t.Fatalf("steam result was not saved and appended correctly: %+v", got[1])
	}
}
