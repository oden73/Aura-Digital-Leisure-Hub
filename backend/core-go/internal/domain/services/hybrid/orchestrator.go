package hybrid

import (
	"time"

	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/domain/services/cf"
	"aura/backend/core-go/internal/infrastructure/clients/ai_engine"
)

// Orchestrator aggregates CF (in-process) and CB (remote AI engine) scores
// and applies the final ranking rules.
type Orchestrator interface {
	GetHybridRecommendations(
		userID string,
		k int,
		filters entities.RecommendationFilters,
	) ([]entities.ScoredItem, error)
}

// DefaultOrchestrator wires the CF coordinator, AI engine client, aggregator
// and ranker.
type DefaultOrchestrator struct {
	CFModule   cf.Coordinator
	AIEngine   ai_engine.Client
	Aggregator *ScoreAggregator
	Ranker     *FinalRanker
}

// NewOrchestrator constructs the orchestrator.
func NewOrchestrator(
	cfModule cf.Coordinator,
	aiEngine ai_engine.Client,
	aggregator *ScoreAggregator,
	ranker *FinalRanker,
) *DefaultOrchestrator {
	return &DefaultOrchestrator{
		CFModule:   cfModule,
		AIEngine:   aiEngine,
		Aggregator: aggregator,
		Ranker:     ranker,
	}
}

// GetHybridRecommendations returns a ranked list for the given user.
func (o *DefaultOrchestrator) GetHybridRecommendations(
	userID string,
	k int,
	filters entities.RecommendationFilters,
) ([]entities.ScoredItem, error) {
	_ = filters

	cfScores, err := o.CFModule.GetRecommendations(userID, k)
	if err != nil {
		return nil, err
	}

	cbScores, err := o.AIEngine.ComputeCBScores(userID, nil, k)
	if err != nil {
		return nil, err
	}

	aggregated := o.Aggregator.AggregateScores(cfScores, cbScores)

	ctx := RankingContext{
		CurrentDate: time.Now(),
		TargetCount: k,
	}
	ranked := o.Ranker.Rank(aggregated, ctx)
	return ranked, nil
}
