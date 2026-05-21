package ai

import (
	"fmt"
	"math"
	"strings"
	"sync"
	"time"
)

// tokenBucket implements a simple token bucket rate limiter.
type tokenBucket struct {
	tokens     float64
	capacity   float64
	rate       float64 // tokens per second
	lastRefill time.Time
	mu         sync.Mutex
}

func newTokenBucket(capacity float64, ratePerMinute float64) *tokenBucket {
	return &tokenBucket{
		tokens:     capacity,
		capacity:   capacity,
		rate:       ratePerMinute / 60.0,
		lastRefill: time.Now(),
	}
}

func (tb *tokenBucket) allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens = math.Min(tb.capacity, tb.tokens+elapsed*tb.rate)
	tb.lastRefill = now

	if tb.tokens >= 1 {
		tb.tokens -= 1
		return true
	}
	return false
}

// chatCompleter is the interface for AI chat completion, allowing mock injection in tests.
type chatCompleter interface {
	ChatCompletion(req *ChatRequest) (*ChatResponse, error)
	ChatCompletionStream(req *ChatRequest, onDelta func(delta string) error) (*ChatResponse, error)
}

// Gateway provides rate limiting, cost tracking, and prompt injection guard.
type Gateway struct {
	client     chatCompleter
	userLimits map[uint]*tokenBucket
	mu         sync.Mutex
	monitor    *Monitor
}

// NewGateway creates a new Gateway.
func NewGateway(client chatCompleter) *Gateway {
	return &Gateway{
		client:     client,
		userLimits: make(map[uint]*tokenBucket),
		monitor:    NewMonitor(),
	}
}

// GetMonitor returns the gateway's monitor instance.
func (g *Gateway) GetMonitor() *Monitor {
	return g.monitor
}

// CheckLimit checks whether the user has exceeded their rate limit (20 req/min).
func (g *Gateway) CheckLimit(userID uint) error {
	g.mu.Lock()
	tb, ok := g.userLimits[userID]
	if !ok {
		tb = newTokenBucket(20, 20)
		g.userLimits[userID] = tb
	}
	g.mu.Unlock()

	if !tb.allow() {
		return fmt.Errorf("rate limit exceeded: 20 requests per minute allowed")
	}
	return nil
}

// LogCost records AI usage cost for tracking purposes.
func (g *Gateway) LogCost(userID uint, model string, promptTokens, completionTokens int, feature string) {
	cost := EstimateCost(model, promptTokens, completionTokens)
	fmt.Printf(
		"[AI-COST] user=%d model=%s feature=%s prompt_tokens=%d completion_tokens=%d cost=$%.6f\n",
		userID, model, feature, promptTokens, completionTokens, cost,
	)

	// Record to monitor for the dashboard.
	g.monitor.Record(CostLog{
		UserID:    userID,
		Model:     model,
		Tokens:    promptTokens + completionTokens,
		Cost:      cost,
		Feature:   feature,
	})
}

// SanitizePrompt checks the prompt for known injection attack patterns.
// Returns an error if a pattern is found; otherwise returns the prompt unchanged.
func (g *Gateway) SanitizePrompt(prompt string) (string, error) {
	lower := strings.ToLower(prompt)

	injectionPatterns := []string{
		"ignore previous instructions",
		"ignore all instructions",
		"ignore the above",
		"ignore your instructions",
		"forget your instructions",
		"system:",
		"<|im_start|>",
		"<|im_end|>",
		"begin prompt",
		"end prompt",
		"dan:",
		"jailbreak",
		"pretend you are",
		"you are now",
		"new instructions:",
	}

	for _, pattern := range injectionPatterns {
		if strings.Contains(lower, pattern) {
			return "", fmt.Errorf("prompt rejected: potential injection attack detected (pattern: %q)", pattern)
		}
	}

	return prompt, nil
}

// ModelForTask returns the best model for a given AI task.
// deepseek-v4-flash: fast, cheap — for classification, extraction, autofill
// deepseek-chat: balanced — for writing, rewriting, summarization, RAG
func ModelForTask(task string) string {
	switch task {
	case "autofill", "classify", "extract_actions", "extract_decisions",
		"extract_entities", "extract_structured", "proofread", "translate":
		return "deepseek-v4-flash" // fast tasks
	case "generate", "rewrite", "expand", "summarize", "rag":
		return "deepseek-chat" // creative/long-form
	default:
		return "deepseek-v4-flash"
	}
}
