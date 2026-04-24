package entities

// ScoreSource identifies which algorithm produced a given score.
type ScoreSource string

const (
	ScoreSourceCF     ScoreSource = "cf"
	ScoreSourceCB     ScoreSource = "cb"
	ScoreSourceHybrid ScoreSource = "hybrid"
)

// ScoredItem represents a candidate with a ranking score.
type ScoredItem struct {
	ItemID   string
	Score    float64
	Source   ScoreSource
	Metadata map[string]any
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
