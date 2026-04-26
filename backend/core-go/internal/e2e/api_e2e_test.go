//go:build e2e

package e2e

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"
)

// ---------- HTTP helpers ----------------------------------------------------

func (e *testEnv) do(method, path string, body any, bearer string) *http.Response {
	e.t.Helper()
	var reader io.Reader
	if body != nil {
		buf, err := json.Marshal(body)
		if err != nil {
			e.t.Fatalf("marshal body: %v", err)
		}
		reader = bytes.NewReader(buf)
	}
	req, err := http.NewRequest(method, e.server.URL+path, reader)
	if err != nil {
		e.t.Fatalf("new request: %v", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		e.t.Fatalf("send request: %v", err)
	}
	return resp
}

func (e *testEnv) decode(resp *http.Response, into any) {
	e.t.Helper()
	defer resp.Body.Close()
	if into == nil {
		_, _ = io.Copy(io.Discard, resp.Body)
		return
	}
	if err := json.NewDecoder(resp.Body).Decode(into); err != nil {
		e.t.Fatalf("decode response: %v", err)
	}
}

func (e *testEnv) requireStatus(resp *http.Response, want int) {
	e.t.Helper()
	if resp.StatusCode != want {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		e.t.Fatalf("expected status %d, got %d (body=%s)", want, resp.StatusCode, string(body))
	}
}

type tokenPair struct {
	Access  string `json:"access"`
	Refresh string `json:"refresh"`
}

// register creates a user and returns its access/refresh pair.
func (e *testEnv) register(username, email, password string) tokenPair {
	e.t.Helper()
	resp := e.do(http.MethodPost, "/v1/auth/register", map[string]string{
		"username": username,
		"email":    email,
		"password": password,
	}, "")
	e.requireStatus(resp, http.StatusCreated)
	var t tokenPair
	e.decode(resp, &t)
	if t.Access == "" || t.Refresh == "" {
		e.t.Fatalf("register: empty tokens %+v", t)
	}
	return t
}

// ---------- scenarios -------------------------------------------------------

func TestE2E_AuthFlow_RegisterLoginRefreshProfile(t *testing.T) {
	env := setup(t)

	// Register.
	tokens := env.register("alice", "alice@example.com", "hunter2")

	// Profile via the freshly-issued access token.
	resp := env.do(http.MethodGet, "/v1/profile", nil, tokens.Access)
	env.requireStatus(resp, http.StatusOK)
	var profile struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Email    string `json:"email"`
	}
	env.decode(resp, &profile)
	if profile.Username != "alice" || profile.Email != "alice@example.com" {
		t.Fatalf("unexpected profile: %+v", profile)
	}
	if profile.ID == "" {
		t.Fatalf("profile id missing: %+v", profile)
	}

	// Login with the same password should yield NEW tokens.
	resp = env.do(http.MethodPost, "/v1/auth/login", map[string]string{
		"email": "alice@example.com", "password": "hunter2",
	}, "")
	env.requireStatus(resp, http.StatusOK)
	var login tokenPair
	env.decode(resp, &login)
	if login.Access == "" {
		t.Fatal("login: empty access token")
	}

	// Refresh should mint another pair.
	resp = env.do(http.MethodPost, "/v1/auth/refresh", map[string]string{
		"refresh": login.Refresh,
	}, "")
	env.requireStatus(resp, http.StatusOK)
	var refreshed tokenPair
	env.decode(resp, &refreshed)
	if refreshed.Access == "" || refreshed.Refresh == "" {
		t.Fatalf("refresh: empty tokens %+v", refreshed)
	}

	// Bad password → 401.
	resp = env.do(http.MethodPost, "/v1/auth/login", map[string]string{
		"email": "alice@example.com", "password": "wrong",
	}, "")
	env.requireStatus(resp, http.StatusUnauthorized)
}

func TestE2E_Auth_RejectsDuplicateRegistration(t *testing.T) {
	env := setup(t)
	env.register("bob", "bob@example.com", "x")

	resp := env.do(http.MethodPost, "/v1/auth/register", map[string]string{
		"username": "bob2", "email": "bob@example.com", "password": "x",
	}, "")
	env.requireStatus(resp, http.StatusBadRequest)
}

func TestE2E_ProtectedRoutes_RequireBearerToken(t *testing.T) {
	env := setup(t)

	cases := []struct {
		method, path, body string
	}{
		{http.MethodGet, "/v1/profile", ""},
		{http.MethodGet, "/v1/library", ""},
		{http.MethodGet, "/v1/library/items", ""},
		{http.MethodPut, "/v1/interactions", `{"item_id":"x","data":{}}`},
		{http.MethodPost, "/v1/recommendations", `{}`},
		{http.MethodPost, "/v1/content", `{"title":"x","media_type":"book"}`},
		{http.MethodPost, "/v1/external-accounts", `{}`},
	}
	for _, c := range cases {
		t.Run(c.method+" "+c.path, func(t *testing.T) {
			req, _ := http.NewRequest(c.method, env.server.URL+c.path, strings.NewReader(c.body))
			if c.body != "" {
				req.Header.Set("Content-Type", "application/json")
			}
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("request: %v", err)
			}
			defer resp.Body.Close()
			if resp.StatusCode != http.StatusUnauthorized {
				t.Fatalf("expected 401 without bearer, got %d", resp.StatusCode)
			}
		})
	}
}

func TestE2E_HealthAndLivez(t *testing.T) {
	env := setup(t)

	resp := env.do(http.MethodGet, "/livez", nil, "")
	env.requireStatus(resp, http.StatusOK)
	resp.Body.Close()

	// /health falls back to the static probe when no checker is wired.
	resp = env.do(http.MethodGet, "/health", nil, "")
	env.requireStatus(resp, http.StatusOK)
	resp.Body.Close()
}

func TestE2E_ContentCRUD_AndSearch(t *testing.T) {
	env := setup(t)
	tokens := env.register("dora", "dora@example.com", "x")

	// Upsert an item.
	body := map[string]any{
		"title":      "Solaris",
		"media_type": "book",
		"description": "ocean planet",
		"criteria":   map[string]string{"genre": "sci-fi"},
		"book_details": map[string]any{
			"author":     "Lem",
			"page_count": 224,
		},
	}
	resp := env.do(http.MethodPost, "/v1/content", body, tokens.Access)
	env.requireStatus(resp, http.StatusNoContent)
	resp.Body.Close()

	// Search must find it (description match).
	resp = env.do(http.MethodGet, "/v1/search?q=ocean&limit=10", nil, "")
	env.requireStatus(resp, http.StatusOK)
	var hits []map[string]any
	env.decode(resp, &hits)
	if len(hits) != 1 {
		t.Fatalf("expected 1 hit, got %d (%v)", len(hits), hits)
	}
	id, _ := hits[0]["id"].(string)
	if id == "" {
		t.Fatalf("missing id in search hit: %v", hits[0])
	}

	// GET /v1/content/{id} returns the same item with details.
	resp = env.do(http.MethodGet, "/v1/content/"+id, nil, "")
	env.requireStatus(resp, http.StatusOK)
	var item struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		MediaType   string `json:"media_type"`
		BookDetails struct {
			Author    string `json:"author"`
			PageCount int    `json:"page_count"`
		} `json:"book_details"`
	}
	env.decode(resp, &item)
	if item.ID != id || item.Title != "Solaris" {
		t.Fatalf("get content mismatch: %+v", item)
	}
	if item.BookDetails.Author != "Lem" || item.BookDetails.PageCount != 224 {
		t.Fatalf("book details lost: %+v", item.BookDetails)
	}

	// Missing id (handler returns 400).
	resp = env.do(http.MethodGet, "/v1/content/", nil, "")
	// This goes to the / route or 404 from net/http's mux because the
	// pattern `/v1/content/{id}` requires a non-empty id segment. Either
	// way, it must not be 200.
	if resp.StatusCode == http.StatusOK {
		t.Fatalf("missing id returned 200")
	}
	resp.Body.Close()
}

func TestE2E_LibraryFlow_SaveInteractionAndListLibrary(t *testing.T) {
	env := setup(t)
	tokens := env.register("evan", "evan@example.com", "x")

	// Seed a couple of items via the API.
	for _, title := range []string{"Solaris", "Stalker"} {
		mt := "book"
		if title == "Stalker" {
			mt = "cinema"
		}
		resp := env.do(http.MethodPost, "/v1/content", map[string]any{
			"title":      title,
			"media_type": mt,
		}, tokens.Access)
		env.requireStatus(resp, http.StatusNoContent)
		resp.Body.Close()
	}

	// Lookup ids via search.
	type hit struct {
		ID    string `json:"id"`
		Title string `json:"title"`
	}
	resp := env.do(http.MethodGet, "/v1/search?q=&limit=10", nil, "")
	env.requireStatus(resp, http.StatusOK)
	var hits []hit
	env.decode(resp, &hits)
	if len(hits) < 2 {
		t.Fatalf("expected at least 2 items, got %d", len(hits))
	}
	idByTitle := map[string]string{}
	for _, h := range hits {
		idByTitle[h.Title] = h.ID
	}

	// PUT interaction for the first item.
	resp = env.do(http.MethodPut, "/v1/interactions", map[string]any{
		"item_id": idByTitle["Solaris"],
		"data": map[string]any{
			"status":      "completed",
			"rating":      8,
			"is_favorite": true,
		},
	}, tokens.Access)
	env.requireStatus(resp, http.StatusNoContent)
	resp.Body.Close()

	// GET /v1/library returns the interaction.
	resp = env.do(http.MethodGet, "/v1/library", nil, tokens.Access)
	env.requireStatus(resp, http.StatusOK)
	var library []map[string]any
	env.decode(resp, &library)
	if len(library) != 1 {
		t.Fatalf("expected 1 interaction in library, got %d (%v)", len(library), library)
	}
	if library[0]["item_id"] != idByTitle["Solaris"] {
		t.Fatalf("library item mismatch: %v", library[0])
	}
	if library[0]["rating"].(float64) != 8 {
		t.Fatalf("rating not propagated: %v", library[0]["rating"])
	}

	// GET /v1/library/items returns the joined view.
	resp = env.do(http.MethodGet, "/v1/library/items?limit=5", nil, tokens.Access)
	env.requireStatus(resp, http.StatusOK)
	var libItems []struct {
		Item struct {
			Title string `json:"title"`
		} `json:"item"`
		Interaction struct {
			Rating int `json:"rating"`
		} `json:"interaction"`
	}
	env.decode(resp, &libItems)
	if len(libItems) != 1 || libItems[0].Item.Title != "Solaris" || libItems[0].Interaction.Rating != 8 {
		t.Fatalf("unexpected library items: %+v", libItems)
	}
}

func TestE2E_Recommendations_ColdStartFallback(t *testing.T) {
	env := setup(t)
	tokens := env.register("fiona", "fiona@example.com", "x")

	// Seed a couple of items so the cold-start fallback has something
	// to surface — without items the endpoint legitimately returns an
	// empty list.
	for i, title := range []string{"alpha", "beta", "gamma"} {
		_ = i
		resp := env.do(http.MethodPost, "/v1/content", map[string]any{
			"title":          title,
			"media_type":     "book",
			"average_rating": 8.0,
		}, tokens.Access)
		env.requireStatus(resp, http.StatusNoContent)
		resp.Body.Close()
	}

	resp := env.do(http.MethodPost, "/v1/recommendations", map[string]any{
		"filters": map[string]any{},
	}, tokens.Access)
	env.requireStatus(resp, http.StatusOK)
	var rec struct {
		Items []map[string]any `json:"items"`
	}
	env.decode(resp, &rec)
	if len(rec.Items) == 0 {
		t.Fatal("expected cold-start fallback to surface popular items, got 0")
	}
	for _, it := range rec.Items {
		if id, _ := it["item_id"].(string); id == "" {
			t.Fatalf("recommendation missing item_id: %v", it)
		}
	}
}

func TestE2E_LinkExternalAccount_BindsCallerUser(t *testing.T) {
	env := setup(t)
	tokens := env.register("garth", "garth@example.com", "x")

	resp := env.do(http.MethodPost, "/v1/external-accounts", map[string]any{
		"service_name":     "steam",
		"external_user_id": "76561198000000099",
	}, tokens.Access)
	env.requireStatus(resp, http.StatusCreated)
	var acc map[string]any
	env.decode(resp, &acc)
	if acc["service_name"] != "steam" {
		t.Fatalf("service mismatch: %v", acc)
	}
	if id, _ := acc["account_id"].(float64); id == 0 {
		t.Fatalf("account_id missing: %v", acc)
	}

	// Validation: missing external_user_id → 400.
	resp = env.do(http.MethodPost, "/v1/external-accounts", map[string]any{
		"service_name": "steam",
	}, tokens.Access)
	env.requireStatus(resp, http.StatusBadRequest)
	resp.Body.Close()
}
