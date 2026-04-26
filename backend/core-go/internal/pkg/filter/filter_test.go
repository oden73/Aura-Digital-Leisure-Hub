package filter

import (
	"testing"
	"time"

	"aura/backend/core-go/internal/domain/entities"
)

type fakeMeta struct {
	items map[string]entities.Item
}

func (f fakeMeta) GetItem(id string) (entities.Item, error) {
	if it, ok := f.items[id]; ok {
		return it, nil
	}
	return entities.Item{}, errNotFound
}

var errNotFound = &simpleErr{"not found"}

type simpleErr struct{ s string }

func (e *simpleErr) Error() string { return e.s }

func TestApply_PassThroughWhenNoFilters(t *testing.T) {
	s := New().WithMetadata(fakeMeta{items: map[string]entities.Item{
		"a": {Criteria: entities.BaseItemCriteria{Genre: "Drama"}},
	}})
	in := []entities.ScoredItem{{ItemID: "a", Score: 1}}
	got := s.Apply(in, entities.RecommendationFilters{})
	if len(got) != 1 {
		t.Fatalf("expected pass-through, got %#v", got)
	}
}

func TestApply_FiltersByGenreCaseInsensitive(t *testing.T) {
	s := New().WithMetadata(fakeMeta{items: map[string]entities.Item{
		"a": {Criteria: entities.BaseItemCriteria{Genre: "Drama"}},
		"b": {Criteria: entities.BaseItemCriteria{Genre: "Sci-Fi"}},
	}})
	in := []entities.ScoredItem{{ItemID: "a"}, {ItemID: "b"}}
	got := s.Apply(in, entities.RecommendationFilters{Genres: []string{"drama"}})
	if len(got) != 1 || got[0].ItemID != "a" {
		t.Fatalf("only drama should pass, got %#v", got)
	}
}

func TestApply_FiltersByMediaType(t *testing.T) {
	s := New().WithMetadata(fakeMeta{items: map[string]entities.Item{
		"book":   {MediaType: entities.MediaTypeBook},
		"game":   {MediaType: entities.MediaTypeGame},
		"cinema": {MediaType: entities.MediaTypeCinema},
	}})
	in := []entities.ScoredItem{{ItemID: "book"}, {ItemID: "game"}, {ItemID: "cinema"}}
	got := s.Apply(in, entities.RecommendationFilters{
		MediaTypes: []entities.MediaType{entities.MediaTypeBook, entities.MediaTypeGame},
	})
	if len(got) != 2 {
		t.Fatalf("expected 2 items, got %#v", got)
	}
}

func TestApply_FiltersByReleasePeriod(t *testing.T) {
	t1 := time.Date(2010, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	s := New().WithMetadata(fakeMeta{items: map[string]entities.Item{
		"old":    {ReleaseDate: &t1},
		"middle": {ReleaseDate: &t2},
		"new":    {ReleaseDate: &t3},
		"undated": {},
	}})
	from := time.Date(2015, 1, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)
	got := s.Apply(
		[]entities.ScoredItem{{ItemID: "old"}, {ItemID: "middle"}, {ItemID: "new"}, {ItemID: "undated"}},
		entities.RecommendationFilters{ReleasePeriod: entities.DateRange{From: &from, To: &to}},
	)
	if len(got) != 1 || got[0].ItemID != "middle" {
		t.Fatalf("only middle should pass, got %#v", got)
	}
}

func TestApply_FiltersByRatingRange(t *testing.T) {
	s := New().WithMetadata(fakeMeta{items: map[string]entities.Item{
		"low":  {AverageRating: 3},
		"mid":  {AverageRating: 6},
		"high": {AverageRating: 9},
	}})
	got := s.Apply(
		[]entities.ScoredItem{{ItemID: "low"}, {ItemID: "mid"}, {ItemID: "high"}},
		entities.RecommendationFilters{RatingRange: entities.RatingRange{Min: 5, Max: 8}},
	)
	if len(got) != 1 || got[0].ItemID != "mid" {
		t.Fatalf("only mid should pass, got %#v", got)
	}
}

func TestApply_DropsItemsWithoutMetadata(t *testing.T) {
	s := New().WithMetadata(fakeMeta{items: map[string]entities.Item{
		"known": {Criteria: entities.BaseItemCriteria{Genre: "Drama"}},
	}})
	got := s.Apply(
		[]entities.ScoredItem{{ItemID: "known"}, {ItemID: "ghost"}},
		entities.RecommendationFilters{Genres: []string{"drama"}},
	)
	if len(got) != 1 || got[0].ItemID != "known" {
		t.Fatalf("expected only known, got %#v", got)
	}
}
