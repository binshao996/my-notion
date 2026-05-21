package ai

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/bin-ke/my-notion/internal/auth"
)

// RAGHandler handles HTTP requests for AI Q&A (retrieval-augmented generation).
type RAGHandler struct {
	service *RAGService
}

// NewRAGHandler creates a new RAGHandler.
func NewRAGHandler(service *RAGService) *RAGHandler {
	return &RAGHandler{service: service}
}

// Ask handles POST /api/v1/ai/ask requests.
func (h *RAGHandler) Ask(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req QARequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	claims := auth.GetUserFromContext(r.Context())
	if claims == nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	// Streaming branch
	if r.URL.Query().Get("stream") == "true" {
		h.AskStream(w, r, claims.UserID, &req)
		return
	}

	resp, err := h.service.Ask(claims.UserID, &req)
	if err != nil {
		log.Printf("rag: ask error: %v", err)
		http.Error(w, "AI service error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// AskStream handles SSE streaming for RAG Q&A requests.
func (h *RAGHandler) AskStream(w http.ResponseWriter, r *http.Request, userID uint, req *QARequest) {
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

	// 1. Retrieve relevant chunks
	results, err := h.service.Retrieve(userID, req.WorkspaceID, req.Question)
	if err != nil {
		log.Printf("rag stream: retrieve error: %v", err)
		sendError("failed to retrieve context")
		return
	}

	// 2. Build system prompt with context
	systemPrompt := "You are a helpful assistant that answers questions based on the provided context. " +
		"Always cite your sources using [N] notation where N is the reference number. " +
		"If the context does not contain enough information, say so clearly."

	if len(results) > 0 {
		var contextParts []string
		for i, r := range results {
			contextParts = append(contextParts,
				fmt.Sprintf("Reference [%d]: Page '%s': %s", i+1, r.PageTitle, r.Text))
		}
		systemPrompt += "\n\nRelevant context:\n" + strings.Join(contextParts, "\n")
	}

	// 3. Stream LLM answer token by token
	chatReq := &ChatRequest{
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: req.Question},
		},
		MaxTokens:   1024,
		Temperature: 0.3,
	}

	fullAnswer, err := h.service.client.ChatCompletionStream(chatReq, func(delta string) error {
		data, _ := json.Marshal(map[string]string{"delta": delta})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return nil
	})

	if err != nil {
		log.Printf("rag stream: chat error: %v", err)
		data, _ := json.Marshal(map[string]string{"error": err.Error()})
		fmt.Fprintf(w, "data: %s\n\n", data)
		flusher.Flush()
		return
	}

	answer := ""
	if fullAnswer != nil && len(fullAnswer.Choices) > 0 {
		answer = fullAnswer.Choices[0].Message.Content
	}

	// 4. Parse citations from the full answer
	citations := h.service.parseCitations(answer, results)

	// 5. Send final event with done flag and citations
	finalData, _ := json.Marshal(map[string]any{
		"done":      true,
		"citations": citations,
	})
	fmt.Fprintf(w, "data: %s\n\n", finalData)
	flusher.Flush()
}
