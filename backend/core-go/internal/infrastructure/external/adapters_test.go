package external

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aura/backend/core-go/internal/domain/entities"
)

func TestExternalDataToItemMetadataMapsCommonAndTypedFields(t *testing.T) {
	item := ExternalData{
		Title: "Blade Runner",
		RawData: map[string]any{
			"media_type":      string(entities.MediaTypeCinema),
			"description":     "Replicants",
			"cover_image_url": "https://example.com/cover.jpg",
			"original_title":  "Blade Runner",
			"average_rating":  8.7,
			"release_date":    "1982",
			"genre":           "Sci-Fi",
			"themes":          "identity",
			"setting":         "future",
			"tonality":        "noir",
			"target_audience": "adult",
			"director":        "Ridley Scott",
			"cast":            "Harrison Ford",
		},
	}.ToItemMetadata()

	if item.Title != "Blade Runner" || item.MediaType != entities.MediaTypeCinema {
		t.Fatalf("unexpected item identity: %+v", item)
	}
	if item.Description != "Replicants" || item.CoverImageURL == "" || item.AverageRating != 8.7 {
		t.Fatalf("common fields not mapped: %+v", item)
	}
	if item.ReleaseDate == nil || item.ReleaseDate.Year() != 1982 {
		t.Fatalf("release date not parsed: %+v", item.ReleaseDate)
	}
	if item.Criteria.Genre != "Sci-Fi" || item.Criteria.Themes != "identity" ||
		item.Criteria.Setting != "future" || item.Criteria.Tonality != "noir" ||
		item.Criteria.TargetAudience != "adult" {
		t.Fatalf("criteria not mapped: %+v", item.Criteria)
	}
	if item.CinemaDetails == nil || item.CinemaDetails.Director != "Ridley Scott" ||
		item.CinemaDetails.Cast != "Harrison Ford" {
		t.Fatalf("cinema details not mapped: %+v", item.CinemaDetails)
	}
}

func TestExternalDataToItemMetadataMapsBookAndGameDetails(t *testing.T) {
	book := ExternalData{
		Title: "Dune",
		RawData: map[string]any{
			"media_type": string(entities.MediaTypeBook),
			"author":     "Frank Herbert",
		},
	}.ToItemMetadata()
	if book.BookDetails == nil || book.BookDetails.Author != "Frank Herbert" {
		t.Fatalf("book details not mapped: %+v", book.BookDetails)
	}

	game := ExternalData{
		Title: "Portal",
		RawData: map[string]any{
			"developer":      "Valve",
			"gameplay_genre": "Puzzle",
			"platforms":      "windows",
			"player_count":   "single-player",
			"perspective":    "first-person",
			"monetization":   "premium",
		},
	}.ToItemMetadata()
	if game.MediaType != entities.MediaTypeGame || game.GameDetails == nil {
		t.Fatalf("game defaults not applied: %+v", game)
	}
	if game.GameDetails.Developer != "Valve" || game.GameDetails.GameplayGenre != "Puzzle" ||
		game.GameDetails.Platforms != "windows" || game.GameDetails.PlayerCount != "single-player" ||
		game.GameDetails.Perspective != "first-person" || game.GameDetails.Monetization != "premium" {
		t.Fatalf("game details not mapped: %+v", game.GameDetails)
	}
}

func TestParseReleaseDateAcceptsSteamFormats(t *testing.T) {
	cases := []string{"12 Nov, 2011", "Nov 12, 2011", "12 November, 2011", "November 12, 2011", "2011", "Nov 2011"}
	for _, input := range cases {
		got, err := parseReleaseDate(input)
		if err != nil {
			t.Fatalf("%q did not parse: %v", input, err)
		}
		if got.Year() != 2011 {
			t.Fatalf("%q parsed as %v", input, got)
		}
	}
	if _, err := parseReleaseDate("not a date"); err == nil {
		t.Fatal("expected invalid date to fail")
	}
}

func TestSteamDataToExternalMapsStorePayload(t *testing.T) {
	data := steamDataToExternal("620", map[string]any{
		"name":              "Portal 2",
		"type":              "game",
		"short_description": "Puzzle game",
		"developers":        []interface{}{"Valve"},
		"genres": []interface{}{
			map[string]interface{}{"description": "Puzzle"},
			map[string]interface{}{"description": "Adventure"},
		},
		"platforms":    map[string]interface{}{"windows": true, "mac": false, "linux": true},
		"release_date": map[string]interface{}{"date": "18 Apr, 2011"},
		"categories": []interface{}{
			map[string]interface{}{"description": "Single-player"},
			map[string]interface{}{"description": "Co-op Multi-player"},
		},
	})

	if data.ExternalID != "620" || data.Source != entities.ExternalServiceSteam || data.Title != "Portal 2" {
		t.Fatalf("unexpected external data: %+v", data)
	}
	if data.RawData["developer"] != "Valve" ||
		data.RawData["genre"] != "Puzzle, Adventure" ||
		data.RawData["platforms"] != "windows, linux" ||
		data.RawData["player_count"] != "multiplayer" ||
		data.RawData["release_date"] != "18 Apr, 2011" {
		t.Fatalf("raw data not mapped: %+v", data.RawData)
	}
}

func TestSteamAdapterFetchMetadata(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/appdetails" || r.URL.Query().Get("appids") != "620" {
			t.Fatalf("unexpected request: %s", r.URL.String())
		}
		_ = json.NewEncoder(w).Encode(map[string]any{
			"620": map[string]any{
				"success": true,
				"data": map[string]any{
					"name":              "Portal 2",
					"short_description": "Puzzle game",
				},
			},
		})
	}))
	defer srv.Close()

	got, err := SteamAdapter{BaseURL: srv.URL}.FetchMetadata("620")
	if err != nil {
		t.Fatalf("fetch metadata: %v", err)
	}
	if got.Title != "Portal 2" || got.RawData["description"] != "Puzzle game" {
		t.Fatalf("unexpected data: %+v", got)
	}
}

func TestSteamAdapterFetchMetadataErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"620":{"success":false}}`))
	}))
	defer srv.Close()

	if _, err := (SteamAdapter{BaseURL: srv.URL}).FetchMetadata("620"); err == nil {
		t.Fatal("expected missing store data error")
	}
}

func TestSteamAdapterSearch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/storesearch/" || r.URL.Query().Get("term") != "portal" {
			t.Fatalf("unexpected request: %s", r.URL.String())
		}
		_, _ = w.Write([]byte(`{"total":2,"items":[{"id":620,"name":"Portal 2"},{"id":400,"name":"Portal"}]}`))
	}))
	defer srv.Close()

	got, err := (SteamAdapter{BaseURL: srv.URL}).Search("portal", 1)
	if err != nil {
		t.Fatalf("search: %v", err)
	}
	if len(got) != 1 || got[0].ExternalID != "620" || got[0].Title != "Portal 2" {
		t.Fatalf("unexpected results: %+v", got)
	}
}

func TestStubAdapters(t *testing.T) {
	if !(SteamAdapter{}).ValidateConnection() || !(TMDBAdapter{}).ValidateConnection() || !(BooksAdapter{}).ValidateConnection() {
		t.Fatal("stub adapters should validate")
	}
	if data, err := (TMDBAdapter{}).FetchMetadata("movie"); err != nil || data.Source != entities.ExternalServiceKinopoisk {
		t.Fatalf("tmdb fetch = %+v %v", data, err)
	}
	if data, err := (BooksAdapter{}).FetchMetadata("book"); err != nil || data.Source != entities.ExternalServiceGoodreads {
		t.Fatalf("books fetch = %+v %v", data, err)
	}
	if got, err := (TMDBAdapter{}).Search("q", 1); err != nil || got != nil {
		t.Fatalf("tmdb search = %+v %v", got, err)
	}
	if got, err := (BooksAdapter{}).Search("q", 1); err != nil || got != nil {
		t.Fatalf("books search = %+v %v", got, err)
	}
}

func TestSteamAdapterStoreBaseDefault(t *testing.T) {
	if got := (SteamAdapter{}).storeBase(); got != "https://store.steampowered.com" {
		t.Fatalf("store base = %q", got)
	}
	if got := (SteamAdapter{BaseURL: "http://example.com"}).storeBase(); got != "http://example.com" {
		t.Fatalf("store base override = %q", got)
	}
}

func TestParseReleaseDateReturnsUTC(t *testing.T) {
	got, err := parseReleaseDate("2011")
	if err != nil {
		t.Fatalf("parse year: %v", err)
	}
	if got.Location() != time.UTC {
		t.Fatalf("location = %v", got.Location())
	}
}
