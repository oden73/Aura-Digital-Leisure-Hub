// Package filter implements the FilterService from the cross-cutting services
// class diagram (docs/predone/diagrams/class/cross_cutting_services.puml).
package filter

import (
	"strings"

	"aura/backend/core-go/internal/domain/entities"
)

// MetadataLookup is the minimum interface FilterService needs to fetch the
// catalog metadata required by genre / media-type / release / rating filters.
type MetadataLookup interface {
	GetItem(itemID string) (entities.Item, error)
}

// Service applies in-memory filters on top of ranked recommendation lists.
type Service struct {
	Metadata MetadataLookup
}

// New constructs a FilterService without metadata access — it will be a
// pass-through. Use WithMetadata to enable real filtering.
func New() *Service { return &Service{} }

// WithMetadata returns a service that resolves item metadata for filtering.
func (s *Service) WithMetadata(m MetadataLookup) *Service {
	s.Metadata = m
	return s
}

// Apply applies all configured filters in RecommendationFilters to items.
// It is safe to call when Metadata is nil — items pass through unchanged.
func (s *Service) Apply(
	items []entities.ScoredItem,
	filters entities.RecommendationFilters,
) []entities.ScoredItem {
	if s.Metadata == nil || isEmpty(filters) || len(items) == 0 {
		return items
	}

	wantedGenres := lowerSet(filters.Genres)
	wantedMedia := mediaSet(filters.MediaTypes)

	out := make([]entities.ScoredItem, 0, len(items))
	for _, it := range items {
		meta, err := s.Metadata.GetItem(it.ItemID)
		if err != nil {
			// Metadata missing: be conservative and drop the item rather
			// than silently violate filters.
			continue
		}
		if !matchGenre(meta, wantedGenres) {
			continue
		}
		if !matchMediaType(meta, wantedMedia) {
			continue
		}
		if !matchReleasePeriod(meta, filters.ReleasePeriod) {
			continue
		}
		if !matchRatingRange(meta, filters.RatingRange) {
			continue
		}
		out = append(out, it)
	}
	return out
}

func isEmpty(f entities.RecommendationFilters) bool {
	return len(f.Genres) == 0 &&
		len(f.MediaTypes) == 0 &&
		f.ReleasePeriod.From == nil &&
		f.ReleasePeriod.To == nil &&
		f.RatingRange.Min == 0 &&
		f.RatingRange.Max == 0
}

func lowerSet(values []string) map[string]struct{} {
	out := make(map[string]struct{}, len(values))
	for _, v := range values {
		v = strings.ToLower(strings.TrimSpace(v))
		if v == "" {
			continue
		}
		out[v] = struct{}{}
	}
	return out
}

func mediaSet(values []entities.MediaType) map[entities.MediaType]struct{} {
	out := make(map[entities.MediaType]struct{}, len(values))
	for _, v := range values {
		out[v] = struct{}{}
	}
	return out
}

func matchGenre(it entities.Item, wanted map[string]struct{}) bool {
	if len(wanted) == 0 {
		return true
	}
	g := strings.ToLower(strings.TrimSpace(it.Criteria.Genre))
	if g == "" {
		return false
	}
	_, ok := wanted[g]
	return ok
}

func matchMediaType(it entities.Item, wanted map[entities.MediaType]struct{}) bool {
	if len(wanted) == 0 {
		return true
	}
	_, ok := wanted[it.MediaType]
	return ok
}

func matchReleasePeriod(it entities.Item, period entities.DateRange) bool {
	if period.From == nil && period.To == nil {
		return true
	}
	if it.ReleaseDate == nil {
		return false
	}
	if period.From != nil && it.ReleaseDate.Before(*period.From) {
		return false
	}
	if period.To != nil && it.ReleaseDate.After(*period.To) {
		return false
	}
	return true
}

func matchRatingRange(it entities.Item, r entities.RatingRange) bool {
	if r.Min == 0 && r.Max == 0 {
		return true
	}
	if r.Min > 0 && it.AverageRating < float64(r.Min) {
		return false
	}
	if r.Max > 0 && it.AverageRating > float64(r.Max) {
		return false
	}
	return true
}
