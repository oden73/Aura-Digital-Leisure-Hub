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
	top   []entities.Item
	err   error
}

func (f fakeMetadata) GetItem(id string) (entities.Item, error) {
	if it, ok := f.items[id]; ok {
		return it, nil
	}
	return entities.Item{}, errors.New("missing")
}

func (f fakeMetadata) SaveItem(_ entities.Item) error                        { return nil }
func (f fakeMetadata) SearchByText(_ string, _ int) ([]entities.Item, error) { return nil, nil }

func (f fakeMetadata) TopRated(limit int, _ []entities.MediaType) ([]entities.Item, error) {
	if f.err != nil {
		return nil, f.err
	}
	if limit > 0 && limit < len(f.top) {
		return f.top[:limit], nil
	}
	return f.top, nil
}

type fakeUsers struct{}

func (fakeUsers) Create(u entities.User) (entities.User, error)        { return u, nil }
func (fakeUsers) GetByID(_ string) (entities.User, error)              { return entities.User{}, nil }
func (fakeUsers) GetByEmail(_ string) (entities.User, error)           { return entities.User{}, nil }
func (fakeUsers) GetProfile(_ string) (entities.UserProfile, error)    { return entities.UserProfile{}, nil }
func (fakeUsers) LinkExternalAccount(a entities.ExternalAccount) (entities.ExternalAccount, error) {
	return a, nil
}

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
	if cs, _ := got.Metadata["cold_start"].(bool); cs {
		t.Fatalf("cold_start should be false when orchestrator returned items")
	}
}

func TestGetRecommendations_ColdStartFallbackBackfillsWithPopular(t *testing.T) {
	orch := fakeOrchestrator{res: hybrid.Result{Items: nil}}
	popular := []entities.Item{
		{ID: "p1", Title: "Pop1", AverageRating: 9.5, MediaType: entities.MediaTypeBook},
		{ID: "p2", Title: "Pop2", AverageRating: 8.7, MediaType: entities.MediaTypeBook},
		{ID: "p3", Title: "Pop3", AverageRating: 8.0, MediaType: entities.MediaTypeBook},
	}
	meta := fakeMetadata{
		items: map[string]entities.Item{
			"p1": popular[0], "p2": popular[1], "p3": popular[2],
		},
		top: popular,
	}
	uc := NewGetRecommendations(orch, fakeUsers{}, meta, filter.New().WithMetadata(meta)).
		WithPopularity(meta)

	got, err := uc.Execute("new-user", entities.RecommendationFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 3 {
		t.Fatalf("expected 3 popular items, got %d", len(got.Items))
	}
	for _, it := range got.Items {
		if it.MatchReason != matchReasonPopular {
			t.Fatalf("expected popular match_reason, got %q", it.MatchReason)
		}
	}
	if cs, _ := got.Metadata["cold_start"].(bool); !cs {
		t.Fatal("cold_start metadata flag must be true")
	}
	if pc, _ := got.Metadata["personalised_count"].(int); pc != 0 {
		t.Fatalf("expected personalised_count=0, got %d", pc)
	}
	if pf, _ := got.Metadata["popular_fallback_count"].(int); pf != 3 {
		t.Fatalf("expected popular_fallback_count=3, got %d", pf)
	}
}

func TestGetRecommendations_FallbackDoesNotDuplicatePersonalised(t *testing.T) {
	orch := fakeOrchestrator{res: hybrid.Result{
		Items: []entities.ScoredItem{
			{ItemID: "a", Score: 0.9},
		},
	}}
	meta := fakeMetadata{
		items: map[string]entities.Item{
			"a":  {ID: "a", Title: "Alpha"},
			"p1": {ID: "p1", Title: "Pop1"},
		},
		top: []entities.Item{
			{ID: "a", Title: "Alpha", AverageRating: 9.0},
			{ID: "p1", Title: "Pop1", AverageRating: 8.0},
		},
	}
	uc := NewGetRecommendations(orch, fakeUsers{}, meta, filter.New().WithMetadata(meta)).
		WithPopularity(meta)

	got, err := uc.Execute("u-1", entities.RecommendationFilters{})
	if err != nil {
		t.Fatal(err)
	}
	ids := []string{}
	for _, it := range got.Items {
		ids = append(ids, it.ItemID)
	}
	if len(ids) != 2 || ids[0] != "a" || ids[1] != "p1" {
		t.Fatalf("expected [a, p1], got %v", ids)
	}
	if cs, _ := got.Metadata["cold_start"].(bool); cs {
		t.Fatal("cold_start must be false when at least one personalised item is returned")
	}
}

func TestGetRecommendations_NoFallbackWhenPopularityNotConfigured(t *testing.T) {
	orch := fakeOrchestrator{res: hybrid.Result{Items: nil}}
	meta := fakeMetadata{}
	uc := NewGetRecommendations(orch, fakeUsers{}, meta, filter.New())

	got, err := uc.Execute("u-1", entities.RecommendationFilters{})
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Items) != 0 {
		t.Fatalf("expected empty response, got %d items", len(got.Items))
	}
	if cs, _ := got.Metadata["cold_start"].(bool); !cs {
		t.Fatal("cold_start flag should still be set")
	}
}

func TestGetRecommendations_PopularRepoErrorIsSwallowed(t *testing.T) {
	orch := fakeOrchestrator{res: hybrid.Result{Items: nil}}
	meta := fakeMetadata{err: errors.New("db boom")}
	uc := NewGetRecommendations(orch, fakeUsers{}, meta, filter.New()).WithPopularity(meta)

	got, err := uc.Execute("u-1", entities.RecommendationFilters{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Items) != 0 {
		t.Fatalf("expected zero items when popularity repo errors, got %d", len(got.Items))
	}
}
