// Package metrics owns the Prometheus registry for the Go core. It
// exposes a small Recorder interface so tests and ad-hoc tools can opt
// out of Prometheus, and a single Handler() entrypoint that scraping
// targets can wire directly into the HTTP router.
package metrics

import (
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Recorder is the surface used by the rest of the codebase. Keeping a
// narrow interface (rather than scattering raw Prometheus calls
// everywhere) makes it trivial to plug in a no-op recorder for unit
// tests, and keeps the metric vocabulary documented in one place.
type Recorder interface {
	// ObserveHTTP records a finished HTTP request. The route argument
	// should be the matched mux pattern (e.g. "GET /v1/content/{id}"),
	// not the raw URL path, to keep metric cardinality bounded.
	ObserveHTTP(method, route string, status int, duration time.Duration)

	// IncRecommendationRequest is incremented every time the
	// recommendation use case returns a response. cold_start is true
	// when the response was supplemented by the popular-items fallback.
	IncRecommendationRequest(strategy string, coldStart bool)

	// IncAIEngineCall tracks outbound calls to the Python AI engine.
	IncAIEngineCall(operation string, success bool)
}

// Prometheus is the production Recorder backed by client_golang. The
// zero value is not usable; construct one with New.
type Prometheus struct {
	registry          *prometheus.Registry
	httpRequestsTotal *prometheus.CounterVec
	httpDuration      *prometheus.HistogramVec
	recCounter        *prometheus.CounterVec
	aiCounter         *prometheus.CounterVec
}

// New builds a Prometheus recorder with its own registry. We deliberately
// do not use prometheus.DefaultRegisterer so tests can construct an
// isolated instance and so a panic during registration cannot affect
// unrelated parts of the process.
func New() *Prometheus {
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	httpReq := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "aura_http_requests_total",
		Help: "Total number of HTTP requests handled by the Go core.",
	}, []string{"method", "route", "status"})
	httpDur := prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "aura_http_request_duration_seconds",
		Help:    "Latency of HTTP requests handled by the Go core.",
		Buckets: prometheus.DefBuckets,
	}, []string{"method", "route", "status"})
	rec := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "aura_recommendation_requests_total",
		Help: "Recommendation responses produced, partitioned by strategy and cold-start flag.",
	}, []string{"strategy", "cold_start"})
	ai := prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "aura_ai_engine_calls_total",
		Help: "Outbound calls to the Python AI engine, partitioned by operation and outcome.",
	}, []string{"operation", "outcome"})

	reg.MustRegister(httpReq, httpDur, rec, ai)

	return &Prometheus{
		registry:          reg,
		httpRequestsTotal: httpReq,
		httpDuration:      httpDur,
		recCounter:        rec,
		aiCounter:         ai,
	}
}

// Handler returns the http.Handler that exposes /metrics. It is bound to
// our isolated registry, not the global one.
func (p *Prometheus) Handler() http.Handler {
	return promhttp.HandlerFor(p.registry, promhttp.HandlerOpts{})
}

// Registry exposes the underlying registry so additional collectors
// (e.g. database/sql stats) can be registered alongside the defaults.
func (p *Prometheus) Registry() *prometheus.Registry { return p.registry }

func (p *Prometheus) ObserveHTTP(method, route string, status int, duration time.Duration) {
	if p == nil {
		return
	}
	statusLabel := strconv.Itoa(status)
	p.httpRequestsTotal.WithLabelValues(method, route, statusLabel).Inc()
	p.httpDuration.WithLabelValues(method, route, statusLabel).Observe(duration.Seconds())
}

func (p *Prometheus) IncRecommendationRequest(strategy string, coldStart bool) {
	if p == nil {
		return
	}
	p.recCounter.WithLabelValues(strategy, boolLabel(coldStart)).Inc()
}

func (p *Prometheus) IncAIEngineCall(operation string, success bool) {
	if p == nil {
		return
	}
	outcome := "error"
	if success {
		outcome = "ok"
	}
	p.aiCounter.WithLabelValues(operation, outcome).Inc()
}

// NoOp implements Recorder without doing anything; useful as a default
// dependency in tests so callers don't have to special-case nil.
type NoOp struct{}

func (NoOp) ObserveHTTP(string, string, int, time.Duration) {}
func (NoOp) IncRecommendationRequest(string, bool)          {}
func (NoOp) IncAIEngineCall(string, bool)                   {}

func boolLabel(b bool) string {
	if b {
		return "true"
	}
	return "false"
}
