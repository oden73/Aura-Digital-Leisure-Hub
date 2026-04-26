package usecase

import (
	"errors"
	"testing"

	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/domain/services/hybrid"
	"aura/backend/core-go/internal/pkg/filter"
)

type fakeOrchestrator struct {
	res hybrid.Result
	err error
}

func (f fakeOrchestrator) GetHybridRecommendations(
	_ string,
	_ int,
	_ entities.RecommendationFilters,
) (hybrid.Result, error) {
	return f.res, f.err
}

type fakeMetadata struct {
	items map[string]entities.Item
}

func (f fakeMetadata) GetItem(id string) (entities.Item, error) {
	if it, ok := f.items[id]; ok {
		return it, nil
	}
	return entities.Item{}, errors.New("missing")
}

func (f fakeMetadata) SaveItem(_ entities.Item) error                        { return nil }
func (f fakeMetadata) SearchByText(_ string, _ int) ([]entities.Item, error) { return nil, nil }

type fakeUsers struct{}

func (fakeUsers) Create(u entities.User) (entities.User, error)            { return u, nil }
func (fakeUsers) GetByID(_ string) (entities.User, error)                  { return entities.User{}, nil }
func (fakeUsers) GetByEmail(_ string) (entities.User, error)               { return entities.User{}, nil }
func (fakeUsers) GetProfile(_ string) (entities.UserProfile, error)        { return entities.UserProfile{}, nil }
func (fakeUsers) LinkExternalAccount(_ entities.ExternalAccount) error     { return nil }

func TestGetRecommendations_EnrichesAndPropagatesReasoning(t *testing.T) {
	orch := fakeOrchestrator{res: hybrid.Result{
		Items: []entities.ScoredItem{
			{ItemID: "a", Score: 0.9, Metadata: map[string]any{"match_reason": "you liked X"}},
			{ItemID: "b", Score: 0.7},
		},
		Reasoning: "narrative",
	}}
	meta := fakeMetadata{items: map[string]entities.Item{
		"a": {ID: "a", Title: "Alpha"},
		"b": {ID: "b", Title: "Beta"},
	}}
	uc := NewGetRecommendations(orch, fakeUsers{}, meta, filter.New())

	got, err := uc.Execute("u-1", entities.RecommendationFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Reasoning != "narrative" {
		t.Fatalf("reasoning not propagated: %q", got.Reasoning)
	}
	if len(got.Items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(got.Items))
	}
	if got.Items[0].ItemID != "a" || got.Items[0].Title != "Alpha" || got.Items[0].MatchReason != "you liked X" {
		t.Fatalf("item a not enriched: %#v", got.Items[0])
	}
	if got.Items[1].Title != "Beta" || got.Items[1].MatchReason != "" {
		t.Fatalf("item b unexpectedly modified: %#v", got.Items[1])
	}
	if uid, _ := got.Metadata["user_id"].(string); uid != "u-1" {
		t.Fatalf("user_id not in response metadata: %#v", got.Metadata)
	}
}
