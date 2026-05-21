package ai

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"sync"
	"time"
)

// EvalCase represents a single evaluation test case.
type EvalCase struct {
	ID         string `json:"id"`
	Category   string `json:"category"` // qa, writing, extraction
	Input      string `json:"input"`     // question or prompt
	Expected   string `json:"expected"`  // expected answer or key facts
	Context    string `json:"context"`   // provided context (for RAG)
	SourcePage uint   `json:"source_page,omitempty"`
}

// EvalResult is the outcome of running a single eval case.
type EvalResult struct {
	CaseID       string    `json:"case_id"`
	Category     string    `json:"category"`
	Model        string    `json:"model"`
	Actual       string    `json:"actual"`
	Expected     string    `json:"expected"`
	Hallucinated bool      `json:"hallucinated"`  // manual/LLM judgment
	CitationOK   bool      `json:"citation_ok"`    // citations match sources
	Accuracy     float64   `json:"accuracy"`       // simple token overlap for MVP
	LatencyMs    int64     `json:"latency_ms"`
	TokensUsed   int       `json:"tokens_used"`
	Cost         float64   `json:"cost"`
	CreatedAt    time.Time `json:"created_at"`
}

// EvalHarness runs AI evaluations and computes metrics.
type EvalHarness struct {
	mu      sync.Mutex
	results []EvalResult
	cases   []EvalCase
}

// NewEvalHarness creates a new eval harness with built-in default cases.
func NewEvalHarness() *EvalHarness {
	return &EvalHarness{
		cases:   defaultEvalCases(),
		results: make([]EvalResult, 0),
	}
}

// LoadCases loads eval cases from a JSON file.
func (h *EvalHarness) LoadCases(path string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read eval cases: %w", err)
	}
	return json.Unmarshal(data, &h.cases)
}

// Run runs all eval cases against the given client, filtering by optional category.
func (h *EvalHarness) Run(client *Client, category string) []EvalResult {
	if client == nil || !client.IsAvailable() {
		return nil
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	// Reset results for a fresh run.
	h.results = make([]EvalResult, 0)

	for _, ec := range h.cases {
		if category != "" && ec.Category != category {
			continue
		}

		result := h.runCase(client, ec)
		h.results = append(h.results, result)
	}

	return h.results
}

// runCase executes a single eval case and returns its result.
func (h *EvalHarness) runCase(client *Client, ec EvalCase) EvalResult {
	result := EvalResult{
		CaseID:    ec.ID,
		Category:  ec.Category,
		Model:     client.config.Model,
		Expected:  ec.Expected,
		CreatedAt: time.Now(),
	}

	// Build chat request. For QA/extraction cases, include context in the system message.
	systemMsg := "You are a helpful AI assistant. Answer concisely and accurately."
	if ec.Context != "" {
		systemMsg = fmt.Sprintf("You are a helpful AI assistant. Use the following context to answer:\n\n%s", ec.Context)
	}

	req := &ChatRequest{
		Model: client.config.Model,
		Messages: []ChatMessage{
			{Role: "system", Content: systemMsg},
			{Role: "user", Content: ec.Input},
		},
		MaxTokens:   256,
		Temperature: 0,
	}

	start := time.Now()
	resp, err := client.ChatCompletion(req)
	elapsed := time.Since(start)
	result.LatencyMs = elapsed.Milliseconds()

	if err != nil {
		// Record error result.
		result.Actual = fmt.Sprintf("ERROR: %v", err)
		result.Accuracy = 0
		return result
	}

	if len(resp.Choices) > 0 {
		result.Actual = resp.Choices[0].Message.Content
	}

	result.TokensUsed = resp.Usage.TotalTokens
	result.Cost = EstimateCost(client.config.Model, resp.Usage.PromptTokens, resp.Usage.CompletionTokens)

	// Compute Jaccard similarity for accuracy.
	result.Accuracy = jaccardSimilarity(result.Expected, result.Actual)

	// Flag potential hallucination: low accuracy but expected is non-empty.
	if result.Accuracy < 0.1 && strings.TrimSpace(ec.Expected) != "" {
		result.Hallucinated = true
	}

	return result
}

// Metrics computes aggregate metrics from all results.
func (h *EvalHarness) Metrics() map[string]interface{} {
	h.mu.Lock()
	defer h.mu.Unlock()

	if len(h.results) == 0 {
		return map[string]interface{}{
			"avg_accuracy":     0.0,
			"avg_latency_ms":   0.0,
			"total_cost":       0.0,
			"hallucination_rate": 0.0,
			"citation_accuracy":   0.0,
			"total_tokens":       0,
			"cases_run":          0,
		}
	}

	var totalAccuracy float64
	var totalLatency int64
	var totalCost float64
	var totalTokens int
	var hallucinationCount int
	var citationOKCount int
	var citationCases int

	for _, r := range h.results {
		totalAccuracy += r.Accuracy
		totalLatency += r.LatencyMs
		totalCost += r.Cost
		totalTokens += r.TokensUsed
		if r.Hallucinated {
			hallucinationCount++
		}
		// CitationOK is meaningful for qa/extraction cases.
		if r.Category == "qa" || r.Category == "extraction" {
			citationCases++
			if r.CitationOK {
				citationOKCount++
			}
		}
	}

	n := float64(len(h.results))
	citationAccuracy := 0.0
	if citationCases > 0 {
		citationAccuracy = float64(citationOKCount) / float64(citationCases)
	}

	return map[string]interface{}{
		"avg_accuracy":      totalAccuracy / n,
		"avg_latency_ms":    float64(totalLatency) / n,
		"total_cost":        totalCost,
		"hallucination_rate": float64(hallucinationCount) / n,
		"citation_accuracy":   citationAccuracy,
		"total_tokens":       totalTokens,
		"cases_run":          len(h.results),
	}
}

// ExportResults writes results to a JSON file.
func (h *EvalHarness) ExportResults(path string) error {
	h.mu.Lock()
	defer h.mu.Unlock()

	data, err := json.MarshalIndent(h.results, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal results: %w", err)
	}
	return os.WriteFile(path, data, 0o644)
}

// jaccardSimilarity computes the Jaccard similarity coefficient between two strings
// using whitespace-delimited token sets. Returns a value between 0.0 and 1.0.
func jaccardSimilarity(a, b string) float64 {
	a = strings.ToLower(strings.TrimSpace(a))
	b = strings.ToLower(strings.TrimSpace(b))
	if a == "" && b == "" {
		return 1.0
	}
	if a == "" || b == "" {
		return 0.0
	}

	tokensA := tokenSet(a)
	tokensB := tokenSet(b)

	intersection := 0
	for token := range tokensA {
		if tokensB[token] {
			intersection++
		}
	}

	union := len(tokensA)
	for token := range tokensB {
		if !tokensA[token] {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}
	return float64(intersection) / float64(union)
}

// tokenSet returns a set of normalized tokens from the given string.
func tokenSet(s string) map[string]bool {
	words := strings.Fields(s)
	set := make(map[string]bool, len(words))
	for _, w := range words {
		// Trim common punctuation from word boundaries.
		w = strings.Trim(w, ".,;:!?\"'()[]{}-")
		if w != "" {
			set[w] = true
		}
	}
	return set
}

// defaultEvalCases returns a set of built-in eval cases for testing.
func defaultEvalCases() []EvalCase {
	return []EvalCase{
		// QA cases
		{ID: "qa-001", Category: "qa", Input: "What is the capital of France?", Expected: "Paris", Context: "France is a country in Europe. Its capital is Paris."},
		{ID: "qa-002", Category: "qa", Input: "When was the project deadline?", Expected: "Friday, June 15", Context: "The project deadline is Friday, June 15. We need to complete all deliverables by then."},
		// Writing cases
		{ID: "write-001", Category: "writing", Input: "Summarize: The team completed the authentication module. Users can now sign up with email and password. The login flow supports JWT tokens with refresh. Next step is OAuth integration.", Expected: "Auth module complete with email/password and JWT. OAuth next."},
		// Extraction cases
		{ID: "extract-001", Category: "extraction", Input: "Extract action items: Alice will deploy the service by Friday. Bob needs to update the documentation. Carol should schedule a team review.", Expected: "Alice deploy service (Friday), Bob update docs, Carol schedule review"},
	}
}
