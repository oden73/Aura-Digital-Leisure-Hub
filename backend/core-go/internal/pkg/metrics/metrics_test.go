package metrics

import (
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPrometheus_HandlerExposesObservedSeries(t *testing.T) {
	rec := New()
	rec.ObserveHTTP("GET", "/v1/recommendations", 200, 12*time.Millisecond)
	rec.IncRecommendationRequest("hybrid", true)
	rec.IncAIEngineCall("compute_cb", false)

	req := httptest.NewRequest("GET", "/metrics", nil)
	w := httptest.NewRecorder()
	rec.Handler().ServeHTTP(w, req)

	body := w.Body.String()
	mustContain(t, body, `aura_http_requests_total{method="GET",route="/v1/recommendations",status="200"} 1`)
	mustContain(t, body, `aura_recommendation_requests_total{cold_start="true",strategy="hybrid"} 1`)
	mustContain(t, body, `aura_ai_engine_calls_total{operation="compute_cb",outcome="error"} 1`)
}

func TestNoOp_ImplementsRecorder(t *testing.T) {
	var r Recorder = NoOp{}
	r.ObserveHTTP("GET", "/", 200, 0)
	r.IncRecommendationRequest("hybrid", false)
	r.IncAIEngineCall("compute_cb", true)
}

func mustContain(t *testing.T, haystack, needle string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("metrics body missing %q\n--- body ---\n%s", needle, haystack)
	}
}
