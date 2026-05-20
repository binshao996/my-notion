package comment

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/bin-ke/my-notion/internal/auth"
	"github.com/bin-ke/my-notion/pkg/db"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	Service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{Service: service}
}

// ListByPage returns all comments for a page.
func (h *Handler) ListByPage(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	pageID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid page id"})
		return
	}

	comments, err := h.Service.ListByPage(uint(pageID))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to list comments"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comments)
}

// Create adds a new comment.
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
		BlockID  *uint  `json:"block_id"`
		Content  string `json:"content"`
		ParentID *uint  `json:"parent_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	if req.Content == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "content is required"})
		return
	}

	comment, err := h.Service.Create(uint(pageID), req.BlockID, claims.UserID, req.Content, req.ParentID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to create comment"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(comment)
}

// Update edits or resolves a comment.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	commentID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid comment id"})
		return
	}

	var req struct {
		Content  string `json:"content"`
		Resolved *bool  `json:"resolved"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	var comment *db.Comment

	if req.Resolved != nil {
		comment, err = h.Service.Resolve(uint(commentID))
	} else if req.Content != "" {
		comment, err = h.Service.Update(uint(commentID), req.Content)
	} else {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "content or resolved field is required"})
		return
	}

	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(comment)
}

// Delete removes a comment.
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	commentID, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid comment id"})
		return
	}

	if err := h.Service.Delete(uint(commentID)); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to delete comment"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "deleted"})
}
