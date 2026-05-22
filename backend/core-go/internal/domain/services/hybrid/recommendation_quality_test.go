package hybrid

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/domain/services/cf"
	"aura/backend/core-go/internal/infrastructure/clients/ai_engine"
)

const (
	recommendationQualityK = 5
	// Candidate pool depth for offline quality mocks only (production CF/CB still pass k).
	recommendationQualityCandidatePool = 56
)

type qualityCFCoordinator struct {
	scores map[string][]entities.ScoredItem
}

func (q qualityCFCoordinator) GetRecommendations(userID string, k int) ([]entities.ScoredItem, error) {
	items := append([]entities.ScoredItem(nil), q.scores[userID]...)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score != items[j].Score {
			return items[i].Score > items[j].Score
		}
		return items[i].ItemID < items[j].ItemID
	})
	limit := recommendationQualityCandidatePool
	if k > limit {
		limit = k
	}
	if limit > 0 && len(items) > limit {
		return items[:limit], nil
	}
	return items, nil
}

func (qualityCFCoordinator) SelectStrategy(_ entities.UserProfile) cf.Strategy {
	return cf.StrategyHybrid
}

type qualityAIClient struct {
	scores map[string][]entities.ScoredItem
}

func (q qualityAIClient) ComputeCB(req ai_engine.Request) (ai_engine.Response, error) {
	items := append([]entities.ScoredItem(nil), q.scores[req.UserID]...)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Score != items[j].Score {
			return items[i].Score > items[j].Score
		}
		return items[i].ItemID < items[j].ItemID
	})
	limit := recommendationQualityCandidatePool
	if req.Limit > limit {
		limit = req.Limit
	}
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return ai_engine.Response{
		Items:     items,
		Reasoning: "offline quality evaluation",
	}, nil
}

func (qualityAIClient) GenerateReasoning(_ string, _ []entities.ScoredItem) (string, error) {
	return "", nil
}

func (qualityAIClient) GenerateEmbedding(_ ai_engine.EmbeddingRequest) error { return nil }

func (qualityAIClient) Chat(_ ai_engine.ChatRequest) (ai_engine.ChatResponse, error) {
	return ai_engine.ChatResponse{}, nil
}

type qualityMetadataRepo struct {
	items map[string]entities.Item
}

func (q qualityMetadataRepo) GetItem(id string) (entities.Item, error) {
	item, ok := q.items[id]
	if !ok {
		return entities.Item{}, fmt.Errorf("missing item metadata: %s", id)
	}
	return item, nil
}

type qualityScenario struct {
	Name     string
	UserID   string
	Relevant map[string]struct{}
}

type qualityScenarioResult struct {
	Name        string
	IDs         []string
	Precision   float64
	Recall      float64
	NDCG        float64
	MRR         float64
	HitRate     float64
	Diversity   float64
	DurationMS  float64
	RelevantHit int
}

type qualitySummary struct {
	PrecisionAtK    float64
	RecallAtK       float64
	NDCGAtK         float64
	MRRAtK          float64
	HitRateAtK      float64
	CoverageAtK     float64
	GenreDiversity  float64
	TotalDurationMS float64
}

func TestRecommendationSystemQualityReport(t *testing.T) {
	now := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)
	catalog := qualityCatalog(now)
	scenarios := []qualityScenario{
		{
			Name:   "new_user",
			UserID: "u-new",
			Relevant: setOf(
				"book-earthsea",
				"game-zelda",
				"film-spirited-away",
			),
		},
		{
			Name:   "narrow_genre_preferences",
			UserID: "u-narrow",
			Relevant: setOf(
				"book-dune",
				"book-foundation",
				"book-hyperion",
				"film-arrival",
			),
		},
		{
			Name:   "multimodal_interests",
			UserID: "u-multimodal",
			Relevant: setOf(
				"book-dune",
				"game-mass-effect",
				"film-arrival",
				"game-zelda",
				"book-earthsea",
			),
		},
		{
			Name:   "conflicting_preferences",
			UserID: "u-conflict",
			Relevant: setOf(
				"book-neuromancer",
				"film-blade-runner",
				"game-deus-ex",
				"book-earthsea",
			),
		},
		{
			Name:   "long_profile",
			UserID: "u-long",
			Relevant: setOf(
				"book-dune",
				"game-mass-effect",
				"film-arrival",
				"book-foundation",
				"film-matrix",
			),
		},
		{
			Name:   "short_profile",
			UserID: "u-short",
			Relevant: setOf(
				"book-earthsea",
				"game-zelda",
			),
		},
	}

	orch := NewOrchestrator(
		qualityCFCoordinator{scores: buildQualityCFScores(catalog)},
		qualityAIClient{scores: buildQualityCBScores(catalog)},
		NewScoreAggregator(0.6, 0.4),
		NewFinalRanker(
			RecencyBoostRule{DecayFactor: 0.01},
			DiversityRule{DiversityThreshold: 0.1},
		),
	).WithMetadata(qualityMetadataRepo{items: catalog})

	results := make([]qualityScenarioResult, 0, len(scenarios))
	recommended := map[string]struct{}{}
	started := time.Now()
	for _, scenario := range scenarios {
		scenarioStart := time.Now()
		res, err := orch.GetHybridRecommendations(
			scenario.UserID,
			recommendationQualityK,
			entities.RecommendationFilters{},
		)
		if err != nil {
			t.Fatalf("%s: recommendations failed: %v", scenario.Name, err)
		}
		ids := itemIDs(res.Items)
		for _, id := range ids {
			recommended[id] = struct{}{}
		}
		results = append(results, evaluateScenario(
			scenario.Name,
			ids,
			scenario.Relevant,
			catalog,
			scenarioStart,
		))
	}

	summary := summarizeQuality(results, len(recommended), len(catalog), time.Since(started))
	reportPath, err := writeRecommendationQualityReport(summary, results, recommendationQualityK)
	if err != nil {
		t.Fatalf("write quality report: %v", err)
	}
	t.Logf("recommendation quality report: %s", reportPath)

	assertAtLeast(t, "precision@5", summary.PrecisionAtK, 0.40)
	assertAtLeast(t, "recall@5", summary.RecallAtK, 0.50)
	assertAtLeast(t, "ndcg@5", summary.NDCGAtK, 0.52)
	assertAtLeast(t, "mrr@5", summary.MRRAtK, 0.65)
	assertAtLeast(t, "hit_rate@5", summary.HitRateAtK, 1.00)
	assertAtLeast(t, "catalog_coverage@5", summary.CoverageAtK, 0.12)
	assertAtLeast(t, "genre_diversity@5", summary.GenreDiversity, 0.35)
}

func evaluateScenario(
	name string,
	ids []string,
	relevant map[string]struct{},
	catalog map[string]entities.Item,
	started time.Time,
) qualityScenarioResult {
	hits := 0
	dcg := 0.0
	firstRelevantRank := 0
	genres := map[string]struct{}{}

	for i, id := range ids {
		if item, ok := catalog[id]; ok {
			genre := strings.ToLower(strings.TrimSpace(item.Criteria.Genre))
			if genre != "" {
				genres[genre] = struct{}{}
			}
		}
		if _, ok := relevant[id]; !ok {
			continue
		}
		hits++
		rank := i + 1
		if firstRelevantRank == 0 {
			firstRelevantRank = rank
		}
		dcg += 1 / math.Log2(float64(rank)+1)
	}

	idealHits := len(relevant)
	if idealHits > len(ids) {
		idealHits = len(ids)
	}
	idcg := 0.0
	for i := 1; i <= idealHits; i++ {
		idcg += 1 / math.Log2(float64(i)+1)
	}

	result := qualityScenarioResult{
		Name:        name,
		IDs:         ids,
		Precision:   ratio(hits, len(ids)),
		Recall:      ratio(hits, len(relevant)),
		NDCG:        safeDivide(dcg, idcg),
		HitRate:     boolScore(hits > 0),
		Diversity:   ratio(len(genres), len(ids)),
		DurationMS:  float64(time.Since(started).Microseconds()) / 1000,
		RelevantHit: hits,
	}
	if firstRelevantRank > 0 {
		result.MRR = 1 / float64(firstRelevantRank)
	}
	return result
}

func summarizeQuality(
	results []qualityScenarioResult,
	recommendedCount int,
	catalogSize int,
	duration time.Duration,
) qualitySummary {
	var summary qualitySummary
	for _, r := range results {
		summary.PrecisionAtK += r.Precision
		summary.RecallAtK += r.Recall
		summary.NDCGAtK += r.NDCG
		summary.MRRAtK += r.MRR
		summary.HitRateAtK += r.HitRate
		summary.GenreDiversity += r.Diversity
	}
	n := float64(len(results))
	if n > 0 {
		summary.PrecisionAtK /= n
		summary.RecallAtK /= n
		summary.NDCGAtK /= n
		summary.MRRAtK /= n
		summary.HitRateAtK /= n
		summary.GenreDiversity /= n
	}
	summary.CoverageAtK = ratio(recommendedCount, catalogSize)
	summary.TotalDurationMS = float64(duration.Microseconds()) / 1000
	return summary
}

func writeRecommendationQualityReport(
	summary qualitySummary,
	results []qualityScenarioResult,
	k int,
) (string, error) {
	root, err := findBackendRoot()
	if err != nil {
		return "", err
	}
	reportsDir := filepath.Join(root, "reports")
	if err := os.MkdirAll(reportsDir, 0o755); err != nil {
		return "", err
	}

	reportPath := filepath.Join(reportsDir, "recommendation_quality_report.md")
	var b strings.Builder
	fmt.Fprintf(&b, "# Recommendation quality report\n\n")
	fmt.Fprintf(&b, "- precision@%d: %.3f\n", k, summary.PrecisionAtK)
	fmt.Fprintf(&b, "- recall@%d: %.3f\n", k, summary.RecallAtK)
	fmt.Fprintf(&b, "- ndcg@%d: %.3f\n", k, summary.NDCGAtK)
	fmt.Fprintf(&b, "- mrr@%d: %.3f\n", k, summary.MRRAtK)
	fmt.Fprintf(&b, "- hit_rate@%d: %.3f\n", k, summary.HitRateAtK)
	fmt.Fprintf(&b, "- catalog_coverage@%d: %.3f\n", k, summary.CoverageAtK)
	fmt.Fprintf(&b, "- genre_diversity@%d: %.3f\n", k, summary.GenreDiversity)
	fmt.Fprintf(&b, "- total_duration_ms: %.3f\n\n", summary.TotalDurationMS)
	fmt.Fprintf(&b, "## Scenarios\n\n")
	for _, r := range results {
		fmt.Fprintf(&b, "### %s\n\n", r.Name)
		fmt.Fprintf(&b, "- recommended: %s\n", strings.Join(r.IDs, ", "))
		fmt.Fprintf(&b, "- relevant_hits: %d\n", r.RelevantHit)
		fmt.Fprintf(&b, "- precision@%d: %.3f\n", k, r.Precision)
		fmt.Fprintf(&b, "- recall@%d: %.3f\n", k, r.Recall)
		fmt.Fprintf(&b, "- ndcg@%d: %.3f\n", k, r.NDCG)
		fmt.Fprintf(&b, "- mrr@%d: %.3f\n", k, r.MRR)
		fmt.Fprintf(&b, "- hit_rate@%d: %.3f\n", k, r.HitRate)
		fmt.Fprintf(&b, "- genre_diversity@%d: %.3f\n", k, r.Diversity)
		fmt.Fprintf(&b, "- duration_ms: %.3f\n\n", r.DurationMS)
	}

	return reportPath, os.WriteFile(reportPath, []byte(b.String()), 0o644)
}

func findBackendRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if filepath.Base(dir) == "core-go" {
			return filepath.Clean(filepath.Join(dir, "..")), nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("backend root not found from %s", dir)
		}
		dir = parent
	}
}

func qualityCoreCFScores() map[string][]entities.ScoredItem {
	return map[string][]entities.ScoredItem{
		"u-new": {},
		"u-narrow": {
			{ItemID: "book-dune", Score: 0.95, Source: entities.ScoreSourceCF},
			{ItemID: "book-foundation", Score: 0.86, Source: entities.ScoreSourceCF},
			{ItemID: "book-hyperion", Score: 0.82, Source: entities.ScoreSourceCF},
			{ItemID: "film-matrix", Score: 0.50, Source: entities.ScoreSourceCF},
		},
		"u-multimodal": {
			{ItemID: "book-dune", Score: 0.95, Source: entities.ScoreSourceCF},
			{ItemID: "game-mass-effect", Score: 0.84, Source: entities.ScoreSourceCF},
			{ItemID: "film-arrival", Score: 0.80, Source: entities.ScoreSourceCF},
			{ItemID: "game-zelda", Score: 0.72, Source: entities.ScoreSourceCF},
			{ItemID: "book-earthsea", Score: 0.68, Source: entities.ScoreSourceCF},
		},
		"u-conflict": {
			{ItemID: "book-neuromancer", Score: 0.88, Source: entities.ScoreSourceCF},
			{ItemID: "film-ghost-shell", Score: 0.80, Source: entities.ScoreSourceCF},
			{ItemID: "game-deus-ex", Score: 0.78, Source: entities.ScoreSourceCF},
			{ItemID: "game-control", Score: 0.45, Source: entities.ScoreSourceCF},
		},
		"u-long": {
			{ItemID: "book-dune", Score: 0.95, Source: entities.ScoreSourceCF},
			{ItemID: "game-mass-effect", Score: 0.84, Source: entities.ScoreSourceCF},
			{ItemID: "film-arrival", Score: 0.80, Source: entities.ScoreSourceCF},
			{ItemID: "film-matrix", Score: 0.45, Source: entities.ScoreSourceCF},
			{ItemID: "book-hyperion", Score: 0.30, Source: entities.ScoreSourceCF},
		},
		"u-short": {
			{ItemID: "book-earthsea", Score: 0.70, Source: entities.ScoreSourceCF},
		},
	}
}

func qualityCoreCBScores() map[string][]entities.ScoredItem {
	return map[string][]entities.ScoredItem{
		"u-new": {
			{ItemID: "game-zelda", Score: 0.88, Source: entities.ScoreSourceCB},
			{ItemID: "book-earthsea", Score: 0.84, Source: entities.ScoreSourceCB},
			{ItemID: "film-spirited-away", Score: 0.76, Source: entities.ScoreSourceCB},
		},
		"u-narrow": {
			{ItemID: "film-arrival", Score: 0.92, Source: entities.ScoreSourceCB},
			{ItemID: "book-dune", Score: 0.88, Source: entities.ScoreSourceCB},
			{ItemID: "book-foundation", Score: 0.78, Source: entities.ScoreSourceCB},
			{ItemID: "book-hyperion", Score: 0.64, Source: entities.ScoreSourceCB},
		},
		"u-multimodal": {
			{ItemID: "book-dune", Score: 0.80, Source: entities.ScoreSourceCB},
			{ItemID: "film-arrival", Score: 0.92, Source: entities.ScoreSourceCB},
			{ItemID: "game-mass-effect", Score: 0.50, Source: entities.ScoreSourceCB},
			{ItemID: "game-zelda", Score: 0.86, Source: entities.ScoreSourceCB},
			{ItemID: "book-earthsea", Score: 0.74, Source: entities.ScoreSourceCB},
		},
		"u-conflict": {
			{ItemID: "film-blade-runner", Score: 0.95, Source: entities.ScoreSourceCB},
			{ItemID: "book-earthsea", Score: 0.84, Source: entities.ScoreSourceCB},
			{ItemID: "film-spirited-away", Score: 0.72, Source: entities.ScoreSourceCB},
			{ItemID: "book-neuromancer", Score: 0.62, Source: entities.ScoreSourceCB},
			{ItemID: "game-deus-ex", Score: 0.55, Source: entities.ScoreSourceCB},
		},
		"u-long": {
			{ItemID: "book-dune", Score: 0.80, Source: entities.ScoreSourceCB},
			{ItemID: "film-arrival", Score: 0.92, Source: entities.ScoreSourceCB},
			{ItemID: "book-foundation", Score: 0.70, Source: entities.ScoreSourceCB},
			{ItemID: "game-mass-effect", Score: 0.50, Source: entities.ScoreSourceCB},
			{ItemID: "game-control", Score: 0.20, Source: entities.ScoreSourceCB},
		},
		"u-short": {
			{ItemID: "book-earthsea", Score: 0.88, Source: entities.ScoreSourceCB},
			{ItemID: "game-zelda", Score: 0.80, Source: entities.ScoreSourceCB},
			{ItemID: "film-spirited-away", Score: 0.40, Source: entities.ScoreSourceCB},
		},
	}
}

func sortedQualityFillerIDs(catalog map[string]entities.Item) []string {
	ids := make([]string, 0)
	for id := range catalog {
		if strings.HasPrefix(id, "fill-") {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func qualityNoiseScore(uid string, channel string, itemID string) float64 {
	h := uint32(2166136261)
	for _, b := range []byte(uid + "|" + channel + "|" + itemID) {
		h ^= uint32(b)
		h *= 16777619
	}
	n := float64(h%8192) / 8192.0
	return 0.30 + n*0.22
}

func mergeQualityScoresWithCatalogNoise(
	uid string,
	channel string,
	fillers []string,
	core []entities.ScoredItem,
	src entities.ScoreSource,
) []entities.ScoredItem {
	out := make([]entities.ScoredItem, 0, len(fillers)+len(core))
	for _, id := range fillers {
		out = append(out, entities.ScoredItem{
			ItemID: id,
			Score:  qualityNoiseScore(uid, channel, id),
			Source: src,
		})
	}
	out = append(out, core...)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Score != out[j].Score {
			return out[i].Score > out[j].Score
		}
		return out[i].ItemID < out[j].ItemID
	})
	return out
}

func buildQualityCFScores(catalog map[string]entities.Item) map[string][]entities.ScoredItem {
	fillers := sortedQualityFillerIDs(catalog)
	core := qualityCoreCFScores()
	out := make(map[string][]entities.ScoredItem, len(core))
	for uid, items := range core {
		f := fillers
		if len(items) == 0 {
			f = nil
		}
		out[uid] = mergeQualityScoresWithCatalogNoise(uid, "cf", f, items, entities.ScoreSourceCF)
	}
	return out
}

func buildQualityCBScores(catalog map[string]entities.Item) map[string][]entities.ScoredItem {
	fillers := sortedQualityFillerIDs(catalog)
	core := qualityCoreCBScores()
	out := make(map[string][]entities.ScoredItem, len(core))
	for uid, items := range core {
		out[uid] = mergeQualityScoresWithCatalogNoise(uid, "cb", fillers, items, entities.ScoreSourceCB)
	}
	return out
}

func qualityCatalog(now time.Time) map[string]entities.Item {
	yearsAgo := func(years int) *time.Time {
		t := now.AddDate(-years, 0, 0)
		return &t
	}
	items := []entities.Item{
		{ID: "book-dune", Title: "Dune", MediaType: entities.MediaTypeBook, AverageRating: 9.3, ReleaseDate: yearsAgo(61), Criteria: entities.BaseItemCriteria{Genre: "Sci-Fi"}},
		{ID: "game-mass-effect", Title: "Mass Effect", MediaType: entities.MediaTypeGame, AverageRating: 9.1, ReleaseDate: yearsAgo(19), Criteria: entities.BaseItemCriteria{Genre: "RPG"}},
		{ID: "film-arrival", Title: "Arrival", MediaType: entities.MediaTypeCinema, AverageRating: 8.4, ReleaseDate: yearsAgo(10), Criteria: entities.BaseItemCriteria{Genre: "Sci-Fi Drama"}},
		{ID: "book-foundation", Title: "Foundation", MediaType: entities.MediaTypeBook, AverageRating: 8.8, ReleaseDate: yearsAgo(75), Criteria: entities.BaseItemCriteria{Genre: "Classic Sci-Fi"}},
		{ID: "film-matrix", Title: "The Matrix", MediaType: entities.MediaTypeCinema, AverageRating: 8.9, ReleaseDate: yearsAgo(27), Criteria: entities.BaseItemCriteria{Genre: "Action Sci-Fi"}},
		{ID: "book-hyperion", Title: "Hyperion", MediaType: entities.MediaTypeBook, AverageRating: 8.7, ReleaseDate: yearsAgo(37), Criteria: entities.BaseItemCriteria{Genre: "Sci-Fi"}},
		{ID: "game-control", Title: "Control", MediaType: entities.MediaTypeGame, AverageRating: 8.2, ReleaseDate: yearsAgo(7), Criteria: entities.BaseItemCriteria{Genre: "Action Adventure"}},
		{ID: "book-neuromancer", Title: "Neuromancer", MediaType: entities.MediaTypeBook, AverageRating: 8.6, ReleaseDate: yearsAgo(42), Criteria: entities.BaseItemCriteria{Genre: "Cyberpunk"}},
		{ID: "film-ghost-shell", Title: "Ghost in the Shell", MediaType: entities.MediaTypeCinema, AverageRating: 8.1, ReleaseDate: yearsAgo(31), Criteria: entities.BaseItemCriteria{Genre: "Cyberpunk Anime"}},
		{ID: "film-blade-runner", Title: "Blade Runner", MediaType: entities.MediaTypeCinema, AverageRating: 8.7, ReleaseDate: yearsAgo(44), Criteria: entities.BaseItemCriteria{Genre: "Neo-noir"}},
		{ID: "game-deus-ex", Title: "Deus Ex", MediaType: entities.MediaTypeGame, AverageRating: 8.9, ReleaseDate: yearsAgo(26), Criteria: entities.BaseItemCriteria{Genre: "Immersive Sim"}},
		{ID: "book-earthsea", Title: "A Wizard of Earthsea", MediaType: entities.MediaTypeBook, AverageRating: 8.5, ReleaseDate: yearsAgo(58), Criteria: entities.BaseItemCriteria{Genre: "Fantasy"}},
		{ID: "game-zelda", Title: "The Legend of Zelda", MediaType: entities.MediaTypeGame, AverageRating: 9.0, ReleaseDate: yearsAgo(39), Criteria: entities.BaseItemCriteria{Genre: "Adventure"}},
		{ID: "film-spirited-away", Title: "Spirited Away", MediaType: entities.MediaTypeCinema, AverageRating: 8.6, ReleaseDate: yearsAgo(25), Criteria: entities.BaseItemCriteria{Genre: "Fantasy Anime"}},
	}

	genres := []string{
		"Comedy", "Horror", "Romance", "Thriller", "Documentary", "Biography",
		"Historical", "Western", "Musical", "Sports", "War", "Mystery",
	}
	mediaCycle := []entities.MediaType{
		entities.MediaTypeBook, entities.MediaTypeCinema, entities.MediaTypeGame,
	}
	for i := 0; i < 92; i++ {
		id := fmt.Sprintf("fill-%03d", i+1)
		items = append(items, entities.Item{
			ID:            id,
			Title:         fmt.Sprintf("Catalog filler %03d", i+1),
			MediaType:     mediaCycle[i%len(mediaCycle)],
			AverageRating: 6.4 + float64(i%34)*0.08,
			ReleaseDate:   yearsAgo(22 + (i % 58)),
			Criteria:      entities.BaseItemCriteria{Genre: genres[i%len(genres)]},
		})
	}

	catalog := make(map[string]entities.Item, len(items))
	for _, item := range items {
		catalog[item.ID] = item
	}
	return catalog
}

func itemIDs(items []entities.ScoredItem) []string {
	ids := make([]string, 0, len(items))
	for _, item := range items {
		ids = append(ids, item.ItemID)
	}
	return ids
}

func setOf(ids ...string) map[string]struct{} {
	out := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		out[id] = struct{}{}
	}
	return out
}

func ratio(num int, den int) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den)
}

func safeDivide(num float64, den float64) float64 {
	if den == 0 {
		return 0
	}
	return num / den
}

func boolScore(ok bool) float64 {
	if ok {
		return 1
	}
	return 0
}

func assertAtLeast(t *testing.T, name string, got float64, want float64) {
	t.Helper()
	if got+1e-9 < want {
		t.Fatalf("%s below threshold: got %.3f, want >= %.3f", name, got, want)
	}
}

func TestRecommendationQualityMetricsHelpers(t *testing.T) {
	relevant := setOf("a", "c")
	catalog := map[string]entities.Item{
		"a": {Criteria: entities.BaseItemCriteria{Genre: "Drama"}},
		"b": {Criteria: entities.BaseItemCriteria{Genre: "Sci-Fi"}},
		"c": {Criteria: entities.BaseItemCriteria{Genre: "Drama"}},
	}

	got := evaluateScenario("helpers", []string{"b", "a", "c"}, relevant, catalog, time.Now())

	if got.Precision != float64(2)/3 {
		t.Fatalf("precision: got %v", got.Precision)
	}
	if got.Recall != 1 {
		t.Fatalf("recall: got %v", got.Recall)
	}
	if got.MRR != 0.5 {
		t.Fatalf("mrr: got %v", got.MRR)
	}
	if got.HitRate != 1 {
		t.Fatalf("hit rate: got %v", got.HitRate)
	}
	if got.Diversity != float64(2)/3 {
		t.Fatalf("diversity: got %v", got.Diversity)
	}
	if got.NDCG <= 0 || got.NDCG >= 1 {
		t.Fatalf("ndcg should be between 0 and 1 for delayed hits, got %v", got.NDCG)
	}
}
