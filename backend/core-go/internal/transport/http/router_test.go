package http

import (
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/pkg/auth"
	"aura/backend/core-go/internal/pkg/metrics"
	"aura/backend/core-go/internal/pkg/ratelimit"
	"aura/backend/core-go/internal/transport/http/handlers"
	"aura/backend/core-go/internal/usecase"
)

type noopGetRecs struct{}

func (noopGetRecs) Execute(_ string, _ entities.RecommendationFilters) (usecase.RecommendationResponse, error) {
	return usecase.RecommendationResponse{}, nil
}

type noopSearch struct{}

func (noopSearch) Execute(_ usecase.SearchQuery) ([]entities.Item, error) {
	return nil, nil
}

type noopUpdate struct{}

func (noopUpdate) Execute(_, _ string, _ usecase.InteractionData) error {
	return nil
}

type noopSync struct{}

func (noopSync) Execute(_ string, _ entities.ExternalService) (entities.Item, error) {
	return entities.Item{}, nil
}

type stubTokenMgr struct{}

func (stubTokenMgr) Generate(string) (auth.Token, error) {
	return auth.Token{Access: "a", Refresh: "r"}, nil
}

func (stubTokenMgr) Validate(string) (string, error) { return "", errors.New("skip") }

func (stubTokenMgr) Refresh(string) (auth.Token, error) {
	return auth.Token{}, errors.New("skip")
}

type stubUsers struct{}

func (stubUsers) Create(u entities.User) (entities.User, error) { return u, nil }

func (stubUsers) GetByID(string) (entities.User, error) { return entities.User{}, nil }

type stubEmailUsers struct{}

func (stubEmailUsers) GetByEmail(string) (entities.User, error) {
	return entities.User{}, errors.New("not found")
}

func testHandlers(t *testing.T) *handlers.Handlers {
	t.Helper()
	h := handlers.New(noopGetRecs{}, noopSearch{}, noopUpdate{}, noopSync{})
	h.Auth = &handlers.AuthHandlers{
		Auth:  auth.New(stubTokenMgr{}, stubEmailUsers{}),
		Users: stubUsers{},
	}
	return h
}

func TestNewRouter_HealthEndpoints(t *testing.T) {
	h := testHandlers(t)
	var sawCustom bool
	health := func(w http.ResponseWriter, _ *http.Request) {
		sawCustom = true
		w.WriteHeader(http.StatusOK)
	}

	m := metrics.New()
	chain := NewRouter(h, RouterOptions{
		Logger:          slog.New(slog.NewTextHandler(io.Discard, nil)),
		HealthCheck:     health,
		MetricsHandler:  m.Handler(),
		MetricsRecorder: m,
	})

	srv := httptest.NewServer(chain)
	defer srv.Close()

	res, err := http.Get(srv.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if !sawCustom {
		t.Fatal("expected custom health handler")
	}

	res2, err := http.Get(srv.URL + "/livez")
	if err != nil {
		t.Fatal(err)
	}
	_ = res2.Body.Close()
	if res2.StatusCode != http.StatusOK {
		t.Fatalf("livez: %d", res2.StatusCode)
	}

	res3, err := http.Get(srv.URL + "/metrics")
	if err != nil {
		t.Fatal(err)
	}
	_ = res3.Body.Close()
	if res3.StatusCode != http.StatusOK {
		t.Fatalf("metrics: %d", res3.StatusCode)
	}
}

func TestNewRouter_CORSAndRateLimitBranches(t *testing.T) {
	h := testHandlers(t)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	m := metrics.New()

	lim := ratelimit.New(50, 50, time.Minute)

	chain := NewRouter(h, RouterOptions{
		Logger:          logger,
		MetricsRecorder: m,
		CORS: &handlers.CORSConfig{
			Origins:          []string{"https://example.test"},
			AllowCredentials: true,
			ExposeHeaders:    []string{"X-Request-ID"},
			MaxAgeSeconds:    60,
		},
		RateLimit: &handlers.RateLimitConfig{
			Limiter:   lim,
			SkipPaths: []string{"/health"},
		},
	})

	srv := httptest.NewServer(chain)
	defer srv.Close()

	req, err := http.NewRequest(http.MethodGet, srv.URL+"/v1/search?q=x", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Origin", "https://example.test")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_ = res.Body.Close()
	if got := res.Header.Get("Access-Control-Allow-Origin"); got != "https://example.test" {
		t.Fatalf("cors allow-origin = %q", got)
	}
}
