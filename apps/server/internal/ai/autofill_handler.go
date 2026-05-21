package ai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/bin-ke/my-notion/internal/auth"
	"github.com/go-chi/chi/v5"
)

// AutofillHandler handles HTTP requests for AI database autofill operations.
type AutofillHandler struct {
	service *AutofillService
}

// NewAutofillHandler creates a new AutofillHandler.
func NewAutofillHandler(service *AutofillService) *AutofillHandler {
	return &AutofillHandler{service: service}
}

// Autofill handles POST /api/v1/ai/autofill — fill one or many records.
func (h *AutofillHandler) Autofill(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "not authenticated"})
		return
	}

	var req AutofillRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	// Batch fill: multiple records or all records in database
	if len(req.RecordIDs) != 1 {
		jobID, err := h.service.FillPropertyBatch(claims.UserID, &req)
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
			return
		}

		job, _ := h.service.GetJobStatus(jobID)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(job)
		return
	}

	// Single record fill: run synchronously
	if err := h.service.FillProperty(claims.UserID, &req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Return the updated record
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":    "completed",
		"record_id": req.RecordIDs[0],
	})
}

// JobStatus handles GET /api/v1/ai/autofill/{jobId} — check batch job status.
func (h *AutofillHandler) JobStatus(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "not authenticated"})
		return
	}

	jobID := chi.URLParam(r, "jobId")
	if jobID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing job ID"})
		return
	}

	job, err := h.service.GetJobStatus(jobID)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(job)
}

// JobStream handles GET /api/v1/ai/autofill/{jobId}/stream — SSE stream of job progress.
func (h *AutofillHandler) JobStream(w http.ResponseWriter, r *http.Request) {
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "not authenticated"})
		return
	}

	jobID := chi.URLParam(r, "jobId")
	if jobID == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "missing job ID"})
		return
	}

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming unsupported"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			job, err := h.service.GetJobStatus(jobID)
			if err != nil {
				data, _ := json.Marshal(map[string]string{"error": err.Error()})
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
				return
			}

			// Send progress update
			data, _ := json.Marshal(map[string]interface{}{
				"completed": job.Completed,
				"failed":    job.Failed,
				"total":     job.Total,
			})
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()

			// Check if job is done
			if job.Status == "completed" || job.Status == "failed" {
				doneData, _ := json.Marshal(map[string]interface{}{
					"done":   true,
					"status": job.Status,
				})
				fmt.Fprintf(w, "data: %s\n\n", doneData)
				flusher.Flush()
				return
			}
		}
	}
}
