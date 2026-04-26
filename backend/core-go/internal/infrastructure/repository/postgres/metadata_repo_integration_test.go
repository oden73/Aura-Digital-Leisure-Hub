//go:build integration

package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"aura/backend/core-go/internal/domain/entities"
)

func TestMetadataRepo_SaveItemBook_RoundTripWithDetails(t *testing.T) {
	pool := mustGetPool(t)
	repo := NewMetadataRepo(pool)

	released := time.Date(1979, 3, 22, 0, 0, 0, 0, time.UTC)
	in := entities.Item{
		Title:         "Roadside Picnic",
		OriginalTitle: "Пикник на обочине",
		Description:   "A novel about an alien Visitation Zone.",
		ReleaseDate:   &released,
		AverageRating: 8.5,
		MediaType:     entities.MediaTypeBook,
		Criteria: entities.BaseItemCriteria{
			Genre:    "sci-fi",
			Setting:  "post-contact zone",
			Themes:   "first contact, ambiguity",
			Tonality: "bleak",
		},
		BookDetails: &entities.BookDetails{
			Author:        "Strugatsky",
			Publisher:     "Macmillan",
			LiteraryForm:  "novel",
			VolumeFormat:  "hardcover",
			NarrativeType: "first-person",
			PageCount:     224,
		},
	}

	id := mustSaveItem(t, repo, in)

	out, err := repo.GetItem(id)
	if err != nil {
		t.Fatalf("get item: %v", err)
	}
	if out.Title != in.Title || out.MediaType != entities.MediaTypeBook {
		t.Fatalf("base mismatch: %+v", out)
	}
	if out.AverageRating != in.AverageRating {
		t.Fatalf("avg rating round-trip: %v", out.AverageRating)
	}
	if out.ReleaseDate == nil || !out.ReleaseDate.Equal(released) {
		t.Fatalf("release date round-trip: %v", out.ReleaseDate)
	}
	if out.Criteria.Genre != "sci-fi" || out.Criteria.Themes != "first contact, ambiguity" {
		t.Fatalf("criteria round-trip: %+v", out.Criteria)
	}
	if out.BookDetails == nil ||
		out.BookDetails.Author != "Strugatsky" ||
		out.BookDetails.PageCount != 224 {
		t.Fatalf("book details round-trip: %+v", out.BookDetails)
	}
	if out.CinemaDetails != nil || out.GameDetails != nil {
		t.Fatalf("only book details should be populated, got %+v / %+v",
			out.CinemaDetails, out.GameDetails)
	}
}

func TestMetadataRepo_SaveItemCinema_PersistsCinemaDetails(t *testing.T) {
	pool := mustGetPool(t)
	repo := NewMetadataRepo(pool)

	in := entities.Item{
		Title:     "Stalker",
		MediaType: entities.MediaTypeCinema,
		CinemaDetails: &entities.CinemaDetails{
			Director:     "Tarkovsky",
			Cast:         "Kaidanovsky, Solonitsyn",
			Format:       "feature",
			VisualStyle:  "long takes",
			DurationMins: 161,
		},
	}
	id := mustSaveItem(t, repo, in)

	out, err := repo.GetItem(id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if out.CinemaDetails == nil || out.CinemaDetails.Director != "Tarkovsky" {
		t.Fatalf("cinema details lost: %+v", out.CinemaDetails)
	}
	if out.BookDetails != nil {
		t.Fatalf("book details should be nil, got %+v", out.BookDetails)
	}
}

func TestMetadataRepo_SaveItemGame_UpsertsBaseAndDetails(t *testing.T) {
	pool := mustGetPool(t)
	repo := NewMetadataRepo(pool)

	in := entities.Item{
		Title:     "Disco Elysium",
		MediaType: entities.MediaTypeGame,
		GameDetails: &entities.GameDetails{
			Developer:      "ZA/UM",
			GameplayGenre:  "isometric rpg",
			Platforms:      "pc, mac, console",
			Perspective:    "isometric",
			WorldStructure: "open hub",
		},
	}
	id := mustSaveItem(t, repo, in)

	// Update with new attributes; SaveItem must upsert.
	in.ID = id
	in.Title = "Disco Elysium: Final Cut"
	in.AverageRating = 9.4
	in.GameDetails.Platforms = "pc, mac, console, switch"
	if err := repo.SaveItem(&in); err != nil {
		t.Fatalf("upsert: %v", err)
	}

	got, err := repo.GetItem(id)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Title != "Disco Elysium: Final Cut" {
		t.Fatalf("upsert did not update title: %q", got.Title)
	}
	if got.AverageRating != 9.4 {
		t.Fatalf("upsert did not update avg rating: %v", got.AverageRating)
	}
	if got.GameDetails == nil || got.GameDetails.Platforms != "pc, mac, console, switch" {
		t.Fatalf("upsert did not refresh details: %+v", got.GameDetails)
	}
}

func TestMetadataRepo_GetItem_NotFound(t *testing.T) {
	pool := mustGetPool(t)
	repo := NewMetadataRepo(pool)

	_, err := repo.GetItem("00000000-0000-0000-0000-000000000000")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestMetadataRepo_TopRated_OrdersByAverageDesc(t *testing.T) {
	pool := mustGetPool(t)
	repo := NewMetadataRepo(pool)

	mustSaveItem(t, repo, entities.Item{Title: "low", AverageRating: 6.5, MediaType: entities.MediaTypeBook})
	mustSaveItem(t, repo, entities.Item{Title: "mid", AverageRating: 7.5, MediaType: entities.MediaTypeBook})
	mustSaveItem(t, repo, entities.Item{Title: "high", AverageRating: 9.0, MediaType: entities.MediaTypeBook})
	mustSaveItem(t, repo, entities.Item{Title: "game", AverageRating: 8.8, MediaType: entities.MediaTypeGame})

	all, err := repo.TopRated(10, nil)
	if err != nil {
		t.Fatalf("top rated: %v", err)
	}
	if len(all) != 4 {
		t.Fatalf("expected 4 rows, got %d", len(all))
	}
	if all[0].Title != "high" || all[1].Title != "game" || all[2].Title != "mid" || all[3].Title != "low" {
		t.Fatalf("unexpected order: %v", titles(all))
	}

	booksOnly, err := repo.TopRated(10, []entities.MediaType{entities.MediaTypeBook})
	if err != nil {
		t.Fatalf("top rated books: %v", err)
	}
	for _, it := range booksOnly {
		if it.MediaType != entities.MediaTypeBook {
			t.Fatalf("got %q in books-only result", it.MediaType)
		}
	}
	if len(booksOnly) != 3 {
		t.Fatalf("expected 3 books, got %d", len(booksOnly))
	}
}

func TestMetadataRepo_IterateAll_VisitsEveryItemExactlyOnce(t *testing.T) {
	pool := mustGetPool(t)
	repo := NewMetadataRepo(pool)

	const N = 7
	want := make(map[string]bool, N)
	for i := 0; i < N; i++ {
		id := mustSaveItem(t, repo, entities.Item{
			Title:     "iter-" + string(rune('a'+i)),
			MediaType: entities.MediaTypeBook,
		})
		want[id] = true
	}

	got := map[string]int{}
	err := repo.IterateAll(context.Background(), 3, func(it entities.Item) error {
		got[it.ID]++
		return nil
	})
	if err != nil {
		t.Fatalf("iterate: %v", err)
	}
	if len(got) != N {
		t.Fatalf("expected to visit %d items, got %d", N, len(got))
	}
	for id, n := range got {
		if n != 1 {
			t.Fatalf("item %s visited %d times", id, n)
		}
		if !want[id] {
			t.Fatalf("unexpected id visited: %s", id)
		}
	}
}

func TestMetadataRepo_SearchByText_ILike(t *testing.T) {
	pool := mustGetPool(t)
	repo := NewMetadataRepo(pool)

	mustSaveItem(t, repo, entities.Item{Title: "Solaris", Description: "ocean planet", MediaType: entities.MediaTypeBook})
	mustSaveItem(t, repo, entities.Item{Title: "Roadside Picnic", Description: "Visitation Zone", MediaType: entities.MediaTypeBook})
	mustSaveItem(t, repo, entities.Item{Title: "Stalker", MediaType: entities.MediaTypeCinema})

	hits, err := repo.SearchByText("solar", 10)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(hits) != 1 || hits[0].Title != "Solaris" {
		t.Fatalf("unexpected hits: %v", titles(hits))
	}
}

func titles(items []entities.Item) []string {
	out := make([]string, len(items))
	for i, it := range items {
		out[i] = it.Title
	}
	return out
}
