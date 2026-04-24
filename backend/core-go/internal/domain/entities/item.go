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
	Genre          string
	Setting        string
	Themes         string
	Tonality       string
	TargetAudience string
}

// Item is the generic multimedia unit persisted in the base_items table.
type Item struct {
	ID            string
	Title         string
	OriginalTitle string
	Description   string
	ReleaseDate   *time.Time
	CoverImageURL string
	AverageRating float64
	MediaType     MediaType
	Criteria      BaseItemCriteria
	BookDetails   *BookDetails
	CinemaDetails *CinemaDetails
	GameDetails   *GameDetails
}
