//go:build e2e

// Package e2e wires the full HTTP stack (router + middleware + use cases
// + real Postgres + stub AI engine) and drives it through httptest. Run
// with:
//
//	INTEGRATION_DATABASE_URL=postgres://aura:aura@127.0.0.1:5432/aura_test \
//	  go test -tags=e2e ./internal/e2e/...
//
// CI brings up an ephemeral Postgres service with the same DSN and runs
// migrations from backend/db/migrations before launching this suite.
package e2e

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/domain/services/cf"
	"aura/backend/core-go/internal/domain/services/embeddings"
	"aura/backend/core-go/internal/domain/services/hybrid"
	"aura/backend/core-go/internal/infrastructure/clients/ai_engine"
	dbpostgres "aura/backend/core-go/internal/infrastructure/db/postgres"
	repopostgres "aura/backend/core-go/internal/infrastructure/repository/postgres"
	"aura/backend/core-go/internal/pkg/auth"
	"aura/backend/core-go/internal/pkg/filter"
	"aura/backend/core-go/internal/pkg/simcache"
	httptransport "aura/backend/core-go/internal/transport/http"
	"aura/backend/core-go/internal/transport/http/handlers"
	"aura/backend/core-go/internal/usecase"
)

// testEnv is the assembled server state shared by tests.
type testEnv struct {
	t      *testing.T
	server *httptest.Server
	pool   *dbpostgres.Pool
}

var (
	sharedPool   *dbpostgres.Pool
	sharedPoolMu sync.Mutex
)

// setup wires every dependency the way `internal/app.Run` does, but
// against the ephemeral Postgres pointed at by $INTEGRATION_DATABASE_URL
// and a no-op AI engine. The returned httptest.Server is owned by the
// test and torn down via t.Cleanup.
func setup(t *testing.T) *testEnv {
	t.Helper()
	pool := mustGetPool(t)
	resetTables(t, pool)

	aiClient := ai_engine.StubClient{}
	embeddingPublisher := embeddings.New(aiClient)

	userRepo := repopostgres.NewUserRepo(pool)
	interactionRepo := repopostgres.NewInteractionRepo(pool)
	metadataRepo := repopostgres.NewMetadataRepo(pool)
	matrixRepo := repopostgres.NewInteractionMatrixRepo(pool)

	tokenMgr := auth.HMACTokenManager{
		Secret:     []byte("e2e-test-secret-key"),
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 30 * 24 * time.Hour,
		Issuer:     "aura-e2e",
	}
	authSvc := auth.New(tokenMgr, userRepo)
	authHandlers := &handlers.AuthHandlers{Auth: authSvc, Users: userRepo}

	userSimCache := simcache.New(time.Minute, 1024)
	itemSimCache := simcache.New(time.Minute, 1024)

	userSim := cf.UserSimilarityCalculator{Matrix: matrixRepo, Cache: userSimCache}
	user2user := cf.User2UserRecommender{
		Similarity:   userSim,
		Neighborhood: cf.UserNeighborhoodBuilder{ThresholdAlpha: 0, Similarity: userSim},
		Predictor:    cf.UserBasedPredictor{Stats: matrixRepo, Matrix: matrixRepo},
	}
	itemSim := cf.ItemSimilarityCalculator{Matrix: matrixRepo, Stats: matrixRepo, Cache: itemSimCache}
	item2item := cf.Item2ItemRecommender{
		Similarity:   itemSim,
		Neighborhood: cf.ItemNeighborhoodBuilder{ThresholdBeta: 0, Similarity: itemSim},
		Predictor:    cf.ItemBasedPredictor{Matrix: matrixRepo},
	}
	cfCoordinator := cf.NewCoordinator(user2user, item2item).
		WithCandidates(matrixRepo).
		WithMatrix(matrixRepo).
		WithStats(matrixRepo)

	aggregator := hybrid.NewScoreAggregator(0.5, 0.5)
	ranker := hybrid.NewFinalRanker(
		hybrid.DiversityRule{DiversityThreshold: 0.3},
		hybrid.RecencyBoostRule{DecayFactor: 0.1},
		hybrid.PopularityBalanceRule{},
	)
	orchestrator := hybrid.NewOrchestrator(cfCoordinator, aiClient, aggregator, ranker).
		WithMetadata(metadataRepo).
		WithProfiles(userRepo)

	filterSvc := filter.New().WithMetadata(metadataRepo)

	getRecs := usecase.NewGetRecommendations(orchestrator, userRepo, metadataRepo, filterSvc).
		WithPopularity(metadataRepo)
	searchUC := usecase.NewSearchContent(metadataRepo)
	getContentUC := usecase.NewGetContent(metadataRepo)
	upsertContentUC := usecase.NewUpsertContent(metadataRepo, embeddingPublisher)
	updateUC := usecase.NewUpdateInteraction(interactionRepo).
		WithCacheInvalidation(userSimCache, itemSimCache)
	libraryUC := usecase.NewListLibrary(interactionRepo)
	libraryItemsUC := usecase.NewListLibraryItems(interactionRepo)
	syncUC := usecase.NewSyncExternalContent(nil, metadataRepo, embeddingPublisher)
	linkExternalUC := usecase.NewLinkExternalAccount(userRepo)

	h := handlers.New(getRecs, searchUC, updateUC, syncUC)
	h.Auth = authHandlers
	h.Users = userRepo
	h.GetContent = getContentUC
	h.UpsertContent = upsertContentUC
	h.Library = libraryUC
	h.LibraryItems = libraryItemsUC
	h.LinkExternalAccount = linkExternalUC

	router := httptransport.NewRouter(h, httptransport.RouterOptions{})
	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return &testEnv{t: t, server: srv, pool: pool}
}

// mustGetPool reuses the same convention as the postgres integration
// setup: a process-wide pool is created on first use and migrations are
// applied once. Run-to-run isolation comes from resetTables in setup().
func mustGetPool(t *testing.T) *dbpostgres.Pool {
	t.Helper()
	dsn := os.Getenv("INTEGRATION_DATABASE_URL")
	if dsn == "" {
		t.Skip("INTEGRATION_DATABASE_URL not set; skipping e2e tests")
	}

	sharedPoolMu.Lock()
	defer sharedPoolMu.Unlock()

	if sharedPool != nil {
		return sharedPool
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	p, err := dbpostgres.Connect(ctx, dsn)
	if err != nil {
		t.Fatalf("connect %s: %v", dsn, err)
	}
	if err := applyMigrations(ctx, p); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	sharedPool = p
	return p
}

func applyMigrations(ctx context.Context, p *dbpostgres.Pool) error {
	dir := findMigrationsDir()
	if dir == "" {
		return errors.New("backend/db/migrations not found")
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	var files []string
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	sort.Strings(files)
	for _, f := range files {
		body, err := os.ReadFile(f)
		if err != nil {
			return err
		}
		if _, err := p.Exec(ctx, string(body)); err != nil {
			if strings.Contains(err.Error(), "already exists") {
				continue
			}
			return fmt.Errorf("apply %s: %w", f, err)
		}
	}
	return nil
}

func findMigrationsDir() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	dir := cwd
	for {
		candidate := filepath.Join(dir, "db", "migrations")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func resetTables(t *testing.T, p *dbpostgres.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if _, err := p.Exec(ctx, `
		TRUNCATE
			external_accounts,
			user_interactions,
			vector_store,
			book_details,
			cinema_details,
			game_details,
			base_items,
			users
		RESTART IDENTITY CASCADE
	`); err != nil {
		t.Fatalf("truncate: %v", err)
	}
}

// Compile-time guard: keep the import of entities live for tests that
// use the package (every test file does, but go vet on this file alone
// must still pass).
var _ entities.MediaType = entities.MediaTypeBook
