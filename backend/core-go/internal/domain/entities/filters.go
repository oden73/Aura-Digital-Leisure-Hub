package entities

import "time"

// DateRange is an inclusive window on release dates.
type DateRange struct {
	From *time.Time `json:"from,omitempty"`
	To   *time.Time `json:"to,omitempty"`
}

// RatingRange is an inclusive window on the 1-10 rating scale.
type RatingRange struct {
	Min int `json:"min,omitempty"`
	Max int `json:"max,omitempty"`
}

// RecommendationFilters describes filters applied to a recommendation response.
type RecommendationFilters struct {
	Genres        []string    `json:"genres,omitempty"`
	MediaTypes    []MediaType `json:"media_types,omitempty"`
	ReleasePeriod DateRange   `json:"release_period,omitempty"`
	RatingRange   RatingRange `json:"rating_range,omitempty"`
}
