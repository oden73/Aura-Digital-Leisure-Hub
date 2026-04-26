package usecase

import (
	"errors"
	"testing"

	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/domain/services/embeddings"
	"aura/backend/core-go/internal/infrastructure/clients/ai_engine"
)

type fakeMetadataSaver struct {
	saved entities.Item
	err   error
}

func (f *fakeMetadataSaver) SaveItem(item entities.Item) error {
	if f.err != nil {
		return f.err
	}
	f.saved = item
	return nil
}

type fakeAIClient struct {
	embedCalls []ai_engine.EmbeddingRequest
	embedErr   error
}

func (f *fakeAIClient) ComputeCB(_ ai_engine.Request) (ai_engine.Response, error) {
	return ai_engine.Response{}, nil
}
func (f *fakeAIClient) GenerateReasoning(_ string, _ []entities.ScoredItem) (string, error) {
	return "", nil
}
func (f *fakeAIClient) GenerateEmbedding(req ai_engine.EmbeddingRequest) error {
	f.embedCalls = append(f.embedCalls, req)
	return f.embedErr
}

func TestUpsertContent_TriggersEmbeddingPublish(t *testing.T) {
	saver := &fakeMetadataSaver{}
	ai := &fakeAIClient{}
	uc := NewUpsertContent(saver, embeddings.New(ai))

	err := uc.Execute(entities.Item{ID: "i-1", Title: "Foo", Description: "Bar"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if saver.saved.ID != "i-1" {
		t.Fatalf("item not saved: %+v", saver.saved)
	}
	if len(ai.embedCalls) != 1 || ai.embedCalls[0].ItemID != "i-1" {
		t.Fatalf("expected embedding push for i-1, got %+v", ai.embedCalls)
	}
}

func TestUpsertContent_EmbeddingFailureDoesNotBubbleUp(t *testing.T) {
	saver := &fakeMetadataSaver{}
	ai := &fakeAIClient{embedErr: errors.New("boom")}
	uc := NewUpsertContent(saver, embeddings.New(ai))

	err := uc.Execute(entities.Item{ID: "i-1", Title: "Foo"})
	if err != nil {
		t.Fatalf("embedding failure must be swallowed, got %v", err)
	}
}

func TestUpsertContent_SaveErrorPropagates(t *testing.T) {
	saver := &fakeMetadataSaver{err: errors.New("db down")}
	ai := &fakeAIClient{}
	uc := NewUpsertContent(saver, embeddings.New(ai))

	err := uc.Execute(entities.Item{ID: "i-1", Title: "Foo"})
	if err == nil {
		t.Fatal("expected save error to propagate")
	}
	if len(ai.embedCalls) != 0 {
		t.Fatal("must not push embedding when save fails")
	}
}

func TestUpsertContent_NilPublisherSkipsEmbedding(t *testing.T) {
	saver := &fakeMetadataSaver{}
	uc := NewUpsertContent(saver, nil)
	if err := uc.Execute(entities.Item{ID: "i-1", Title: "Foo"}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
