package permission

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/bin-ke/my-notion/internal/auth"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	Service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{Service: service}
}

// ListByPage returns all permissions for a page.
func (h *Handler) ListByPage(w http.ResponseWriter, r *http.Request) {
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

	// Only editors can view permissions
	if !h.Service.CanEdit(claims.UserID, uint(pageID)) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	perms, err := h.Service.ListByPage(uint(pageID))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to list permissions"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(perms)
}

// Set adds or updates a permission on a page.
func (h *Handler) Set(w http.ResponseWriter, r *http.Request) {
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

	if !h.Service.CanEdit(claims.UserID, uint(pageID)) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	var req struct {
		SubjectType string `json:"subject_type"`
		SubjectID   uint   `json:"subject_id"`
		Role        string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if req.Role != "editor" && req.Role != "commenter" && req.Role != "viewer" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "role must be editor, commenter, or viewer"})
		return
	}

	if req.SubjectType == "" {
		req.SubjectType = "user"
	}

	perm, err := h.Service.Set(uint(pageID), req.SubjectType, req.SubjectID, req.Role)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to set permission"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(perm)
}

// Remove deletes a permission.
func (h *Handler) Remove(w http.ResponseWriter, r *http.Request) {
	pageIDStr := chi.URLParam(r, "id")
	pageID, err := strconv.ParseUint(pageIDStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid page id"})
		return
	}

	permIDStr := chi.URLParam(r, "permId")
	permID, err := strconv.ParseUint(permIDStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid permission id"})
		return
	}

	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	if !h.Service.CanEdit(claims.UserID, uint(pageID)) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "forbidden"})
		return
	}

	if err := h.Service.Remove(uint(permID)); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to remove permission"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusNoContent)
}
