package entities

// ScoreSource identifies which algorithm produced a given score.
type ScoreSource string

const (
	ScoreSourceCF     ScoreSource = "cf"
	ScoreSourceCB     ScoreSource = "cb"
	ScoreSourceHybrid ScoreSource = "hybrid"
	// ScoreSourcePopular tags items produced by the cold-start fallback
	// (top-rated items of the catalog) instead of a personalised algorithm.
	ScoreSourcePopular ScoreSource = "popular"
)

// ScoredItem represents a candidate with a ranking score.
type ScoredItem struct {
	ItemID   string         `json:"item_id"`
	Score    float64        `json:"score"`
	Source   ScoreSource    `json:"source,omitempty"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

// CombineWith merges two scores for the same item using the provided weight
// for the other side. The receiver acts as the base (1 - weight).
func (s ScoredItem) CombineWith(other ScoredItem, weight float64) ScoredItem {
	return ScoredItem{
		ItemID:   s.ItemID,
		Score:    s.Score*(1-weight) + other.Score*weight,
		Source:   ScoreSourceHybrid,
		Metadata: s.Metadata,
	}
}
