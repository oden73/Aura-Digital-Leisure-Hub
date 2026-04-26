package handlers

import (
	"net/http"
	"strconv"
	"strings"
)

// CORSConfig configures the cross-origin middleware. Origins is the
// allowlist; the literal value "*" allows any origin (only sensible for
// fully public endpoints — never combine with credentials). Methods and
// Headers default to a permissive but safe set.
type CORSConfig struct {
	Origins          []string
	Methods          []string
	Headers          []string
	ExposeHeaders    []string
	AllowCredentials bool
	MaxAgeSeconds    int
}

// CORS returns a middleware that handles CORS preflight and adds the
// appropriate response headers to actual requests. The implementation
// is intentionally small — we don't need every knob from
// github.com/rs/cors, just enough to let the web client talk to the API.
//
// Behaviour:
//   - Requests without an Origin header pass through untouched.
//   - Requests with an Origin matching the allowlist (or "*" wildcard)
//     get Access-Control-Allow-Origin echoing the actual origin so
//     credentialed requests work; the Vary header advertises the
//     dependency on Origin so caches don't poison each other.
//   - Preflight (OPTIONS with Access-Control-Request-Method) responses
//     are answered with the configured methods/headers and a 204 — the
//     downstream handler is *not* invoked.
//   - Disallowed origins still get the response, just without any
//     Allow-Origin header; the browser refuses the response.
func CORS(cfg CORSConfig) func(http.Handler) http.Handler {
	allowAny := false
	allowed := make(map[string]struct{}, len(cfg.Origins))
	for _, o := range cfg.Origins {
		o = strings.TrimSpace(o)
		if o == "" {
			continue
		}
		if o == "*" {
			allowAny = true
			continue
		}
		allowed[o] = struct{}{}
	}

	methods := cfg.Methods
	if len(methods) == 0 {
		methods = []string{
			http.MethodGet, http.MethodPost, http.MethodPut,
			http.MethodPatch, http.MethodDelete, http.MethodOptions,
		}
	}
	headers := cfg.Headers
	if len(headers) == 0 {
		headers = []string{"Content-Type", "Authorization", "X-Request-ID"}
	}

	allowMethods := strings.Join(methods, ", ")
	allowHeaders := strings.Join(headers, ", ")
	exposeHeaders := strings.Join(cfg.ExposeHeaders, ", ")
	maxAge := ""
	if cfg.MaxAgeSeconds > 0 {
		maxAge = strconv.Itoa(cfg.MaxAgeSeconds)
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if origin == "" {
				next.ServeHTTP(w, r)
				return
			}

			allow := allowAny
			if !allow {
				_, allow = allowed[origin]
			}

			h := w.Header()
			// Cache shielding: any response that varies based on the
			// Origin header must say so explicitly.
			h.Add("Vary", "Origin")

			if allow {
				if allowAny && !cfg.AllowCredentials {
					h.Set("Access-Control-Allow-Origin", "*")
				} else {
					h.Set("Access-Control-Allow-Origin", origin)
				}
				if cfg.AllowCredentials {
					h.Set("Access-Control-Allow-Credentials", "true")
				}
				if exposeHeaders != "" {
					h.Set("Access-Control-Expose-Headers", exposeHeaders)
				}
			}

			if r.Method == http.MethodOptions && r.Header.Get("Access-Control-Request-Method") != "" {
				if allow {
					h.Set("Access-Control-Allow-Methods", allowMethods)
					h.Set("Access-Control-Allow-Headers", allowHeaders)
					h.Add("Vary", "Access-Control-Request-Method")
					h.Add("Vary", "Access-Control-Request-Headers")
					if maxAge != "" {
						h.Set("Access-Control-Max-Age", maxAge)
					}
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

