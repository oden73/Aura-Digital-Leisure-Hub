package hybrid

import (
	"errors"
	"testing"

	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/domain/services/cf"
	"aura/backend/core-go/internal/infrastructure/clients/ai_engine"
)

type stubCoordinator struct {
	scores []entities.ScoredItem
	err    error
}

func (s stubCoordinator) GetRecommendations(_ string, _ int) ([]entities.ScoredItem, error) {
	return s.scores, s.err
}

func (stubCoordinator) SelectStrategy(_ entities.UserProfile) cf.Strategy {
	return cf.StrategyHybrid
}

type stubAIClient struct{}

func (stubAIClient) ComputeCB(_ ai_engine.Request) (ai_engine.Response, error) {
	return ai_engine.Response{}, nil
}
func (stubAIClient) GenerateReasoning(_ string, _ []entities.ScoredItem) (string, error) {
	return "", nil
}
func (stubAIClient) GenerateEmbedding(_ ai_engine.EmbeddingRequest) error { return nil }
func (stubAIClient) Chat(_ ai_engine.ChatRequest) (ai_engine.ChatResponse, error) {
	return ai_engine.ChatResponse{}, nil
}

type captureRule struct {
	captured RankingContext
}

func (c *captureRule) Apply(items []entities.ScoredItem, ctx RankingContext) []entities.ScoredItem {
	c.captured = ctx
	return items
}

type fakeProfileRepo struct {
	profile entities.UserProfile
	err     error
}

func (f fakeProfileRepo) GetProfile(_ string) (entities.UserProfile, error) {
	return f.profile, f.err
}

func TestOrchestrator_InjectsProfileIntoRankingContext(t *testing.T) {
	rule := &captureRule{}
	orch := NewOrchestrator(
		stubCoordinator{scores: []entities.ScoredItem{{ItemID: "a", Score: 1}}},
		stubAIClient{},
		NewScoreAggregator(0.5, 0.5),
		NewFinalRanker(rule),
	).WithProfiles(fakeProfileRepo{profile: entities.UserProfile{
		UserID:              "u-1",
		MeanRating:          7.5,
		PreferredGenres:     []string{"rpg"},
		PreferredMediaTypes: []entities.MediaType{entities.MediaTypeGame},
	}})

	if _, err := orch.GetHybridRecommendations("u-1", 10, entities.RecommendationFilters{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.captured.UserProfile.UserID != "u-1" {
		t.Fatalf("expected profile user_id u-1 in context, got %#v", rule.captured.UserProfile)
	}
	if rule.captured.UserProfile.MeanRating != 7.5 {
		t.Fatalf("mean rating not propagated: %v", rule.captured.UserProfile.MeanRating)
	}
}

func TestOrchestrator_ProfileLookupErrorFallsBackToZeroProfile(t *testing.T) {
	rule := &captureRule{}
	orch := NewOrchestrator(
		stubCoordinator{scores: []entities.ScoredItem{{ItemID: "a", Score: 1}}},
		stubAIClient{},
		NewScoreAggregator(0.5, 0.5),
		NewFinalRanker(rule),
	).WithProfiles(fakeProfileRepo{err: errors.New("db down")})

	if _, err := orch.GetHybridRecommendations("u-1", 10, entities.RecommendationFilters{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rule.captured.UserProfile.UserID != "" {
		t.Fatalf("expected zero profile on error, got %#v", rule.captured.UserProfile)
	}
}
