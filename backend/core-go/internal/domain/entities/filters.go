package entities

import "time"

// DateRange is an inclusive window on release dates.
type DateRange struct {
	From *time.Time
	To   *time.Time
}

// RatingRange is an inclusive window on the 1-10 rating scale.
type RatingRange struct {
	Min int
	Max int
}

// RecommendationFilters describes filters applied to a recommendation response.
type RecommendationFilters struct {
	Genres        []string
	MediaTypes    []MediaType
	ReleasePeriod DateRange
	RatingRange   RatingRange
}
