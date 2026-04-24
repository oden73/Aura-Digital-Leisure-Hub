package entities

type MediaType string

const (
	MediaTypeBook   MediaType = "book"
	MediaTypeCinema MediaType = "cinema"
	MediaTypeGame   MediaType = "game"
)

type User struct {
	ID       string
	Username string
	Email    string
}

type Item struct {
	ID          string
	Title       string
	Description string
	MediaType   MediaType
}

type ScoredItem struct {
	ItemID   string
	Score    float64
	Source   string
	Metadata map[string]any
}

type RecommendationFilters struct {
	Genres     []string
	MediaTypes []MediaType
}
