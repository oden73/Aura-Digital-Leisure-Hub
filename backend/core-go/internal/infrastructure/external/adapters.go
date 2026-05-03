package external

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"aura/backend/core-go/internal/domain/entities"
)

// ExternalData is the DTO exchanged between external adapters and the
// transformation layer (see docs/predone/diagrams/class/infrastructure_layer.puml).
//
// RawData stores source-specific fields under well-known string keys so that
// ToItemMetadata can reconstruct a fully-populated Item without losing any
// data that adapters have fetched.
type ExternalData struct {
	ExternalID string
	Source     entities.ExternalService
	Title      string
	RawData    map[string]any
}

// ToItemMetadata converts ExternalData into a domain Item, mapping every
// well-known RawData key to the corresponding Item field.
//
// Well-known keys (all optional):
//
//	media_type        string  — entities.MediaType value (default: "game" for Steam)
//	description       string
//	cover_image_url   string
//	original_title    string
//	average_rating    float64
//	release_date      string  — parsed with parseReleaseDate
//	genre             string  — Criteria.Genre
//	themes            string  — Criteria.Themes
//	setting           string  — Criteria.Setting
//	tonality          string  — Criteria.Tonality
//	target_audience   string  — Criteria.TargetAudience
//	developer         string  — GameDetails.Developer
//	gameplay_genre    string  — GameDetails.GameplayGenre
//	platforms         string  — GameDetails.Platforms
//	player_count      string  — GameDetails.PlayerCount
//	perspective       string  — GameDetails.Perspective
//	monetization      string  — GameDetails.Monetization
//	director          string  — CinemaDetails.Director
//	cast              string  — CinemaDetails.Cast
//	author            string  — BookDetails.Author
func (d ExternalData) ToItemMetadata() entities.Item {
	item := entities.Item{
		Title:     d.Title,
		MediaType: entities.MediaTypeGame,
	}

	if v, _ := d.RawData["media_type"].(string); v != "" {
		item.MediaType = entities.MediaType(v)
	}
	if v, _ := d.RawData["description"].(string); v != "" {
		item.Description = v
	}
	if v, _ := d.RawData["cover_image_url"].(string); v != "" {
		item.CoverImageURL = v
	}
	if v, _ := d.RawData["original_title"].(string); v != "" {
		item.OriginalTitle = v
	}
	if v, ok := d.RawData["average_rating"].(float64); ok {
		item.AverageRating = v
	}
	if v, _ := d.RawData["release_date"].(string); v != "" {
		if t, err := parseReleaseDate(v); err == nil {
			item.ReleaseDate = &t
		}
	}

	item.Criteria = entities.BaseItemCriteria{}
	if v, _ := d.RawData["genre"].(string); v != "" {
		item.Criteria.Genre = v
	}
	if v, _ := d.RawData["themes"].(string); v != "" {
		item.Criteria.Themes = v
	}
	if v, _ := d.RawData["setting"].(string); v != "" {
		item.Criteria.Setting = v
	}
	if v, _ := d.RawData["tonality"].(string); v != "" {
		item.Criteria.Tonality = v
	}
	if v, _ := d.RawData["target_audience"].(string); v != "" {
		item.Criteria.TargetAudience = v
	}

	switch item.MediaType {
	case entities.MediaTypeGame:
		gd := &entities.GameDetails{}
		if v, _ := d.RawData["developer"].(string); v != "" {
			gd.Developer = v
		}
		if v, _ := d.RawData["gameplay_genre"].(string); v != "" {
			gd.GameplayGenre = v
		}
		if v, _ := d.RawData["platforms"].(string); v != "" {
			gd.Platforms = v
		}
		if v, _ := d.RawData["player_count"].(string); v != "" {
			gd.PlayerCount = v
		}
		if v, _ := d.RawData["perspective"].(string); v != "" {
			gd.Perspective = v
		}
		if v, _ := d.RawData["monetization"].(string); v != "" {
			gd.Monetization = v
		}
		item.GameDetails = gd
	case entities.MediaTypeCinema:
		cd := &entities.CinemaDetails{}
		if v, _ := d.RawData["director"].(string); v != "" {
			cd.Director = v
		}
		if v, _ := d.RawData["cast"].(string); v != "" {
			cd.Cast = v
		}
		item.CinemaDetails = cd
	case entities.MediaTypeBook:
		bd := &entities.BookDetails{}
		if v, _ := d.RawData["author"].(string); v != "" {
			bd.Author = v
		}
		item.BookDetails = bd
	}

	return item
}

// Adapter is implemented by every external data source (Steam, TMDB, etc.).
type Adapter interface {
	FetchMetadata(externalID string) (ExternalData, error)
	Search(query string, limit int) ([]ExternalData, error)
	ValidateConnection() bool
}

// ---------------------------------------------------------------------------
// SteamAdapter
// ---------------------------------------------------------------------------

// SteamAdapter fetches game metadata from the Steam Web API and Store API.
type SteamAdapter struct {
	APIKey  string
	BaseURL string // override in tests; defaults to https://store.steampowered.com
}

func (a SteamAdapter) storeBase() string {
	if a.BaseURL != "" {
		return a.BaseURL
	}
	return "https://store.steampowered.com"
}

// FetchMetadata calls the Steam Store appdetails endpoint and returns
// an ExternalData populated with all available game fields.
func (a SteamAdapter) FetchMetadata(externalID string) (ExternalData, error) {
	target := fmt.Sprintf(
		"%s/api/appdetails?appids=%s&cc=us&l=en",
		a.storeBase(), url.QueryEscape(externalID),
	)
	resp, err := http.Get(target) //nolint:noctx
	if err != nil {
		return ExternalData{}, fmt.Errorf("steam appdetails request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ExternalData{}, fmt.Errorf("steam appdetails read: %w", err)
	}

	// Response shape: {"<appid>": {"success": bool, "data": {...}}}
	var outer map[string]struct {
		Success bool           `json:"success"`
		Data    map[string]any `json:"data"`
	}
	if err := json.Unmarshal(body, &outer); err != nil {
		return ExternalData{}, fmt.Errorf("steam appdetails decode: %w", err)
	}
	entry, ok := outer[externalID]
	if !ok || !entry.Success || entry.Data == nil {
		return ExternalData{}, fmt.Errorf("steam: no data for appid %s", externalID)
	}

	return steamDataToExternal(externalID, entry.Data), nil
}

// Search queries the Steam Store search endpoint and returns up to limit results.
// Each result contains only the data available from the search index (title,
// cover image); call FetchMetadata for full details.
func (a SteamAdapter) Search(query string, limit int) ([]ExternalData, error) {
	if limit <= 0 {
		limit = 10
	}
	target := fmt.Sprintf(
		"https://store.steampowered.com/api/storesearch/?term=%s&l=en&cc=us",
		url.QueryEscape(query),
	)
	resp, err := http.Get(target) //nolint:noctx
	if err != nil {
		return nil, fmt.Errorf("steam search request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("steam search read: %w", err)
	}

	var result struct {
		Total int `json:"total"`
		Items []struct {
			ID    int    `json:"id"`
			Name  string `json:"name"`
			Image string `json:"tiny_image"`
		} `json:"items"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("steam search decode: %w", err)
	}

	out := make([]ExternalData, 0, min(limit, len(result.Items)))
	for i, it := range result.Items {
		if i >= limit {
			break
		}
		out = append(out, ExternalData{
			ExternalID: fmt.Sprintf("%d", it.ID),
			Source:     entities.ExternalServiceSteam,
			Title:      it.Name,
			RawData: map[string]any{
				"cover_image_url": it.Image,
				"media_type":      string(entities.MediaTypeGame),
			},
		})
	}
	return out, nil
}

func (SteamAdapter) ValidateConnection() bool { return true }

// steamDataToExternal maps a Steam appdetails "data" blob to ExternalData.
func steamDataToExternal(appID string, data map[string]any) ExternalData {
	title, _ := data["name"].(string)
	raw := map[string]any{
		"media_type": string(entities.MediaTypeGame),
	}

	if v, _ := data["short_description"].(string); v != "" {
		raw["description"] = v
	}
	if v, _ := data["header_image"].(string); v != "" {
		raw["cover_image_url"] = v
	}

	// developers: []interface{} of strings
	if devs, ok := data["developers"].([]interface{}); ok && len(devs) > 0 {
		names := make([]string, 0, len(devs))
		for _, d := range devs {
			if s, ok := d.(string); ok {
				names = append(names, s)
			}
		}
		if len(names) > 0 {
			raw["developer"] = strings.Join(names, ", ")
		}
	}

	// genres: [{"id": "1", "description": "Action"}, ...]
	if genres, ok := data["genres"].([]interface{}); ok && len(genres) > 0 {
		gs := make([]string, 0, len(genres))
		for _, g := range genres {
			if gm, ok := g.(map[string]interface{}); ok {
				if desc, ok := gm["description"].(string); ok {
					gs = append(gs, desc)
				}
			}
		}
		if len(gs) > 0 {
			joined := strings.Join(gs, ", ")
			raw["genre"] = joined
			raw["gameplay_genre"] = joined
		}
	}

	// platforms: {"windows": true, "mac": false, "linux": false}
	if platforms, ok := data["platforms"].(map[string]interface{}); ok {
		parts := make([]string, 0, 3)
		for _, p := range []string{"windows", "mac", "linux"} {
			if b, ok := platforms[p].(bool); ok && b {
				parts = append(parts, p)
			}
		}
		if len(parts) > 0 {
			raw["platforms"] = strings.Join(parts, ", ")
		}
	}

	// release_date: {"coming_soon": false, "date": "12 Nov, 2011"}
	if rd, ok := data["release_date"].(map[string]interface{}); ok {
		if dateStr, _ := rd["date"].(string); dateStr != "" {
			raw["release_date"] = dateStr
		}
	}

	// categories: [{"id": 1, "description": "Multi-player"}, ...]
	if cats, ok := data["categories"].([]interface{}); ok {
		multiplayer := false
		for _, c := range cats {
			if cm, ok := c.(map[string]interface{}); ok {
				if desc, ok := cm["description"].(string); ok &&
					strings.Contains(strings.ToLower(desc), "multi") {
					multiplayer = true
				}
			}
		}
		if multiplayer {
			raw["player_count"] = "multiplayer"
		} else {
			raw["player_count"] = "single-player"
		}
	}

	return ExternalData{
		ExternalID: appID,
		Source:     entities.ExternalServiceSteam,
		Title:      title,
		RawData:    raw,
	}
}

// parseReleaseDate parses the date formats Steam uses in appdetails responses.
func parseReleaseDate(s string) (time.Time, error) {
	for _, f := range []string{
		"2 Jan, 2006",
		"Jan 2, 2006",
		"2 January, 2006",
		"January 2, 2006",
		"2006",
		"Jan 2006",
	} {
		if t, err := time.Parse(f, s); err == nil {
			return t, nil
		}
	}
	return time.Time{}, fmt.Errorf("cannot parse Steam date %q", s)
}

// ---------------------------------------------------------------------------
// TMDBAdapter — stub
// ---------------------------------------------------------------------------

// TMDBAdapter fetches cinema metadata from TMDB (stub).
type TMDBAdapter struct {
	APIKey string
}

func (TMDBAdapter) FetchMetadata(externalID string) (ExternalData, error) {
	_ = externalID
	return ExternalData{Source: entities.ExternalServiceKinopoisk}, nil
}
func (TMDBAdapter) Search(query string, limit int) ([]ExternalData, error) {
	_, _ = query, limit
	return nil, nil
}
func (TMDBAdapter) ValidateConnection() bool { return true }

// ---------------------------------------------------------------------------
// BooksAdapter — stub
// ---------------------------------------------------------------------------

// BooksAdapter fetches book metadata from an ISBN/book provider (stub).
type BooksAdapter struct{}

func (BooksAdapter) FetchMetadata(externalID string) (ExternalData, error) {
	_ = externalID
	return ExternalData{Source: entities.ExternalServiceGoodreads}, nil
}
func (BooksAdapter) Search(query string, limit int) ([]ExternalData, error) {
	_, _ = query, limit
	return nil, nil
}
func (BooksAdapter) ValidateConnection() bool { return true }
