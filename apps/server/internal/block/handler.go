package block

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/bin-ke/my-notion/pkg/db"
	"github.com/go-chi/chi/v5"
)

type Handler struct {
	Service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{Service: service}
}

func (h *Handler) GetByPage(w http.ResponseWriter, r *http.Request) {
	pageIDStr := chi.URLParam(r, "pageId")
	pageID, err := strconv.ParseUint(pageIDStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid page id"})
		return
	}

	blocks, err := h.Service.GetByPage(uint(pageID))
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to get blocks"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(blocks)
}

func (h *Handler) BatchSave(w http.ResponseWriter, r *http.Request) {
	pageIDStr := chi.URLParam(r, "pageId")
	pageID, err := strconv.ParseUint(pageIDStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid page id"})
		return
	}

	var blocks []db.Block
	if err := json.NewDecoder(r.Body).Decode(&blocks); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	saved, err := h.Service.BatchSave(uint(pageID), blocks)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "failed to save blocks"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(saved)
}

func (h *Handler) ApplyOps(w http.ResponseWriter, r *http.Request) {
	pageIDStr := chi.URLParam(r, "pageId")
	pageID, err := strconv.ParseUint(pageIDStr, 10, 64)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid page id"})
		return
	}

	var ops []OpRequest
	if err := json.NewDecoder(r.Body).Decode(&ops); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	h.Service.ApplyOps(uint(pageID), ops)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}
