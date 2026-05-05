package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"aura/backend/core-go/internal/domain/entities"
	"aura/backend/core-go/internal/infrastructure/clients/ai_engine"
	"aura/backend/core-go/internal/usecase"
)

// Handlers bundles the HTTP handlers of the core API.
type Handlers struct {
	GetRecommendations  usecase.GetRecommendationsUseCase
	Search              usecase.SearchContentUseCase
	GetContent          usecase.GetContentUseCase
	UpsertContent       usecase.UpsertContentUseCase
	UpdateInteraction   usecase.UpdateInteractionUseCase
	SyncExternal        usecase.SyncExternalContentUseCase
	Library             usecase.ListLibraryUseCase
	LibraryItems        usecase.ListLibraryItemsUseCase
	LinkExternalAccount usecase.LinkExternalAccountUseCase
	Auth                *AuthHandlers
	AIClient            ai_engine.Client
	Users               interface {
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

type linkExternalAccountRequest struct {
	ServiceName        entities.ExternalService `json:"service_name"`
	ExternalUserID     string                   `json:"external_user_id"`
	ExternalProfileURL string                   `json:"external_profile_url,omitempty"`
}

// HandleLinkExternalAccount serves POST /v1/external-accounts: associates a
// third-party service profile (Steam, Goodreads, ...) with the authenticated
// user. The use case ignores any user_id in the body — the link is bound to
// the caller's session.
func (h *Handlers) HandleLinkExternalAccount(w http.ResponseWriter, r *http.Request) {
	if h.LinkExternalAccount == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "External account linking is not configured")
		return
	}
	uid, ok := userIDFromContext(r.Context())
	if !ok {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Unauthorized")
		return
	}

	var req linkExternalAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Invalid JSON body")
		return
	}
	if req.ServiceName == "" || req.ExternalUserID == "" {
		writeError(w, http.StatusBadRequest, "missing_fields", "Missing service_name or external_user_id")
		return
	}

	account, err := h.LinkExternalAccount.Execute(uid, entities.ExternalAccount{
		ServiceName:        req.ServiceName,
		ExternalUserID:     req.ExternalUserID,
		ExternalProfileURL: req.ExternalProfileURL,
	})
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_request", "Invalid request")
		return
	}
	writeJSON(w, http.StatusCreated, account)
}

// HandleGetProfile returns the current user (requires auth middleware).
// Goes through a typed DTO instead of marshalling entities.User directly
// so we never depend on JSON tags inside the domain to keep secrets out
// of the wire.
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
	writeJSON(w, http.StatusOK, profileResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		CreatedAt: u.CreatedAt.UTC().Format(time.RFC3339),
	})
}

type profileResponse struct {
	ID        string `json:"id"`
	Username  string `json:"username"`
	Email     string `json:"email"`
	CreatedAt string `json:"created_at"`
}

type assistantChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type assistantChatRequest struct {
	Message string                 `json:"message"`
	History []assistantChatMessage `json:"history"`
}

type assistantChatResponse struct {
	Text              string   `json:"text"`
	RecommendationIDs []string `json:"recommendation_ids"`
}

// HandleAssistantChat serves POST /v1/assistant — proxies to the AI engine chat endpoint.
func (h *Handlers) HandleAssistantChat(w http.ResponseWriter, r *http.Request) {
	if h.AIClient == nil {
		writeError(w, http.StatusServiceUnavailable, "not_configured", "AI assistant is not configured")
		return
	}
	var req assistantChatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_body", "Invalid JSON body")
		return
	}
	if req.Message == "" {
		writeError(w, http.StatusBadRequest, "missing_message", "message is required")
		return
	}

	history := make([]ai_engine.ChatMessage, 0, len(req.History))
	for _, m := range req.History {
		history = append(history, ai_engine.ChatMessage{Role: m.Role, Content: m.Content})
	}

	result, err := h.AIClient.Chat(ai_engine.ChatRequest{Message: req.Message, History: history})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "ai_error", "AI assistant error")
		return
	}
	ids := result.RecommendationIDs
	if ids == nil {
		ids = []string{}
	}
	writeJSON(w, http.StatusOK, assistantChatResponse{
		Text:              result.Text,
		RecommendationIDs: ids,
	})
}
