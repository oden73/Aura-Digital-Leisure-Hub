package app

import (
	"fmt"
	"net/http"
	"time"

	"aura/backend/core-go/internal/config"
	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/domain/services/cf"
	"aura/backend/core-go/internal/domain/services/hybrid"
	"aura/backend/core-go/internal/infrastructure/clients/ai_engine"
	"aura/backend/core-go/internal/infrastructure/external"
	"aura/backend/core-go/internal/pkg/auth"
	"aura/backend/core-go/internal/pkg/filter"
	httptransport "aura/backend/core-go/internal/transport/http"
	"aura/backend/core-go/internal/transport/http/handlers"
	"aura/backend/core-go/internal/usecase"
)

// Run bootstraps every dependency and starts the HTTP server.
func Run() error {
	cfg := config.Load()

	// Infrastructure clients / adapters.
	aiClient := ai_engine.StubClient{}
	adapters := map[entities.ExternalService]external.Adapter{
		entities.ExternalServiceSteam:     external.SteamAdapter{},
		entities.ExternalServiceKinopoisk: external.TMDBAdapter{},
		entities.ExternalServiceGoodreads: external.BooksAdapter{},
	}

	userRepo := newMemoryUserRepo()
	var interactionRepo stubInteractionRepo
	var metadataRepo stubMetadataRepo

	tokenMgr := auth.HMACTokenManager{
		Secret:     []byte(cfg.JWTSecret),
		AccessTTL:  15 * time.Minute,
		RefreshTTL: 30 * 24 * time.Hour,
		Issuer:     "aura",
	}
	authSvc := auth.New(tokenMgr, userRepo)
	authHandlers := &handlers.AuthHandlers{Auth: authSvc, Users: userRepo}

	// Domain services.
	cfCoordinator := cf.NewCoordinator(cf.User2UserRecommender{}, cf.Item2ItemRecommender{})
	aggregator := hybrid.NewScoreAggregator(0.5, 0.5)
	ranker := hybrid.NewFinalRanker(
		hybrid.DiversityRule{DiversityThreshold: 0.3},
		hybrid.RecencyBoostRule{DecayFactor: 0.1},
		hybrid.PopularityBalanceRule{},
	)
	orchestrator := hybrid.NewOrchestrator(cfCoordinator, aiClient, aggregator, ranker)

	// Cross-cutting services.
	filterSvc := filter.New()

	// Use cases.
	getRecs := usecase.NewGetRecommendations(orchestrator, userRepo, filterSvc)
	searchUC := usecase.NewSearchContent(metadataRepo)
	updateUC := usecase.NewUpdateInteraction(interactionRepo)
	syncUC := usecase.NewSyncExternalContent(adapters, metadataRepo)

	// HTTP transport.
	h := handlers.New(getRecs, searchUC, updateUC, syncUC)
	h.Auth = authHandlers
	router := httptransport.NewRouter(h)

	addr := fmt.Sprintf("%s:%d", cfg.HTTPHost, cfg.HTTPPort)
	return http.ListenAndServe(addr, router)
}
