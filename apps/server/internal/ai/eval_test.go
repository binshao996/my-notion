package ai

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestNewEvalHarness(t *testing.T) {
	h := NewEvalHarness()
	if h == nil {
		t.Fatal("NewEvalHarness() returned nil")
	}
	if h.cases == nil {
		t.Error("cases should not be nil")
	}
	if h.results == nil {
		t.Error("results should not be nil")
	}
}

func TestDefaultEvalCasesNotEmpty(t *testing.T) {
	h := NewEvalHarness()

	if len(h.cases) == 0 {
		t.Fatal("expected at least one default eval case")
	}
	if len(h.cases) < 4 {
		t.Errorf("expected at least 4 default cases, got %d", len(h.cases))
	}

	// Verify all cases have required fields and expected categories are present.
	categories := map[string]bool{}
	for _, c := range h.cases {
		if c.ID == "" {
			t.Error("eval case should have an ID")
		}
		categories[c.Category] = true
	}
	for _, cat := range []string{"qa", "writing", "extraction"} {
		if !categories[cat] {
			t.Errorf("expected category %q in default cases", cat)
		}
	}
}

func TestEvalHarnessLoadCasesFromFile(t *testing.T) {
	h := NewEvalHarness()

	testCases := []EvalCase{
		{ID: "test-1", Category: "qa", Input: "What is Go?", Expected: "A programming language"},
		{ID: "test-2", Category: "writing", Input: "Summarize Go", Expected: "Go is a language"},
	}

	tmpFile := filepath.Join(t.TempDir(), "cases.json")
	data, err := json.Marshal(testCases)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmpFile, data, 0o644); err != nil {
		t.Fatal(err)
	}

	if err := h.LoadCases(tmpFile); err != nil {
		t.Fatalf("LoadCases failed: %v", err)
	}

	if len(h.cases) != 2 {
		t.Fatalf("expected 2 loaded cases, got %d", len(h.cases))
	}
	if h.cases[0].ID != "test-1" {
		t.Errorf("expected first case ID 'test-1', got %q", h.cases[0].ID)
	}
	if h.cases[1].ID != "test-2" {
		t.Errorf("expected second case ID 'test-2', got %q", h.cases[1].ID)
	}
}

func TestEvalHarnessLoadCasesNonexistentFile(t *testing.T) {
	h := NewEvalHarness()
	err := h.LoadCases("/nonexistent/path/cases.json")
	if err == nil {
		t.Error("expected error loading nonexistent file, got nil")
	}
}

func TestEvalHarnessMetricsEmptyResults(t *testing.T) {
	h := NewEvalHarness()
	metrics := h.Metrics()

	if metrics["cases_run"].(int) != 0 {
		t.Errorf("expected cases_run 0, got %v", metrics["cases_run"])
	}
	if metrics["avg_accuracy"].(float64) != 0.0 {
		t.Errorf("expected avg_accuracy 0.0, got %v", metrics["avg_accuracy"])
	}
	if metrics["total_cost"].(float64) != 0.0 {
		t.Errorf("expected total_cost 0.0, got %v", metrics["total_cost"])
	}
	if metrics["total_tokens"].(int) != 0 {
		t.Errorf("expected total_tokens 0, got %v", metrics["total_tokens"])
	}
	if metrics["hallucination_rate"].(float64) != 0.0 {
		t.Errorf("expected hallucination_rate 0.0, got %v", metrics["hallucination_rate"])
	}
	if metrics["avg_latency_ms"].(float64) != 0.0 {
		t.Errorf("expected avg_latency_ms 0.0, got %v", metrics["avg_latency_ms"])
	}
}

func TestJaccardSimilarity(t *testing.T) {
	tests := []struct {
		name     string
		a        string
		b        string
		expected float64
	}{
		{"identical strings", "hello world", "hello world", 1.0},
		{"disjoint strings", "hello world", "goodbye moon", 0.0},
		{"partial overlap", "a b c", "a b d", 0.5}, // intersect={"a","b"}=2, union={"a","b","c","d"}=4 → 0.5
		{"empty both", "", "", 1.0},
		{"empty a", "", "hello", 0.0},
		{"empty b", "hello", "", 0.0},
		{"case insensitive", "Hello World", "hello world", 1.0},
		{"single word match", "hello", "hello", 1.0},
		{"single word no match", "hello", "world", 0.0},
		{"with punctuation", "hello, world!", "hello world", 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := jaccardSimilarity(tt.a, tt.b)
			tolerance := 0.01
			diff := result - tt.expected
			if diff < 0 {
				diff = -diff
			}
			if diff > tolerance {
				t.Errorf("jaccardSimilarity(%q, %q) = %f, want %f", tt.a, tt.b, result, tt.expected)
			}
		})
	}
}
