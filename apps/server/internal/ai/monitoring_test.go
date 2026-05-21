package ai

import (
	"testing"
	"time"
)

func TestNewMonitor(t *testing.T) {
	m := NewMonitor()
	if m == nil {
		t.Fatal("NewMonitor() returned nil")
	}
	if m.logs == nil {
		t.Error("logs should be initialized")
	}
	if m.metrics == nil {
		t.Error("metrics should be initialized")
	}
	if len(m.logs) != 0 {
		t.Errorf("expected empty logs, got %d", len(m.logs))
	}
	if len(m.metrics) != 0 {
		t.Errorf("expected empty metrics, got %d", len(m.metrics))
	}
}

func TestMonitorRecord(t *testing.T) {
	m := NewMonitor()

	log := CostLog{
		UserID:  1,
		Model:   "deepseek-chat",
		Feature: "writing",
		Tokens:  100,
		Cost:    0.01,
	}
	m.Record(log)

	if len(m.logs) != 1 {
		t.Fatalf("expected 1 log, got %d", len(m.logs))
	}
	if m.logs[0].Feature != "writing" {
		t.Errorf("expected feature 'writing', got %q", m.logs[0].Feature)
	}
	if m.logs[0].Tokens != 100 {
		t.Errorf("expected tokens 100, got %d", m.logs[0].Tokens)
	}
	if m.logs[0].Cost != 0.01 {
		t.Errorf("expected cost 0.01, got %f", m.logs[0].Cost)
	}
	if m.logs[0].CreatedAt.IsZero() {
		t.Error("expected non-zero CreatedAt timestamp")
	}

	// Verify metrics were updated.
	fm, ok := m.metrics["writing"]
	if !ok {
		t.Fatal("expected metrics for 'writing' feature")
	}
	if fm.TotalCalls != 1 {
		t.Errorf("expected TotalCalls 1, got %d", fm.TotalCalls)
	}
	if fm.TotalTokens != 100 {
		t.Errorf("expected TotalTokens 100, got %d", fm.TotalTokens)
	}
	if fm.TotalCost != 0.01 {
		t.Errorf("expected TotalCost 0.01, got %f", fm.TotalCost)
	}
}

func TestMonitorStatsAggregation(t *testing.T) {
	m := NewMonitor()

	// Record 2 "writing" logs and 1 "rag" log.
	m.Record(CostLog{Feature: "writing", Tokens: 50, Cost: 0.005})
	m.Record(CostLog{Feature: "writing", Tokens: 30, Cost: 0.003})
	m.Record(CostLog{Feature: "rag", Tokens: 200, Cost: 0.02})

	stats := m.Stats(24)

	// Totals across all features.
	if stats.TotalCalls != 3 {
		t.Errorf("expected TotalCalls 3, got %d", stats.TotalCalls)
	}
	if stats.TotalTokens != 280 {
		t.Errorf("expected TotalTokens 280, got %d", stats.TotalTokens)
	}
	if stats.TotalCost < 0.027 || stats.TotalCost > 0.029 {
		t.Errorf("expected TotalCost ~0.028, got %f", stats.TotalCost)
	}

	// Should have 2 feature entries: writing and rag.
	if len(stats.ByFeature) != 2 {
		t.Fatalf("expected 2 features in ByFeature, got %d", len(stats.ByFeature))
	}

	writingMetrics := stats.ByFeature["writing"]
	if writingMetrics == nil {
		t.Fatal("expected 'writing' in ByFeature")
	}
	if writingMetrics.TotalCalls != 2 {
		t.Errorf("expected writing TotalCalls 2, got %d", writingMetrics.TotalCalls)
	}
	if writingMetrics.TotalTokens != 80 {
		t.Errorf("expected writing TotalTokens 80, got %d", writingMetrics.TotalTokens)
	}
	if writingMetrics.TotalCost < 0.007 || writingMetrics.TotalCost > 0.009 {
		t.Errorf("expected writing TotalCost ~0.008, got %f", writingMetrics.TotalCost)
	}

	ragMetrics := stats.ByFeature["rag"]
	if ragMetrics == nil {
		t.Fatal("expected 'rag' in ByFeature")
	}
	if ragMetrics.TotalCalls != 1 {
		t.Errorf("expected rag TotalCalls 1, got %d", ragMetrics.TotalCalls)
	}
	if ragMetrics.TotalTokens != 200 {
		t.Errorf("expected rag TotalTokens 200, got %d", ragMetrics.TotalTokens)
	}
}

func TestMonitorStatsMultipleFeatures(t *testing.T) {
	m := NewMonitor()

	m.Record(CostLog{Feature: "writing", Tokens: 10, Cost: 0.001})
	m.Record(CostLog{Feature: "rag", Tokens: 20, Cost: 0.002})
	m.Record(CostLog{Feature: "autofill", Tokens: 30, Cost: 0.003})

	stats := m.Stats(24)

	if len(stats.ByFeature) != 3 {
		t.Fatalf("expected 3 feature entries, got %d", len(stats.ByFeature))
	}

	for _, f := range []string{"writing", "rag", "autofill"} {
		if _, ok := stats.ByFeature[f]; !ok {
			t.Errorf("expected feature %q in ByFeature", f)
		}
	}

	if stats.WindowHours != 24 {
		t.Errorf("expected WindowHours 24, got %d", stats.WindowHours)
	}
}

func TestMonitorStatsWindowFiltering(t *testing.T) {
	m := NewMonitor()

	// Record a recent log through the public API.
	m.Record(CostLog{Feature: "writing", Tokens: 100, Cost: 0.01})

	// Manually insert an old log (25 hours ago) plus its metrics
	// so we can test window filtering on RecentLogs.
	oldLog := CostLog{
		Feature:   "rag",
		Tokens:    50,
		Cost:      0.005,
		CreatedAt: time.Now().Add(-25 * time.Hour),
	}

	m.mu.Lock()
	m.logs = append(m.logs, oldLog)
	fm, ok := m.metrics[oldLog.Feature]
	if !ok {
		fm = &FeatureMetrics{}
		m.metrics[oldLog.Feature] = fm
	}
	fm.TotalCalls++
	fm.TotalTokens += oldLog.Tokens
	fm.TotalCost += oldLog.Cost
	m.mu.Unlock()

	stats := m.Stats(24)

	// RecentLogs should only include the log within the 24h window.
	if len(stats.RecentLogs) != 1 {
		t.Fatalf("expected 1 recent log in window, got %d", len(stats.RecentLogs))
	}
	if stats.RecentLogs[0].Feature != "writing" {
		t.Errorf("expected 'writing' log in recent, got %q", stats.RecentLogs[0].Feature)
	}

	// ByFeature accumulates all-time metrics (both features still present).
	if len(stats.ByFeature) != 2 {
		t.Errorf("expected 2 features in ByFeature (all-time), got %d", len(stats.ByFeature))
	}
}
