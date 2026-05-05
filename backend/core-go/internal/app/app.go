package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"aura/backend/core-go/internal/config"
	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/domain/services/cf"
	"aura/backend/core-go/internal/domain/services/embeddings"
	"aura/backend/core-go/internal/domain/services/hybrid"
	"aura/backend/core-go/internal/infrastructure/clients/ai_engine"
	dbpostgres "aura/backend/core-go/internal/infrastructure/db/postgres"
	"aura/backend/core-go/internal/infrastructure/external"
	repopostgres "aura/backend/core-go/internal/infrastructure/repository/postgres"
	"aura/backend/core-go/internal/pkg/auth"
	"aura/backend/core-go/internal/pkg/filter"
	"aura/backend/core-go/internal/pkg/logging"
	"aura/backend/core-go/internal/pkg/metrics"
	"aura/backend/core-go/internal/pkg/ratelimit"
	"aura/backend/core-go/internal/pkg/simcache"
	httptransport "aura/backend/core-go/internal/transport/http"
	"aura/backend/core-go/internal/transport/http/handlers"
	"aura/backend/core-go/internal/usecase"
)

// Run bootstraps every dependency and starts the HTTP server.
func Run() error {
	cfg := config.Load()

	logger := logging.New(cfg.Environment)
	logging.SetDefault(logger)
	logger.Info("starting", "env", cfg.Environment, "addr", fmt.Sprintf("%s:%d", cfg.HTTPHost, cfg.HTTPPort))

	metricsRecorder := metrics.New()

	db, err := dbpostgres.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		logger.Error("db_connect_failed", "error", err)
		return err
	}
	defer db.Close()

	// Infrastructure clients / adapters.
	aiHTTP := ai_engine.NewHTTPClient(cfg.AIEngineURL, cfg.AIEngineTimeout).WithMetrics(metricsRecorder)
	var aiClient ai_engine.Client = aiHTTP
	embeddingPublisher := embeddings.New(aiClient)
	adapters := map[entities.ExternalService]external.Adapter{
		entities.ExternalServiceSteam:     external.SteamAdapter{APIKey: cfg.SteamAPIKey},
		entities.ExternalServiceKinopoisk: external.TMDBAdapter{},
		entities.ExternalServiceGoodreads: external.BooksAdapter{},
	}

	userRepo := repopostgres.NewUserRepo(db)
	interactionRepo := repopostgres.NewInteractionRepo(db)
	metadataRepo := repopostgres.NewMetadataRepo(db)
	matrixRepo := repopostgres.NewInteractionMatrixRepo(db)

	tokenMgr := auth.HMACTokenManager{
		Secret:     []byte(cfg.JWTSecret),
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 30 * 24 * time.Hour,
		Issuer:     "aura",
	}
	authSvc := auth.New(tokenMgr, userRepo)
	authHandlers := &handlers.AuthHandlers{Auth: authSvc, Users: userRepo}

	// Domain services: collaborative filtering pipeline.
	// Similarity caches are sized to comfortably fit a few thousand
	// active users / popular items at once and expire after 30 minutes;
	// any rating change invalidates the affected slots immediately via
	// UpdateInteraction.WithCacheInvalidation.
	userSimCache := simcache.New(cfg.SimilarityCacheTTL, cfg.UserSimilarityCacheMaxEntries)
	itemSimCache := simcache.New(cfg.SimilarityCacheTTL, cfg.ItemSimilarityCacheMaxEntries)

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

	// Cross-cutting services.
	filterSvc := filter.New().WithMetadata(metadataRepo)

	// Use cases.
	getRecs := usecase.NewGetRecommendations(orchestrator, userRepo, metadataRepo, filterSvc).
		WithPopularity(metadataRepo).
		WithMetrics(metricsRecorder)
	searchUC := usecase.NewSearchContent(metadataRepo)
	getContentUC := usecase.NewGetContent(metadataRepo)
	upsertContentUC := usecase.NewUpsertContent(metadataRepo, embeddingPublisher)
	updateUC := usecase.NewUpdateInteraction(interactionRepo).
		WithCacheInvalidation(userSimCache, itemSimCache)
	libraryUC := usecase.NewListLibrary(interactionRepo)
	libraryItemsUC := usecase.NewListLibraryItems(interactionRepo)
	syncUC := usecase.NewSyncExternalContent(adapters, metadataRepo, embeddingPublisher)
	linkExternalUC := usecase.NewLinkExternalAccount(userRepo)

	// HTTP transport.
	h := handlers.New(getRecs, searchUC, updateUC, syncUC)
	h.Auth = authHandlers
	h.AIClient = aiClient
	h.Users = userRepo
	h.GetContent = getContentUC
	h.UpsertContent = upsertContentUC
	h.Library = libraryUC
	h.LibraryItems = libraryItemsUC
	h.LinkExternalAccount = linkExternalUC
	healthHandler := handlers.HealthHandler(2*time.Second,
		handlers.CheckerFunc{
			NameValue: "database",
			Fn: func(ctx context.Context) error {
				return db.Ping(ctx)
			},
		},
		handlers.CheckerFunc{
			NameValue: "ai_engine",
			Fn: func(ctx context.Context) error {
				return checkAIEngine(ctx, cfg.AIEngineURL, cfg.AIEngineTimeout)
			},
		},
	)

	routerOpts := httptransport.RouterOptions{
		Logger:          logger,
		HealthCheck:     healthHandler,
		MetricsHandler:  metricsRecorder.Handler(),
		MetricsRecorder: metricsRecorder,
	}
	if len(cfg.CORSAllowedOrigins) > 0 {
		routerOpts.CORS = &handlers.CORSConfig{
			Origins:          cfg.CORSAllowedOrigins,
			AllowCredentials: true,
			ExposeHeaders:    []string{"X-Request-ID"},
			MaxAgeSeconds:    600,
		}
	}

	var limiter *ratelimit.Limiter
	if cfg.RateLimitRPS > 0 {
		// 5-minute idle window keeps memory bounded under churn (NAT'd
		// users coming and going) without evicting genuinely active
		// clients — at 20rps a single user touches their bucket far
		// more often than once every 5 minutes.
		limiter = ratelimit.New(cfg.RateLimitRPS, cfg.RateLimitBurst, 5*time.Minute)
		routerOpts.RateLimit = &handlers.RateLimitConfig{
			Limiter:   limiter,
			SkipPaths: []string{"/health", "/livez", "/metrics"},
		}
	}

	router := httptransport.NewRouter(h, routerOpts)

	addr := fmt.Sprintf("%s:%d", cfg.HTTPHost, cfg.HTTPPort)
	// Conservative server-side timeouts. They are deliberately wider than
	// the AI engine timeout (cfg.AIEngineTimeout, ~2s by default) plus
	// database round-trip headroom — we want slow clients to be cut off
	// without breaking legitimate end-to-end recommendation calls.
	srv := &http.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Shut down on SIGINT/SIGTERM so Kubernetes / docker stop / Ctrl-C
	// give in-flight requests a chance to finish before we close the
	// database pool. A hard timeout caps how long we are willing to
	// wait so deployments don't hang on stuck connections.
	shutdownTimeout := cfg.ShutdownTimeout
	if shutdownTimeout <= 0 {
		shutdownTimeout = 15 * time.Second
	}

	serveErr := make(chan error, 1)
	go func() {
		logger.Info("http_listen", "addr", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serveErr <- err
			return
		}
		serveErr <- nil
	}()

	// Background sweeper for the rate-limit map so idle buckets are
	// eventually freed; uses a context so it stops on shutdown.
	sweeperCtx, stopSweeper := context.WithCancel(context.Background())
	defer stopSweeper()
	if limiter != nil {
		go runRateLimitSweeper(sweeperCtx, limiter, time.Minute)
	}

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)

	select {
	case sig := <-stop:
		logger.Info("shutdown_signal", "signal", sig.String(), "timeout", shutdownTimeout.String())
	case err := <-serveErr:
		if err != nil {
			logger.Error("http_listen_failed", "error", err)
			return err
		}
		// ListenAndServe returned with a clean ErrServerClosed before any
		// signal — unusual, treat it as a normal shutdown.
		return nil
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Error("http_shutdown_failed", "error", err)
		_ = srv.Close()
		return err
	}
	// Wait for ListenAndServe to actually return before reporting success.
	if err := <-serveErr; err != nil {
		return err
	}
	logger.Info("shutdown_complete")
	return nil
}

// runRateLimitSweeper periodically prunes idle buckets from the limiter
// map so memory stays bounded across long-running deployments. It exits
// when ctx is cancelled (driven by app shutdown).
func runRateLimitSweeper(ctx context.Context, l *ratelimit.Limiter, every time.Duration) {
	ticker := time.NewTicker(every)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.Sweep()
		}
	}
}

// checkAIEngine probes the Python AI engine /health endpoint. We do this
// inline (instead of adding it to ai_engine.Client) because the health
// check intentionally bypasses the request metrics and TLS-aware
// transport — it's a reachability test, not a real call.
func checkAIEngine(ctx context.Context, baseURL string, timeout time.Duration) error {
	if baseURL == "" {
		return errors.New("ai engine url not configured")
	}
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	probeCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(probeCtx, http.MethodGet, baseURL+"/health", nil)
	if err != nil {
		return err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return fmt.Errorf("ai engine: status=%d", resp.StatusCode)
	}
	return nil
}
