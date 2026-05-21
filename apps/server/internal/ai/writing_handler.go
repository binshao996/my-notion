package ai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/bin-ke/my-notion/internal/auth"
)

// WritingHandler handles HTTP requests for AI writing operations.
type WritingHandler struct {
	service *WritingService
}

// NewWritingHandler creates a new WritingHandler.
func NewWritingHandler(service *WritingService) *WritingHandler {
	return &WritingHandler{service: service}
}

// Write handles POST /api/v1/ai/write requests.
func (h *WritingHandler) Write(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	// Parse request body
	var req WritingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "invalid request body"})
		return
	}

	// Validate action
	validActions := map[string]bool{
		"generate":           true,
		"rewrite":            true,
		"summarize":          true,
		"expand":             true,
		"translate":          true,
		"proofread":          true,
		"extract_actions":    true,
		"extract_decisions":  true,
		"extract_entities":   true,
		"classify":           true,
		"extract_structured": true,
	}
	if !validActions[req.Action] {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{
			"error": "invalid action: must be one of generate, rewrite, summarize, expand, translate, proofread, extract_actions, extract_decisions, extract_entities, classify, extract_structured",
		})
		return
	}

	// Require context for actions that need it
	if req.Action != "generate" && req.Context == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "context is required for this action"})
		return
	}

	// Get authenticated user from context
	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "authentication required"})
		return
	}

	// Streaming branch
	if r.URL.Query().Get("stream") == "true" {
		h.WriteStream(w, r, claims.UserID, &req)
		return
	}

	// Execute the writing operation
	resp, err := h.service.Execute(claims.UserID, &req)
	if err != nil {
		// Determine status code based on error type
		statusCode := http.StatusInternalServerError
		if strings.Contains(err.Error(), "rate limit") {
			statusCode = http.StatusTooManyRequests
		} else if strings.Contains(err.Error(), "rejected") {
			statusCode = http.StatusBadRequest
		}
		w.WriteHeader(statusCode)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Return success response
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(resp)
}

// WriteStream handles SSE streaming for AI writing requests.
func (h *WritingHandler) WriteStream(w http.ResponseWriter, r *http.Request, userID uint, req *WritingRequest) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, `{"error":"streaming unsupported"}`, http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	sendError := func(msg string) {
		data, _ := json.Marshal(map[string]string{"error": msg})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
	}

	// Sanitize inputs
	if _, err := h.service.gateway.SanitizePrompt(req.Prompt); err != nil {
		sendError("prompt rejected")
		return
	}
	if _, err := h.service.gateway.SanitizePrompt(req.Context); err != nil {
		sendError("context rejected")
		return
	}

	// Rate limit
	if err := h.service.gateway.CheckLimit(userID); err != nil {
		sendError("rate limit exceeded: 20 requests per minute allowed")
		return
	}

	// Build user prompt
	userPrompt := buildUserPrompt(req)
	if userPrompt == "" {
		sendError(fmt.Sprintf("action %q requires prompt or context", req.Action))
		return
	}

	// Set temperature
	temperature := 0.7
	if req.Action == "summarize" || req.Action == "translate" || req.Action == "proofread" ||
		req.Action == "extract_actions" || req.Action == "extract_decisions" || req.Action == "extract_entities" ||
		req.Action == "classify" || req.Action == "extract_structured" {
		temperature = 0.3
	}

	chatReq := &ChatRequest{
		Model: "deepseek-chat",
		Messages: []ChatMessage{
			{Role: "system", Content: writingSystemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   2048,
		Temperature: temperature,
	}

	// Stream from DeepSeek, sending each chunk as an SSE event
	response, err := h.service.gateway.client.ChatCompletionStream(chatReq, func(delta string) error {
		data, _ := json.Marshal(map[string]string{"delta": delta})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return nil
	})

	if err != nil {
		data, _ := json.Marshal(map[string]string{"error": err.Error()})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return
	}

	// Send final [DONE] event
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()

	// Log cost
	if response != nil && response.Usage.TotalTokens > 0 {
		h.service.gateway.LogCost(userID, chatReq.Model, response.Usage.PromptTokens, response.Usage.CompletionTokens, "writing")
	}
}
