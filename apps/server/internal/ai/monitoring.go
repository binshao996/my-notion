package ai

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"
)

// Monitor tracks AI usage and costs across all features.
type Monitor struct {
	mu      sync.Mutex
	logs    []CostLog
	metrics map[string]*FeatureMetrics
}

// FeatureMetrics holds aggregated stats for a single feature.
type FeatureMetrics struct {
	TotalCalls   int     `json:"total_calls"`
	TotalTokens  int     `json:"total_tokens"`
	TotalCost    float64 `json:"total_cost"`
	AvgLatencyMs int64   `json:"avg_latency_ms"`
	ErrorCount   int     `json:"error_count"`
}

// MonitorStats is the full monitoring snapshot returned by the API.
type MonitorStats struct {
	TotalCost   float64                    `json:"total_cost"`
	TotalTokens int                        `json:"total_tokens"`
	TotalCalls  int                        `json:"total_calls"`
	ByFeature   map[string]*FeatureMetrics `json:"by_feature"`
	RecentLogs  []CostLog                  `json:"recent_logs"`
	WindowHours int                        `json:"window_hours"`
}

// NewMonitor creates a new Monitor.
func NewMonitor() *Monitor {
	return &Monitor{
		logs:    make([]CostLog, 0),
		metrics: make(map[string]*FeatureMetrics),
	}
}

// Record logs a cost entry and updates feature metrics.
func (m *Monitor) Record(log CostLog) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Set timestamp if not already set.
	if log.CreatedAt.IsZero() {
		log.CreatedAt = time.Now()
	}

	m.logs = append(m.logs, log)

	// Update feature metrics.
	fm, ok := m.metrics[log.Feature]
	if !ok {
		fm = &FeatureMetrics{}
		m.metrics[log.Feature] = fm
	}
	fm.TotalCalls++
	fm.TotalTokens += log.Tokens
	fm.TotalCost += log.Cost

	// Keep only last 1000 logs.
	if len(m.logs) > 1000 {
		m.logs = m.logs[len(m.logs)-1000:]
	}
}

// Stats returns current monitoring statistics filtered by the given time window.
func (m *Monitor) Stats(windowHours int) *MonitorStats {
	m.mu.Lock()
	defer m.mu.Unlock()

	cutoff := time.Now().Add(-time.Duration(windowHours) * time.Hour)
	var recent []CostLog
	for _, l := range m.logs {
		if l.CreatedAt.After(cutoff) {
			recent = append(recent, l)
		}
	}

	stats := &MonitorStats{
		ByFeature:   make(map[string]*FeatureMetrics),
		RecentLogs:  recent,
		WindowHours: windowHours,
	}

	// Copy metrics.
	for k, v := range m.metrics {
		stats.ByFeature[k] = &FeatureMetrics{
			TotalCalls:   v.TotalCalls,
			TotalTokens:  v.TotalTokens,
			TotalCost:    v.TotalCost,
			AvgLatencyMs: v.AvgLatencyMs,
			ErrorCount:   v.ErrorCount,
		}
		stats.TotalCost += v.TotalCost
		stats.TotalTokens += v.TotalTokens
		stats.TotalCalls += v.TotalCalls
	}

	return stats
}

// ServeHTTP serves the monitoring dashboard as JSON.
func (m *Monitor) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	hours := 24 // default 24h window

	if h := r.URL.Query().Get("hours"); h != "" {
		if parsed, err := strconv.Atoi(h); err == nil && parsed > 0 {
			hours = parsed
		}
	}

	stats := m.Stats(hours)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	json.NewEncoder(w).Encode(stats)
}
