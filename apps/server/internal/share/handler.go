package share

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/bin-ke/my-notion/internal/auth"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	Service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{Service: service}
}

// ListByPage returns all share tokens for a page.
func (h *Handler) ListByPage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	pageID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid page id"})
		return
	}

	tokens, err := h.Service.ListByPage(uint(pageID))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to list share tokens"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tokens)
}

// Create generates a new share token.
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	pageID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid page id"})
		return
	}

	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	var req struct {
		Role      string     `json:"role"`
		ExpiresAt *time.Time `json:"expires_at"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if req.Role == "" {
		req.Role = "viewer"
	}

	token, err := h.Service.Create(uint(pageID), req.Role, req.ExpiresAt, claims.UserID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to create share token"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(token)
}

// Revoke deletes a share token.
func (h *Handler) Revoke(w http.ResponseWriter, r *http.Request) {
	tokIDStr := chi.URLParam(r, "tokId")
	tokID, err := strconv.ParseUint(tokIDStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid token id"})
		return
	}

	if err := h.Service.Revoke(uint(tokID)); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to revoke token"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}

// ResolveToken is a public endpoint — resolves a share token and returns page data.
func (h *Handler) ResolveToken(w http.ResponseWriter, r *http.Request) {
	tokenStr := chi.URLParam(r, "token")
	if tokenStr == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing token"})
		return
	}

	token, page, blocks, err := h.Service.ResolveToken(tokenStr)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"page":   page,
		"blocks": blocks,
		"role":   token.Role,
	})
}
