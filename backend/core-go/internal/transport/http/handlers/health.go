package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"
)

// HealthChecker is anything that can answer "is my dependency reachable
// right now?". Returning a non-nil error marks the dependency as
// unhealthy in the response. Each checker is invoked under a timeout
// derived from the request context.
type HealthChecker interface {
	Name() string
	Check(ctx context.Context) error
}

// CheckerFunc adapts a plain func into a HealthChecker. The name is
// used as the JSON key in the response body and as the failure label
// returned to operators / probes.
type CheckerFunc struct {
	NameValue string
	Fn        func(ctx context.Context) error
}

func (c CheckerFunc) Name() string                    { return c.NameValue }
func (c CheckerFunc) Check(ctx context.Context) error { return c.Fn(ctx) }

// HealthHandler returns a content-aware /health handler that runs every
// checker concurrently and reports per-dependency status. It returns
// 200 OK only when every checker succeeded; any failure flips the
// response to 503 so Kubernetes / load balancers can take the instance
// out of rotation.
//
// When no checkers are supplied the handler degrades to a plain liveness
// probe: 200 OK with {"status":"ok"}. That preserves the previous
// behaviour for tests / dev runs.
func HealthHandler(timeout time.Duration, checkers ...HealthChecker) http.HandlerFunc {
	if timeout <= 0 {
		timeout = 2 * time.Second
	}
	return func(w http.ResponseWriter, r *http.Request) {
		if len(checkers) == 0 {
			writeJSONStatus(w, http.StatusOK, map[string]any{"status": "ok"})
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()

		type result struct {
			name string
			err  error
		}
		results := make([]result, len(checkers))

		var wg sync.WaitGroup
		wg.Add(len(checkers))
		for i, ch := range checkers {
			i, ch := i, ch
			go func() {
				defer wg.Done()
				results[i] = result{name: ch.Name(), err: ch.Check(ctx)}
			}()
		}
		wg.Wait()

		statusCode := http.StatusOK
		checks := make(map[string]any, len(results))
		for _, r := range results {
			if r.err != nil {
				statusCode = http.StatusServiceUnavailable
				checks[r.name] = map[string]any{
					"status": "down",
					"error":  r.err.Error(),
				}
				continue
			}
			checks[r.name] = map[string]any{"status": "up"}
		}
		body := map[string]any{
			"status": map[bool]string{true: "ok", false: "degraded"}[statusCode == http.StatusOK],
			"checks": checks,
		}
		writeJSONStatus(w, statusCode, body)
	}
}

func writeJSONStatus(w http.ResponseWriter, code int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(body)
}
