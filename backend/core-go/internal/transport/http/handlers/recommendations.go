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
	GetContent         usecase.GetContentUseCase
	UpsertContent      usecase.UpsertContentUseCase
	UpdateInteraction  usecase.UpdateInteractionUseCase
	SyncExternal       usecase.SyncExternalContentUseCase
	Library            usecase.ListLibraryUseCase
	LibraryItems       usecase.ListLibraryItemsUseCase
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
		writeError(w, http.StatusBadRequest, "invalid_body", "Invalid JSON body")
		return
	}
	uid, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}
	resp, err := h.GetRecommendations.Execute(uid, req.Filters)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal error")
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
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal error")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// HandleGetContent serves GET /v1/content/{id}
func (h *Handlers) HandleGetContent(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if id == "" {
		writeError(w, http.StatusBadRequest, "missing_id", "Missing id")
		return
	}
	it, err := h.GetContent.Execute(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Not found")
		return
	}
	writeJSON(w, http.StatusOK, it)
}

// HandleUpsertContent serves POST /v1/content (upsert/insert base item).
func (h *Handlers) HandleUpsertContent(w http.ResponseWriter, r *http.Request) {
	var it entities.Item
	if err := json.NewDecoder(r.Body).Decode(&it); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Invalid JSON body")
		return
	}
	if it.Title == "" || it.MediaType == "" {
		writeError(w, http.StatusBadRequest, "missing_fields", "Missing title or media_type")
		return
	}
	if err := h.UpsertContent.Execute(it); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

type interactionRequest struct {
	ItemID string                  `json:"item_id"`
	Data   usecase.InteractionData `json:"data"`
}

// HandleUpdateInteraction serves PUT /v1/interactions.
func (h *Handlers) HandleUpdateInteraction(w http.ResponseWriter, r *http.Request) {
	var req interactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Invalid JSON body")
		return
	}
	if req.ItemID == "" {
		writeError(w, http.StatusBadRequest, "missing_item_id", "Missing item_id")
		return
	}
	// Rating is optional (0 means "unset"); when set, enforce 1..10.
	if req.Data.Rating != 0 && (req.Data.Rating < 1 || req.Data.Rating > 10) {
		writeError(w, http.StatusBadRequest, "invalid_rating", "Rating must be 1..10 or 0")
		return
	}
	uid, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}
	if err := h.UpdateInteraction.Execute(uid, req.ItemID, req.Data); err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal error")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// HandleGetLibrary serves GET /v1/library.
func (h *Handlers) HandleGetLibrary(w http.ResponseWriter, r *http.Request) {
	uid, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}
	items, err := h.Library.Execute(uid)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal error")
		return
	}
	writeJSON(w, http.StatusOK, items)
}

// HandleGetLibraryItems serves GET /v1/library/items.
func (h *Handlers) HandleGetLibraryItems(w http.ResponseWriter, r *http.Request) {
	uid, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}
	limit := 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			limit = n
		}
	}
	items, err := h.LibraryItems.Execute(uid, limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal error")
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
		writeError(w, http.StatusBadRequest, "invalid_body", "Invalid JSON body")
		return
	}
	if req.ExternalID == "" || req.Source == "" {
		writeError(w, http.StatusBadRequest, "missing_fields", "Missing external_id or source")
		return
	}
	item, err := h.SyncExternal.Execute(req.ExternalID, req.Source)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "Internal error")
		return
	}
	writeJSON(w, http.StatusOK, item)
}

// HandleGetProfile returns the current user (requires auth middleware).
func (h *Handlers) HandleGetProfile(w http.ResponseWriter, r *http.Request) {
	uid, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}
	u, err := h.Users.GetByID(uid)
	if err != nil {
		writeError(w, http.StatusNotFound, "not_found", "Not found")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":         u.ID,
		"username":   u.Username,
		"email":      u.Email,
		"created_at": u.CreatedAt,
	})
}
