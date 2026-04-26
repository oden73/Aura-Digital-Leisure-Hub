package embeddings

import (
	"errors"
	"testing"

	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/infrastructure/clients/ai_engine"
)

type recorder struct {
	calls []ai_engine.EmbeddingRequest
	err   error
}

func (r *recorder) ComputeCB(_ ai_engine.Request) (ai_engine.Response, error) {
	return ai_engine.Response{}, nil
}
func (r *recorder) GenerateReasoning(_ string, _ []entities.ScoredItem) (string, error) {
	return "", nil
}
func (r *recorder) GenerateEmbedding(req ai_engine.EmbeddingRequest) error {
	r.calls = append(r.calls, req)
	return r.err
}

func TestBuildText_ConcatenatesDescriptiveFields(t *testing.T) {
	item := entities.Item{
		ID:            "i-1",
		Title:         "Witcher",
		OriginalTitle: "Wiedźmin",
		Description:   "A dark fantasy saga",
		Criteria: entities.BaseItemCriteria{
			Genre:          "RPG",
			Themes:         "monsters, choices",
			Setting:        "medieval",
			Tonality:       "grim",
			TargetAudience: "adults",
		},
	}
	text := BuildText(item)
	for _, want := range []string{"Witcher", "Wiedźmin", "dark fantasy", "RPG", "medieval", "grim"} {
		if !contains(text, want) {
			t.Fatalf("BuildText missing %q in %q", want, text)
		}
	}
}

func TestBuildText_DropsDuplicateOriginalTitle(t *testing.T) {
	item := entities.Item{Title: "Foo", OriginalTitle: "foo", Description: "d"}
	text := BuildText(item)
	if got := count(text, "Foo"); got != 1 {
		t.Fatalf("expected single Title occurrence, got %d in %q", got, text)
	}
}

func TestBuildText_EmptyWhenNothingToSay(t *testing.T) {
	if BuildText(entities.Item{}) != "" {
		t.Fatal("expected empty text for empty item")
	}
}

func TestPublisher_Publish_ForwardsItemText(t *testing.T) {
	rec := &recorder{}
	p := New(rec)
	err := p.Publish(entities.Item{ID: "i-1", Title: "Foo", Description: "Bar"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rec.calls) != 1 {
		t.Fatalf("expected 1 call, got %d", len(rec.calls))
	}
	if rec.calls[0].ItemID != "i-1" || !contains(rec.calls[0].Text, "Foo") {
		t.Fatalf("payload mismatch: %#v", rec.calls[0])
	}
}

func TestPublisher_Publish_ErrNoTextWhenItemEmpty(t *testing.T) {
	rec := &recorder{}
	p := New(rec)
	err := p.Publish(entities.Item{ID: "i-1"})
	if !errors.Is(err, ErrNoText) {
		t.Fatalf("expected ErrNoText, got %v", err)
	}
	if len(rec.calls) != 0 {
		t.Fatal("client should not be called when no text is available")
	}
}

func TestPublisher_Publish_RequiresID(t *testing.T) {
	rec := &recorder{}
	p := New(rec)
	if err := p.Publish(entities.Item{Title: "Foo"}); err == nil {
		t.Fatal("expected error when item id is empty")
	}
}

func TestPublisher_Publish_NilClientIsNoop(t *testing.T) {
	var p *Publisher
	if err := p.Publish(entities.Item{ID: "i-1", Title: "Foo"}); err != nil {
		t.Fatalf("nil publisher must be a noop, got %v", err)
	}
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

func count(s, sub string) int {
	if sub == "" {
		return 0
	}
	n := 0
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			n++
			i += len(sub) - 1
		}
	}
	return n
}
