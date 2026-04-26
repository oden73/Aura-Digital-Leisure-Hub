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
	) (Result, error)
}

// Result is the orchestrator output: ranked items plus optional reasoning
// produced upstream by the AI engine.
type Result struct {
	Items     []entities.ScoredItem
	Reasoning string
}

// MetadataLookup is an optional dependency used by the orchestrator to
// hydrate the RankingContext so that ranking rules can reason about catalog
// metadata (genre, release date, average rating).
type MetadataLookup interface {
	GetItem(itemID string) (entities.Item, error)
}

// ProfileLookup is an optional dependency used by the orchestrator to
// inject the user's preference profile into the RankingContext. The
// profile feeds preference-aware ranking rules without each rule having
// to query the database.
type ProfileLookup interface {
	GetProfile(userID string) (entities.UserProfile, error)
}

// DefaultOrchestrator wires the CF coordinator, AI engine client, aggregator
// and ranker.
type DefaultOrchestrator struct {
	CFModule   cf.Coordinator
	AIEngine   ai_engine.Client
	Aggregator *ScoreAggregator
	Ranker     *FinalRanker
	Metadata   MetadataLookup
	Profiles   ProfileLookup
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

// WithMetadata attaches a metadata lookup used to enrich the ranking context.
func (o *DefaultOrchestrator) WithMetadata(m MetadataLookup) *DefaultOrchestrator {
	o.Metadata = m
	return o
}

// WithProfiles attaches a profile lookup used to enrich the ranking
// context with the caller's preference profile.
func (o *DefaultOrchestrator) WithProfiles(p ProfileLookup) *DefaultOrchestrator {
	o.Profiles = p
	return o
}

// GetHybridRecommendations returns a ranked list for the given user.
func (o *DefaultOrchestrator) GetHybridRecommendations(
	userID string,
	k int,
	filters entities.RecommendationFilters,
) (Result, error) {
	cfScores, err := o.CFModule.GetRecommendations(userID, k)
	if err != nil {
		return Result{}, err
	}

	cbResp, err := o.AIEngine.ComputeCB(ai_engine.Request{
		UserID:     userID,
		Limit:      k,
		MediaTypes: filters.MediaTypes,
	})
	if err != nil {
		// AI engine is best-effort: degrade gracefully if it is unavailable.
		cbResp = ai_engine.Response{}
	}

	aggregated := o.Aggregator.AggregateScores(cfScores, cbResp.Items)

	ctx := RankingContext{
		CurrentDate: time.Now(),
		TargetCount: k,
	}
	if o.Profiles != nil {
		if profile, err := o.Profiles.GetProfile(userID); err == nil {
			ctx.UserProfile = profile
		}
	}
	if o.Metadata != nil {
		ctx.ItemMeta = make(map[string]entities.Item, len(aggregated))
		for _, it := range aggregated {
			meta, err := o.Metadata.GetItem(it.ItemID)
			if err != nil {
				continue
			}
			ctx.ItemMeta[it.ItemID] = meta
		}
	}

	ranked := o.Ranker.Rank(aggregated, ctx)
	return Result{Items: ranked, Reasoning: cbResp.Reasoning}, nil
}
