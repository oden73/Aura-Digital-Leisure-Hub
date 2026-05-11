//go:build e2e

package e2e

import (
	"fmt"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const onlineRecommendationQualityK = 5

type onlineQualityCatalogItem struct {
	Title     string
	MediaType string
	Rating    float64
	Genre     string
	Mood      string
}

type onlineQualityScenario struct {
	Name           string
	Username       string
	Email          string
	Filters        map[string]any
	RelevantTitles []string
}

type onlineQualityScenarioResult struct {
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

type onlineQualitySummary struct {
	PrecisionAtK    float64
	RecallAtK       float64
	NDCGAtK         float64
	MRRAtK          float64
	HitRateAtK      float64
	CoverageAtK     float64
	GenreDiversity  float64
	TotalDurationMS float64
}

func TestE2E_RecommendationOnlineQualityReport(t *testing.T) {
	env := setup(t)
	tokens := env.register("quality-seed", "quality-seed@example.com", "x")

	catalog := onlineQualityCatalog()
	idByTitle := seedOnlineQualityCatalog(t, env, tokens.Access, catalog)
	catalogByID := map[string]onlineQualityCatalogItem{}
	for _, item := range catalog {
		catalogByID[idByTitle[item.Title]] = item
	}

	scenarios := []onlineQualityScenario{
		{
			Name:     "new_user_all_media",
			Username: "quality-new",
			Email:    "quality-new@example.com",
			Filters:  map[string]any{},
			RelevantTitles: []string{
				"Dune",
				"The Legend of Zelda",
				"Mass Effect",
				"Arrival",
				"Foundation",
			},
		},
		{
			Name:     "books_only",
			Username: "quality-books",
			Email:    "quality-books@example.com",
			Filters:  map[string]any{"media_types": []string{"book"}},
			RelevantTitles: []string{
				"Dune",
				"Foundation",
				"Hyperion",
				"A Wizard of Earthsea",
				"Neuromancer",
			},
		},
		{
			Name:     "dark_mood",
			Username: "quality-dark",
			Email:    "quality-dark@example.com",
			Filters:  map[string]any{"moods": []string{"dark"}},
			RelevantTitles: []string{
				"Blade Runner",
				"Ghost in the Shell",
				"Deus Ex",
				"Control",
				"Neuromancer",
			},
		},
		{
			Name:     "games_only",
			Username: "quality-games",
			Email:    "quality-games@example.com",
			Filters:  map[string]any{"media_types": []string{"game"}},
			RelevantTitles: []string{
				"The Legend of Zelda",
				"Mass Effect",
				"Deus Ex",
				"Control",
			},
		},
	}

	results := make([]onlineQualityScenarioResult, 0, len(scenarios))
	recommended := map[string]struct{}{}
	started := time.Now()
	for _, scenario := range scenarios {
		userTokens := env.register(scenario.Username, scenario.Email, "x")
		relevant := titleSetToIDSet(t, idByTitle, scenario.RelevantTitles)
		scenarioStart := time.Now()
		ids := requestOnlineRecommendations(t, env, userTokens.Access, scenario.Filters, onlineRecommendationQualityK)
		for _, id := range ids {
			recommended[id] = struct{}{}
		}
		results = append(results, evaluateOnlineScenario(
			scenario.Name,
			ids,
			relevant,
			catalogByID,
			scenarioStart,
		))
	}

	summary := summarizeOnlineQuality(results, len(recommended), len(catalog), time.Since(started))
	reportPath, err := writeRecommendationOnlineQualityReport(summary, results, onlineRecommendationQualityK)
	if err != nil {
		t.Fatalf("write online quality report: %v", err)
	}
	t.Logf("recommendation online quality report: %s", reportPath)

	assertOnlineAtLeast(t, "precision@5", summary.PrecisionAtK, 0.28)
	assertOnlineAtLeast(t, "recall@5", summary.RecallAtK, 0.25)
	assertOnlineAtLeast(t, "ndcg@5", summary.NDCGAtK, 0.30)
	assertOnlineAtLeast(t, "mrr@5", summary.MRRAtK, 0.40)
	assertOnlineAtLeast(t, "hit_rate@5", summary.HitRateAtK, 0.75)
	assertOnlineAtLeast(t, "catalog_coverage@5", summary.CoverageAtK, 0.06)
	assertOnlineAtLeast(t, "genre_diversity@5", summary.GenreDiversity, 0.28)
}

func seedOnlineQualityCatalog(
	t *testing.T,
	env *testEnv,
	accessToken string,
	catalog []onlineQualityCatalogItem,
) map[string]string {
	t.Helper()
	for _, item := range catalog {
		resp := env.do(http.MethodPost, "/v1/content", map[string]any{
			"title":          item.Title,
			"media_type":     item.MediaType,
			"average_rating": item.Rating,
			"criteria": map[string]string{
				"genre":    item.Genre,
				"tonality": item.Mood,
			},
		}, accessToken)
		env.requireStatus(resp, http.StatusNoContent)
		resp.Body.Close()
	}

	resp := env.do(http.MethodGet, fmt.Sprintf("/v1/search?q=&limit=%d", len(catalog)+5), nil, "")
	env.requireStatus(resp, http.StatusOK)
	var hits []struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	env.decode(resp, &hits)

	idByTitle := map[string]string{}
	for _, hit := range hits {
		idByTitle[hit.Title] = hit.ID
	}
	for _, item := range catalog {
		if idByTitle[item.Title] == "" {
			t.Fatalf("seeded item %q not found via search", item.Title)
		}
	}
	return idByTitle
}

func requestOnlineRecommendations(
	t *testing.T,
	env *testEnv,
	accessToken string,
	filters map[string]any,
	k int,
) []string {
	t.Helper()
	resp := env.do(http.MethodPost, "/v1/recommendations", map[string]any{
		"filters": filters,
	}, accessToken)
	env.requireStatus(resp, http.StatusOK)
	var rec struct {
		Items []struct {
			ItemID string `json:"item_id"`
		} `json:"items"`
	}
	env.decode(resp, &rec)
	if len(rec.Items) == 0 {
		t.Fatalf("expected online recommendations, got empty response")
	}
	ids := make([]string, 0, len(rec.Items))
	for _, item := range rec.Items {
		if item.ItemID == "" {
			t.Fatalf("recommendation missing item_id: %+v", item)
		}
		ids = append(ids, item.ItemID)
	}
	if len(ids) > k {
		return ids[:k]
	}
	return ids
}

func evaluateOnlineScenario(
	name string,
	ids []string,
	relevant map[string]struct{},
	catalog map[string]onlineQualityCatalogItem,
	started time.Time,
) onlineQualityScenarioResult {
	hits := 0
	dcg := 0.0
	firstRelevantRank := 0
	genres := map[string]struct{}{}

	for i, id := range ids {
		if item, ok := catalog[id]; ok {
			genre := strings.ToLower(strings.TrimSpace(item.Genre))
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

	result := onlineQualityScenarioResult{
		Name:        name,
		IDs:         ids,
		Precision:   onlineRatio(hits, len(ids)),
		Recall:      onlineRatio(hits, len(relevant)),
		NDCG:        onlineSafeDivide(dcg, idcg),
		HitRate:     onlineBoolScore(hits > 0),
		Diversity:   onlineRatio(len(genres), len(ids)),
		DurationMS:  float64(time.Since(started).Microseconds()) / 1000,
		RelevantHit: hits,
	}
	if firstRelevantRank > 0 {
		result.MRR = 1 / float64(firstRelevantRank)
	}
	return result
}

func summarizeOnlineQuality(
	results []onlineQualityScenarioResult,
	recommendedCount int,
	catalogSize int,
	duration time.Duration,
) onlineQualitySummary {
	var summary onlineQualitySummary
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
	summary.CoverageAtK = onlineRatio(recommendedCount, catalogSize)
	summary.TotalDurationMS = float64(duration.Microseconds()) / 1000
	return summary
}

func writeRecommendationOnlineQualityReport(
	summary onlineQualitySummary,
	results []onlineQualityScenarioResult,
	k int,
) (string, error) {
	root, err := findOnlineQualityBackendRoot()
	if err != nil {
		return "", err
	}
	reportsDir := filepath.Join(root, "reports")
	if err := os.MkdirAll(reportsDir, 0o755); err != nil {
		return "", err
	}

	reportPath := filepath.Join(reportsDir, "recommendation_online_quality_report.md")
	var b strings.Builder
	fmt.Fprintf(&b, "# Recommendation online quality report\n\n")
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

func findOnlineQualityBackendRoot() (string, error) {
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

func onlineQualityCatalog() []onlineQualityCatalogItem {
	core := []onlineQualityCatalogItem{
		{Title: "Dune", MediaType: "book", Rating: 9.8, Genre: "Sci-Fi", Mood: "epic"},
		{Title: "The Legend of Zelda", MediaType: "game", Rating: 9.7, Genre: "Adventure", Mood: "bright"},
		{Title: "Mass Effect", MediaType: "game", Rating: 9.6, Genre: "RPG", Mood: "epic"},
		{Title: "Arrival", MediaType: "cinema", Rating: 9.5, Genre: "Sci-Fi Drama", Mood: "thoughtful"},
		{Title: "Foundation", MediaType: "book", Rating: 9.4, Genre: "Classic Sci-Fi", Mood: "epic"},
		{Title: "Blade Runner", MediaType: "cinema", Rating: 9.3, Genre: "Neo-noir", Mood: "dark"},
		{Title: "Hyperion", MediaType: "book", Rating: 9.2, Genre: "Space Opera", Mood: "epic"},
		{Title: "Ghost in the Shell", MediaType: "cinema", Rating: 9.1, Genre: "Cyberpunk Anime", Mood: "dark"},
		{Title: "A Wizard of Earthsea", MediaType: "book", Rating: 9.0, Genre: "Fantasy", Mood: "bright"},
		{Title: "Deus Ex", MediaType: "game", Rating: 8.9, Genre: "Immersive Sim", Mood: "dark"},
		{Title: "Neuromancer", MediaType: "book", Rating: 8.8, Genre: "Cyberpunk", Mood: "dark"},
		{Title: "Control", MediaType: "game", Rating: 8.7, Genre: "Action Adventure", Mood: "dark"},
		{Title: "The Matrix", MediaType: "cinema", Rating: 8.6, Genre: "Action Sci-Fi", Mood: "dark"},
		{Title: "Spirited Away", MediaType: "cinema", Rating: 8.5, Genre: "Fantasy Anime", Mood: "bright"},
	}
	genres := []string{
		"Thriller", "Comedy", "Horror", "Romance", "Documentary", "Biography",
		"Historical", "Western", "Musical", "Sports", "War", "Mystery",
	}
	moods := []string{"bright", "dark", "thoughtful", "epic", "neutral"}
	mt := []string{"book", "cinema", "game"}
	for i := 0; i < 90; i++ {
		r := 7.0 + float64(i%32)*0.06
		if i%9 == 0 {
			r = 9.0 + float64(i%10)*0.05
		}
		core = append(core, onlineQualityCatalogItem{
			Title:     fmt.Sprintf("Catalog noise title %04d", i+1),
			MediaType: mt[i%len(mt)],
			Rating:    r,
			Genre:     genres[i%len(genres)],
			Mood:      moods[i%len(moods)],
		})
	}
	return core
}

func titleSetToIDSet(t *testing.T, idByTitle map[string]string, titles []string) map[string]struct{} {
	t.Helper()
	out := make(map[string]struct{}, len(titles))
	for _, title := range titles {
		id := idByTitle[title]
		if id == "" {
			t.Fatalf("missing id for relevant title %q", title)
		}
		out[id] = struct{}{}
	}
	return out
}

func onlineRatio(num int, den int) float64 {
	if den == 0 {
		return 0
	}
	return float64(num) / float64(den)
}

func onlineSafeDivide(num float64, den float64) float64 {
	if den == 0 {
		return 0
	}
	return num / den
}

func onlineBoolScore(ok bool) float64 {
	if ok {
		return 1
	}
	return 0
}

func assertOnlineAtLeast(t *testing.T, name string, got float64, want float64) {
	t.Helper()
	if got+1e-9 < want {
		t.Fatalf("%s below threshold: got %.3f, want >= %.3f", name, got, want)
	}
}
