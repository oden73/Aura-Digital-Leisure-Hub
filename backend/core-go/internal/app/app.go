package app

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"aura/backend/core-go/internal/config"
	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/domain/services/cf"
	"aura/backend/core-go/internal/domain/services/hybrid"
	"aura/backend/core-go/internal/infrastructure/clients/ai_engine"
	dbpostgres "aura/backend/core-go/internal/infrastructure/db/postgres"
	"aura/backend/core-go/internal/infrastructure/external"
	repopostgres "aura/backend/core-go/internal/infrastructure/repository/postgres"
	"aura/backend/core-go/internal/pkg/auth"
	"aura/backend/core-go/internal/pkg/filter"
	httptransport "aura/backend/core-go/internal/transport/http"
	"aura/backend/core-go/internal/transport/http/handlers"
	"aura/backend/core-go/internal/usecase"
)

// Run bootstraps every dependency and starts the HTTP server.
func Run() error {
	cfg := config.Load()

	db, err := dbpostgres.Connect(context.Background(), cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer db.Close()

	// Infrastructure clients / adapters.
	var aiClient ai_engine.Client = ai_engine.NewHTTPClient(cfg.AIEngineURL, cfg.AIEngineTimeout)
	adapters := map[entities.ExternalService]external.Adapter{
		entities.ExternalServiceSteam:     external.SteamAdapter{},
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
	userSim := cf.UserSimilarityCalculator{Matrix: matrixRepo}
	user2user := cf.User2UserRecommender{
		Similarity:   userSim,
		Neighborhood: cf.UserNeighborhoodBuilder{ThresholdAlpha: 0, Similarity: userSim},
		Predictor:    cf.UserBasedPredictor{Stats: matrixRepo, Matrix: matrixRepo},
	}
	itemSim := cf.ItemSimilarityCalculator{Matrix: matrixRepo, Stats: matrixRepo}
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
		WithMetadata(metadataRepo)

	// Cross-cutting services.
	filterSvc := filter.New()

	// Use cases.
	getRecs := usecase.NewGetRecommendations(orchestrator, userRepo, metadataRepo, filterSvc)
	searchUC := usecase.NewSearchContent(metadataRepo)
	getContentUC := usecase.NewGetContent(metadataRepo)
	upsertContentUC := usecase.NewUpsertContent(metadataRepo)
	updateUC := usecase.NewUpdateInteraction(interactionRepo)
	libraryUC := usecase.NewListLibrary(interactionRepo)
	libraryItemsUC := usecase.NewListLibraryItems(interactionRepo)
	syncUC := usecase.NewSyncExternalContent(adapters, metadataRepo)

	// HTTP transport.
	h := handlers.New(getRecs, searchUC, updateUC, syncUC)
	h.Auth = authHandlers
	h.Users = userRepo
	h.GetContent = getContentUC
	h.UpsertContent = upsertContentUC
	h.Library = libraryUC
	h.LibraryItems = libraryItemsUC
	router := httptransport.NewRouter(h)

	addr := fmt.Sprintf("%s:%d", cfg.HTTPHost, cfg.HTTPPort)
	return http.ListenAndServe(addr, router)
}
