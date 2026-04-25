package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/usecase"
)

// Handlers bundles the HTTP handlers of the core API.
type Handlers struct {
	GetRecommendations usecase.GetRecommendationsUseCase
	Search             usecase.SearchContentUseCase
	UpdateInteraction  usecase.UpdateInteractionUseCase
	SyncExternal       usecase.SyncExternalContentUseCase
	Library            usecase.ListLibraryUseCase
	Auth               *AuthHandlers
	Users              interface {
		GetByID(userID string) (entities.User, error)
	}
}

// New constructs an HTTP adapter around the use-case interfaces.
func New(
	getRecs usecase.GetRecommendationsUseCase,
	search usecase.SearchContentUseCase,
	updateInt usecase.UpdateInteractionUseCase,
	syncExt usecase.SyncExternalContentUseCase,
) *Handlers {
	return &Handlers{
		GetRecommendations: getRecs,
		Search:             search,
		UpdateInteraction:  updateInt,
		SyncExternal:       syncExt,
	}
}

// Health is a liveness probe.
func Health(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("ok"))
}

type recommendationsRequest struct {
	Filters entities.RecommendationFilters `json:"filters"`
}

// HandleGetRecommendations serves POST /v1/recommendations.
func (h *Handlers) HandleGetRecommendations(w http.ResponseWriter, r *http.Request) {
	var req recommendationsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	uid, ok := userIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	resp, err := h.GetRecommendations.Execute(uid, req.Filters)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

// HandleSearch serves GET /v1/search?q=...
func (h *Handlers) HandleSearch(w http.ResponseWriter, r *http.Request) {
	limit := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	query := usecase.SearchQuery{Text: r.URL.Query().Get("q"), Limit: limit}
	items, err := h.Search.Execute(query)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

type interactionRequest struct {
	ItemID string                  `json:"item_id"`
	Data   usecase.InteractionData `json:"data"`
}

// HandleUpdateInteraction serves PUT /v1/interactions.
func (h *Handlers) HandleUpdateInteraction(w http.ResponseWriter, r *http.Request) {
	var req interactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	uid, ok := userIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := h.UpdateInteraction.Execute(uid, req.ItemID, req.Data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// HandleGetLibrary serves GET /v1/library.
func (h *Handlers) HandleGetLibrary(w http.ResponseWriter, r *http.Request) {
	uid, ok := userIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	items, err := h.Library.Execute(uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, items)
}

type syncRequest struct {
	ExternalID string                   `json:"external_id"`
	Source     entities.ExternalService `json:"source"`
}

// HandleSyncExternal serves POST /v1/sync/external.
func (h *Handlers) HandleSyncExternal(w http.ResponseWriter, r *http.Request) {
	var req syncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid body", http.StatusBadRequest)
		return
	}
	item, err := h.SyncExternal.Execute(req.ExternalID, req.Source)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// HandleGetProfile returns the current user (requires auth middleware).
func (h *Handlers) HandleGetProfile(w http.ResponseWriter, r *http.Request) {
	uid, ok := userIDFromContext(r.Context())
	if !ok {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	u, err := h.Users.GetByID(uid)
	if err != nil {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":         u.ID,
		"username":   u.Username,
		"email":      u.Email,
		"created_at": u.CreatedAt,
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
