// Package embeddings owns the pipeline that turns catalog items into vector
// embeddings stored in the AI engine. The Go core is the producer: it never
// computes embeddings itself, only pushes the source text downstream.
package embeddings

import (
	"errors"
	"strings"

	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/infrastructure/clients/ai_engine"
)

// Publisher turns an Item into the text representation used for embedding
// and forwards it to the AI engine.
type Publisher struct {
	Client ai_engine.Client
}

// New constructs a Publisher; client must be non-nil.
func New(client ai_engine.Client) *Publisher {
	return &Publisher{Client: client}
}

// ErrNoText is returned when an item has no fields suitable for embedding.
var ErrNoText = errors.New("embeddings: item has no text to embed")

// Publish builds the textual representation of the item and forwards it to
// the AI engine. Returns ErrNoText if the item lacks any meaningful field.
func (p *Publisher) Publish(item entities.Item) error {
	if p == nil || p.Client == nil {
		return nil
	}
	if item.ID == "" {
		return errors.New("embeddings: item id is required")
	}
	text := BuildText(item)
	if text == "" {
		return ErrNoText
	}
	return p.Client.GenerateEmbedding(ai_engine.EmbeddingRequest{
		ItemID: item.ID,
		Text:   text,
	})
}

// BuildText assembles the canonical textual representation of an item. We
// concatenate the most descriptive fields — title, original title,
// description, genre, themes, setting, tonality and target audience — so
// that the downstream sentence-transformer captures the maximum amount of
// semantic signal for content-based filtering.
func BuildText(item entities.Item) string {
	parts := make([]string, 0, 8)
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s != "" {
			parts = append(parts, s)
		}
	}
	add(item.Title)
	if item.OriginalTitle != "" && !strings.EqualFold(item.OriginalTitle, item.Title) {
		add(item.OriginalTitle)
	}
	add(item.Description)
	add(item.Criteria.Genre)
	add(item.Criteria.Themes)
	add(item.Criteria.Setting)
	add(item.Criteria.Tonality)
	add(item.Criteria.TargetAudience)
	return strings.Join(parts, ". ")
}
