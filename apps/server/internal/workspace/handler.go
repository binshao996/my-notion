package workspace

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"

	"github.com/bin-ke/my-notion/internal/auth"
	"github.com/bin-ke/my-notion/internal/notification"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	Service             *Service
	NotificationService *notification.Service
}

func NewHandler(service *Service, notifService *notification.Service) *Handler {
	return &Handler{Service: service, NotificationService: notifService}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "not authenticated"})
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Name == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "name is required"})
		return
	}

	ws, err := h.Service.Create(req.Name, claims.UserID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to create workspace"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(ws)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid workspace id"})
		return
	}

	ws, err := h.Service.GetByID(uint(id))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ws)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "not authenticated"})
		return
	}

	workspaces, err := h.Service.ListByUser(claims.UserID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to list workspaces"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(workspaces)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid workspace id"})
		return
	}

	if !h.Service.IsOwner(uint(id), claims.UserID) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "only workspace owners can update"})
		return
	}

	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	ws, err := h.Service.Update(uint(id), req.Name)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ws)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid workspace id"})
		return
	}

	if !h.Service.IsOwner(uint(id), claims.UserID) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "only workspace owners can delete"})
		return
	}

	if err := h.Service.Delete(uint(id)); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to delete workspace"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

type memberResponse struct {
	ID       uint   `json:"id"`
	UserID   uint   `json:"user_id"`
	Name     string `json:"name"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	JoinedAt string `json:"joined_at"`
}

func (h *Handler) ListMembers(w http.ResponseWriter, r *http.Request) {
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid workspace id"})
		return
	}

	members, err := h.Service.ListMembers(uint(id))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to list members"})
		return
	}

	out := make([]memberResponse, len(members))
	for i, m := range members {
		out[i] = memberResponse{
			ID:       m.ID,
			UserID:   m.UserID,
			Name:     m.User.Name,
			Email:    m.User.Email,
			Role:     m.Role,
			JoinedAt: m.JoinedAt.Format("2006-01-02T15:04:05Z"),
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(out)
}

func (h *Handler) AddMember(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseUint(idStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid workspace id"})
		return
	}

	if !h.Service.IsOwner(uint(id), claims.UserID) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "only workspace owners can add members"})
		return
	}

	var req struct {
		UserID uint   `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.UserID == 0 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "user_id is required"})
		return
	}
	if req.Role == "" {
		req.Role = "member"
	}

	if err := h.Service.AddMember(uint(id), req.UserID, req.Role); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to add member"})
		return
	}

	// Notify the added user
	if h.NotificationService != nil {
		if _, err := h.NotificationService.Create(req.UserID, "invite", claims.UserID, 0, nil); err != nil {
			log.Printf("WARNING: failed to create invite notification: %v", err)
		}
	}

	w.WriteHeader(http.StatusCreated)
}

func (h *Handler) RemoveMember(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	idStr := chi.URLParam(r, "id")
	memberIDStr := chi.URLParam(r, "memberId")
	id, err := strconv.ParseUint(idStr, 10, 64)
	memberID, err2 := strconv.ParseUint(memberIDStr, 10, 64)
	if err != nil || err2 != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid id"})
		return
	}

	if !h.Service.IsOwner(uint(id), claims.UserID) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]string{"error": "only workspace owners can remove members"})
		return
	}

	if err := h.Service.RemoveMember(uint(id), uint(memberID)); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to remove member"})
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
