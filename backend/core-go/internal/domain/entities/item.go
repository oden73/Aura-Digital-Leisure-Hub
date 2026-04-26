package entities

import "time"

// MediaType enumerates the kinds of multimedia the system works with.
type MediaType string

const (
	MediaTypeBook   MediaType = "book"
	MediaTypeCinema MediaType = "cinema"
	MediaTypeGame   MediaType = "game"
)

// BaseItemCriteria captures the criteria shared by every media type.
// Mirrors the "Общие критерии" section of docs/predone/data_model.md.
type BaseItemCriteria struct {
	Genre          string `json:"genre,omitempty"`
	Setting        string `json:"setting,omitempty"`
	Themes         string `json:"themes,omitempty"`
	Tonality       string `json:"tonality,omitempty"`
	TargetAudience string `json:"target_audience,omitempty"`
}

// Item is the generic multimedia unit persisted in the base_items table.
//
// release_date is intentionally encoded without `omitempty`: the OpenAPI
// contract marks it as nullable so clients can read it without checking
// for presence (it serialises to null when nil). Media-specific details
// are emitted only for the matching media_type and remain nil otherwise,
// hence `omitempty` keeps responses compact.
type Item struct {
	ID            string           `json:"id"`
	Title         string           `json:"title"`
	OriginalTitle string           `json:"original_title,omitempty"`
	Description   string           `json:"description,omitempty"`
	ReleaseDate   *time.Time       `json:"release_date"`
	CoverImageURL string           `json:"cover_image_url,omitempty"`
	AverageRating float64          `json:"average_rating"`
	MediaType     MediaType        `json:"media_type"`
	Criteria      BaseItemCriteria `json:"criteria"`
	BookDetails   *BookDetails     `json:"book_details,omitempty"`
	CinemaDetails *CinemaDetails   `json:"cinema_details,omitempty"`
	GameDetails   *GameDetails     `json:"game_details,omitempty"`
}
