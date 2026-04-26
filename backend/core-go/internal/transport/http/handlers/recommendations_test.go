package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"aura/backend/core-go/internal/domain/entities"
	repopostgres "aura/backend/core-go/internal/infrastructure/repository/postgres"
	"aura/backend/core-go/internal/usecase"
)

// ---------- fakes -----------------------------------------------------------

type fakeGetRecs struct {
	in     string
	resp   usecase.RecommendationResponse
	err    error
	called bool
}

func (f *fakeGetRecs) Execute(uid string, _ entities.RecommendationFilters) (usecase.RecommendationResponse, error) {
	f.called = true
	f.in = uid
	return f.resp, f.err
}

type fakeSearch struct {
	in   usecase.SearchQuery
	resp []entities.Item
	err  error
}

func (f *fakeSearch) Execute(q usecase.SearchQuery) ([]entities.Item, error) {
	f.in = q
	return f.resp, f.err
}

type fakeGetContent struct {
	in   string
	resp entities.Item
	err  error
}

func (f *fakeGetContent) Execute(id string) (entities.Item, error) {
	f.in = id
	return f.resp, f.err
}

type fakeUpsertContent struct {
	in  entities.Item
	err error
}

func (f *fakeUpsertContent) Execute(it entities.Item) error {
	f.in = it
	return f.err
}

type fakeUpdateInt struct {
	uid    string
	itemID string
	data   usecase.InteractionData
	err    error
}

func (f *fakeUpdateInt) Execute(uid string, itemID string, data usecase.InteractionData) error {
	f.uid = uid
	f.itemID = itemID
	f.data = data
	return f.err
}

type fakeSync struct {
	in   string
	src  entities.ExternalService
	resp entities.Item
	err  error
}

func (f *fakeSync) Execute(externalID string, source entities.ExternalService) (entities.Item, error) {
	f.in = externalID
	f.src = source
	return f.resp, f.err
}

type fakeLibrary struct {
	in   string
	resp []entities.Interaction
	err  error
}

func (f *fakeLibrary) Execute(uid string) ([]entities.Interaction, error) {
	f.in = uid
	return f.resp, f.err
}

type fakeLibraryItems struct {
	uid   string
	limit int
	resp  []usecase.LibraryItem
	err   error
}

func (f *fakeLibraryItems) Execute(uid string, limit int) ([]usecase.LibraryItem, error) {
	f.uid = uid
	f.limit = limit
	return f.resp, f.err
}

type fakeLinkExt struct {
	uid     string
	account entities.ExternalAccount
	resp    entities.ExternalAccount
	err     error
}

func (f *fakeLinkExt) Execute(uid string, account entities.ExternalAccount) (entities.ExternalAccount, error) {
	f.uid = uid
	f.account = account
	return f.resp, f.err
}

type fakeUserLookup struct {
	in   string
	resp entities.User
	err  error
}

func (f *fakeUserLookup) GetByID(uid string) (entities.User, error) {
	f.in = uid
	return f.resp, f.err
}

// ---------- helpers ---------------------------------------------------------

// withUser injects a user_id into the request context — emulating what
// the auth middleware would do without dragging in token signing.
func withUser(r *http.Request, uid string) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), ctxUserID, uid))
}

func decodeError(t *testing.T, body []byte) apiError {
	t.Helper()
	var e apiError
	if err := json.Unmarshal(body, &e); err != nil {
		t.Fatalf("decode error: %v (body=%q)", err, string(body))
	}
	return e
}

// ---------- recommendations -------------------------------------------------

func TestHandleGetRecommendations_Unauthorized(t *testing.T) {
	h := &Handlers{GetRecommendations: &fakeGetRecs{}}
	r := httptest.NewRequest(http.MethodPost, "/v1/recommendations", strings.NewReader(`{}`))
	w := httptest.NewRecorder()
	h.HandleGetRecommendations(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestHandleGetRecommendations_BadJSON(t *testing.T) {
	h := &Handlers{GetRecommendations: &fakeGetRecs{}}
	r := withUser(httptest.NewRequest(http.MethodPost, "/v1/recommendations", strings.NewReader(`{`)), "u-1")
	w := httptest.NewRecorder()
	h.HandleGetRecommendations(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
	if e := decodeError(t, w.Body.Bytes()); e.Code != "invalid_body" {
		t.Fatalf("expected invalid_body, got %q", e.Code)
	}
}

func TestHandleGetRecommendations_PassesUserID(t *testing.T) {
	uc := &fakeGetRecs{resp: usecase.RecommendationResponse{
		Items: []usecase.RecommendationItem{{ItemID: "i-1", Title: "X", Score: 0.5}},
	}}
	h := &Handlers{GetRecommendations: uc}
	r := withUser(
		httptest.NewRequest(http.MethodPost, "/v1/recommendations", strings.NewReader(`{"filters":{}}`)),
		"u-42",
	)
	w := httptest.NewRecorder()
	h.HandleGetRecommendations(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if uc.in != "u-42" {
		t.Fatalf("expected use case to receive uid u-42, got %q", uc.in)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("expected JSON content-type, got %q", ct)
	}
}

func TestHandleGetRecommendations_UseCaseError(t *testing.T) {
	h := &Handlers{GetRecommendations: &fakeGetRecs{err: errors.New("boom")}}
	r := withUser(httptest.NewRequest(http.MethodPost, "/v1/recommendations", strings.NewReader(`{}`)), "u-1")
	w := httptest.NewRecorder()
	h.HandleGetRecommendations(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ---------- search ----------------------------------------------------------

func TestHandleSearch_PassesQueryAndLimit(t *testing.T) {
	s := &fakeSearch{resp: []entities.Item{{ID: "i-1", Title: "Dune"}}}
	h := &Handlers{Search: s}
	r := httptest.NewRequest(http.MethodGet, "/v1/search?q=dune&limit=7", nil)
	w := httptest.NewRecorder()
	h.HandleSearch(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if s.in.Text != "dune" || s.in.Limit != 7 {
		t.Fatalf("unexpected query: %+v", s.in)
	}
}

func TestHandleSearch_IgnoresInvalidLimit(t *testing.T) {
	s := &fakeSearch{}
	h := &Handlers{Search: s}
	r := httptest.NewRequest(http.MethodGet, "/v1/search?q=x&limit=oops", nil)
	w := httptest.NewRecorder()
	h.HandleSearch(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if s.in.Limit != 0 {
		t.Fatalf("expected non-numeric limit to be dropped, got %d", s.in.Limit)
	}
}

func TestHandleSearch_UseCaseError(t *testing.T) {
	h := &Handlers{Search: &fakeSearch{err: errors.New("db down")}}
	r := httptest.NewRequest(http.MethodGet, "/v1/search?q=x", nil)
	w := httptest.NewRecorder()
	h.HandleSearch(w, r)
	if w.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d", w.Code)
	}
}

// ---------- content ---------------------------------------------------------

func TestHandleGetContent_MissingID(t *testing.T) {
	h := &Handlers{GetContent: &fakeGetContent{}}
	r := httptest.NewRequest(http.MethodGet, "/v1/content/", nil)
	// No path value set; PathValue("id") returns "" — handler should refuse.
	w := httptest.NewRecorder()
	h.HandleGetContent(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleGetContent_NotFound(t *testing.T) {
	h := &Handlers{GetContent: &fakeGetContent{err: errors.New("nope")}}
	r := httptest.NewRequest(http.MethodGet, "/v1/content/x", nil)
	r.SetPathValue("id", "x")
	w := httptest.NewRecorder()
	h.HandleGetContent(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleGetContent_OK(t *testing.T) {
	uc := &fakeGetContent{resp: entities.Item{ID: "i-1", Title: "Dune", MediaType: entities.MediaTypeBook}}
	h := &Handlers{GetContent: uc}
	r := httptest.NewRequest(http.MethodGet, "/v1/content/i-1", nil)
	r.SetPathValue("id", "i-1")
	w := httptest.NewRecorder()
	h.HandleGetContent(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if uc.in != "i-1" {
		t.Fatalf("expected i-1, got %q", uc.in)
	}
}

func TestHandleUpsertContent_Validation(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"empty", `{}`},
		{"no-media", `{"title":"x"}`},
		{"no-title", `{"media_type":"book"}`},
		{"bad-json", `{`},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := &Handlers{UpsertContent: &fakeUpsertContent{}}
			r := httptest.NewRequest(http.MethodPost, "/v1/content", strings.NewReader(c.body))
			w := httptest.NewRecorder()
			h.HandleUpsertContent(w, r)
			if w.Code != http.StatusBadRequest {
				t.Fatalf("expected 400, got %d", w.Code)
			}
		})
	}
}

func TestHandleUpsertContent_OK(t *testing.T) {
	uc := &fakeUpsertContent{}
	h := &Handlers{UpsertContent: uc}
	r := httptest.NewRequest(http.MethodPost, "/v1/content",
		strings.NewReader(`{"id":"i-1","title":"Dune","media_type":"book"}`))
	w := httptest.NewRecorder()
	h.HandleUpsertContent(w, r)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", w.Code)
	}
	if uc.in.Title != "Dune" {
		t.Fatalf("payload not propagated: %+v", uc.in)
	}
}

// ---------- interactions ----------------------------------------------------

func TestHandleUpdateInteraction_RatingValidation(t *testing.T) {
	cases := []struct {
		name string
		body string
		want int
	}{
		{"missing-item-id", `{"item_id":"","data":{}}`, http.StatusBadRequest},
		{"rating-too-high", `{"item_id":"i","data":{"rating":11}}`, http.StatusBadRequest},
		{"rating-too-low-but-nonzero", `{"item_id":"i","data":{"rating":-1}}`, http.StatusBadRequest},
		{"rating-zero-allowed", `{"item_id":"i","data":{"rating":0}}`, http.StatusNoContent},
		{"rating-mid", `{"item_id":"i","data":{"rating":7}}`, http.StatusNoContent},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := &Handlers{UpdateInteraction: &fakeUpdateInt{}}
			r := withUser(httptest.NewRequest(http.MethodPut, "/v1/interactions", strings.NewReader(c.body)), "u-1")
			w := httptest.NewRecorder()
			h.HandleUpdateInteraction(w, r)
			if w.Code != c.want {
				t.Fatalf("expected %d, got %d", c.want, w.Code)
			}
		})
	}
}

func TestHandleUpdateInteraction_RequiresAuth(t *testing.T) {
	h := &Handlers{UpdateInteraction: &fakeUpdateInt{}}
	r := httptest.NewRequest(http.MethodPut, "/v1/interactions",
		strings.NewReader(`{"item_id":"i","data":{"rating":5}}`))
	w := httptest.NewRecorder()
	h.HandleUpdateInteraction(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401 without auth context, got %d", w.Code)
	}
}

func TestHandleUpdateInteraction_PropagatesPayload(t *testing.T) {
	uc := &fakeUpdateInt{}
	h := &Handlers{UpdateInteraction: uc}
	body := `{"item_id":"i-1","data":{"status":"completed","rating":8,"is_favorite":true,"review_text":"great"}}`
	r := withUser(httptest.NewRequest(http.MethodPut, "/v1/interactions", strings.NewReader(body)), "u-9")
	w := httptest.NewRecorder()
	h.HandleUpdateInteraction(w, r)
	if w.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d (body=%s)", w.Code, w.Body.String())
	}
	if uc.uid != "u-9" || uc.itemID != "i-1" {
		t.Fatalf("unexpected uid/itemID: %q %q", uc.uid, uc.itemID)
	}
	if uc.data.Rating != 8 || !uc.data.IsFavorite || uc.data.Status != entities.InteractionStatusCompleted {
		t.Fatalf("payload mismatch: %+v", uc.data)
	}
}

// ---------- library ---------------------------------------------------------

func TestHandleGetLibrary_Auth(t *testing.T) {
	h := &Handlers{Library: &fakeLibrary{}}
	r := httptest.NewRequest(http.MethodGet, "/v1/library", nil)
	w := httptest.NewRecorder()
	h.HandleGetLibrary(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestHandleGetLibrary_OK(t *testing.T) {
	uc := &fakeLibrary{resp: []entities.Interaction{{ID: 1, ItemID: "i", Status: entities.InteractionStatusCompleted}}}
	h := &Handlers{Library: uc}
	r := withUser(httptest.NewRequest(http.MethodGet, "/v1/library", nil), "u-1")
	w := httptest.NewRecorder()
	h.HandleGetLibrary(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if uc.in != "u-1" {
		t.Fatalf("uid not forwarded: %q", uc.in)
	}
}

func TestHandleGetLibraryItems_Limit(t *testing.T) {
	uc := &fakeLibraryItems{resp: []usecase.LibraryItem{{Item: entities.Item{ID: "i"}}}}
	h := &Handlers{LibraryItems: uc}
	r := withUser(httptest.NewRequest(http.MethodGet, "/v1/library/items?limit=12", nil), "u-1")
	w := httptest.NewRecorder()
	h.HandleGetLibraryItems(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if uc.limit != 12 {
		t.Fatalf("limit not forwarded: %d", uc.limit)
	}
}

// Compile-time guard: usecase.LibraryItem still wraps a postgres LibraryItem.
var _ = repopostgres.LibraryItem{}

// ---------- sync external ---------------------------------------------------

func TestHandleSyncExternal_Validation(t *testing.T) {
	h := &Handlers{SyncExternal: &fakeSync{}}
	r := httptest.NewRequest(http.MethodPost, "/v1/sync/external", strings.NewReader(`{"external_id":"x"}`))
	w := httptest.NewRecorder()
	h.HandleSyncExternal(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleSyncExternal_OK(t *testing.T) {
	uc := &fakeSync{resp: entities.Item{ID: "i", Title: "y"}}
	h := &Handlers{SyncExternal: uc}
	body := `{"external_id":"steam-1","source":"steam"}`
	r := httptest.NewRequest(http.MethodPost, "/v1/sync/external", strings.NewReader(body))
	w := httptest.NewRecorder()
	h.HandleSyncExternal(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if uc.in != "steam-1" || uc.src != entities.ExternalServiceSteam {
		t.Fatalf("unexpected sync args: %+v %s", uc.in, uc.src)
	}
}

// ---------- link external account -------------------------------------------

func TestHandleLinkExternalAccount_NotConfigured(t *testing.T) {
	h := &Handlers{}
	r := withUser(
		httptest.NewRequest(http.MethodPost, "/v1/external-accounts", strings.NewReader(`{}`)),
		"u-1",
	)
	w := httptest.NewRecorder()
	h.HandleLinkExternalAccount(w, r)
	if w.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d", w.Code)
	}
}

func TestHandleLinkExternalAccount_Validation(t *testing.T) {
	h := &Handlers{LinkExternalAccount: &fakeLinkExt{}}
	r := withUser(
		httptest.NewRequest(http.MethodPost, "/v1/external-accounts", strings.NewReader(`{"service_name":"steam"}`)),
		"u-1",
	)
	w := httptest.NewRecorder()
	h.HandleLinkExternalAccount(w, r)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", w.Code)
	}
}

func TestHandleLinkExternalAccount_BindsCallerUserID(t *testing.T) {
	uc := &fakeLinkExt{resp: entities.ExternalAccount{AccountID: 17}}
	h := &Handlers{LinkExternalAccount: uc}
	body := `{"service_name":"steam","external_user_id":"steam-99"}`
	r := withUser(
		httptest.NewRequest(http.MethodPost, "/v1/external-accounts", strings.NewReader(body)),
		"u-555",
	)
	w := httptest.NewRecorder()
	h.HandleLinkExternalAccount(w, r)
	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d", w.Code)
	}
	if uc.uid != "u-555" {
		t.Fatalf("expected caller uid to be passed, got %q", uc.uid)
	}
	if uc.account.ServiceName != entities.ExternalServiceSteam {
		t.Fatalf("expected steam service, got %q", uc.account.ServiceName)
	}
}

// ---------- profile ---------------------------------------------------------

func TestHandleGetProfile_Auth(t *testing.T) {
	h := &Handlers{Users: &fakeUserLookup{}}
	r := httptest.NewRequest(http.MethodGet, "/v1/profile", nil)
	w := httptest.NewRecorder()
	h.HandleGetProfile(w, r)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestHandleGetProfile_NotFound(t *testing.T) {
	h := &Handlers{Users: &fakeUserLookup{err: errors.New("missing")}}
	r := withUser(httptest.NewRequest(http.MethodGet, "/v1/profile", nil), "u-1")
	w := httptest.NewRecorder()
	h.HandleGetProfile(w, r)
	if w.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", w.Code)
	}
}

func TestHandleGetProfile_OmitsPasswordHash(t *testing.T) {
	h := &Handlers{Users: &fakeUserLookup{
		resp: entities.User{ID: "u-1", Username: "alice", Email: "a@b.c", PasswordHash: "should-not-leak"},
	}}
	r := withUser(httptest.NewRequest(http.MethodGet, "/v1/profile", nil), "u-1")
	w := httptest.NewRecorder()
	h.HandleGetProfile(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	body := w.Body.String()
	if strings.Contains(body, "should-not-leak") {
		t.Fatalf("password hash leaked in response: %s", body)
	}
	var resp profileResponse
	if err := json.Unmarshal([]byte(body), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Username != "alice" || resp.Email != "a@b.c" {
		t.Fatalf("unexpected profile: %+v", resp)
	}
}
