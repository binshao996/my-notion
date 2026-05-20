package search

import (
	"encoding/json"
	"net/http"
	"strconv"

	"gorm.io/gorm"
)

type Handler struct {
	Service *Service
	DB      *gorm.DB
}

func NewHandler(service *Service, db *gorm.DB) *Handler {
	return &Handler{Service: service, DB: db}
}

func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	q := r.URL.Query().Get("q")
	wsID, err := strconv.ParseUint(r.URL.Query().Get("workspace_id"), 10, 64)
	if err != nil || wsID == 0 {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "workspace_id is required"})
		return
	}

	results, err := h.Service.Search(uint(wsID), q)
	if err != nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]string{"error": "search unavailable"})
		return
	}

	json.NewEncoder(w).Encode(results)
}

func (h *Handler) Reindex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if err := h.Service.ReindexAll(h.DB); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
