package ai

import (
	"math"
	"testing"
)

// =============================================================================
// LoadConfig tests
// =============================================================================

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name        string
		baseURL     string
		model       string
		apiKey      string
		wantBaseURL string
		wantModel   string
		wantAPIKey  string
	}{
		{
			name:        "with DEEPSEEK_BASE_URL set",
			baseURL:     "https://custom.api.com",
			wantBaseURL: "https://custom.api.com",
			wantModel:   "deepseek-v4-flash",
			wantAPIKey:  "",
		},
		{
			name:        "without DEEPSEEK_BASE_URL defaults",
			wantBaseURL: "https://api.deepseek.com",
			wantModel:   "deepseek-v4-flash",
			wantAPIKey:  "",
		},
		{
			name:        "with DEEPSEEK_MODEL set",
			model:       "deepseek-chat",
			wantBaseURL: "https://api.deepseek.com",
			wantModel:   "deepseek-chat",
			wantAPIKey:  "",
		},
		{
			name:        "without DEEPSEEK_MODEL defaults",
			wantBaseURL: "https://api.deepseek.com",
			wantModel:   "deepseek-v4-flash",
			wantAPIKey:  "",
		},
		{
			name:        "with DEEPSEEK_API_KEY set",
			apiKey:      "sk-test-key-123",
			wantBaseURL: "https://api.deepseek.com",
			wantModel:   "deepseek-v4-flash",
			wantAPIKey:  "sk-test-key-123",
		},
		{
			name:        "without DEEPSEEK_API_KEY empty",
			wantBaseURL: "https://api.deepseek.com",
			wantModel:   "deepseek-v4-flash",
			wantAPIKey:  "",
		},
		{
			name:        "all custom values",
			baseURL:     "https://my.deepseek.proxy",
			model:       "deepseek-v5",
			apiKey:      "sk-all-custom",
			wantBaseURL: "https://my.deepseek.proxy",
			wantModel:   "deepseek-v5",
			wantAPIKey:  "sk-all-custom",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("DEEPSEEK_BASE_URL", tt.baseURL)
			t.Setenv("DEEPSEEK_MODEL", tt.model)
			t.Setenv("DEEPSEEK_API_KEY", tt.apiKey)

			cfg := LoadConfig()

			if cfg.BaseURL != tt.wantBaseURL {
				t.Errorf("BaseURL = %q, want %q", cfg.BaseURL, tt.wantBaseURL)
			}
			if cfg.Model != tt.wantModel {
				t.Errorf("Model = %q, want %q", cfg.Model, tt.wantModel)
			}
			if cfg.APIKey != tt.wantAPIKey {
				t.Errorf("APIKey = %q, want %q", cfg.APIKey, tt.wantAPIKey)
			}
		})
	}
}

// =============================================================================
// NewClient tests
// =============================================================================

func TestNewClient(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantNil bool
	}{
		{
			name:    "nil config returns nil",
			config:  nil,
			wantNil: true,
		},
		{
			name:    "empty APIKey returns nil",
			config:  &Config{APIKey: ""},
			wantNil: true,
		},
		{
			name:    "config with APIKey returns non-nil client",
			config:  &Config{APIKey: "sk-test"},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.config)
			if (client == nil) != tt.wantNil {
				t.Errorf("NewClient() = %v, want nil=%v", client, tt.wantNil)
			}
			// When client is non-nil, verify config is stored correctly.
			if !tt.wantNil && client.config != tt.config {
				t.Error("client.config does not match input config")
			}
		})
	}
}

// =============================================================================
// IsAvailable tests
// =============================================================================

func TestIsAvailable(t *testing.T) {
	tests := []struct {
		name   string
		config *Config
		want   bool
	}{
		{
			name:   "nil config returns false",
			config: nil,
			want:   false,
		},
		{
			name:   "config with empty APIKey returns false",
			config: &Config{APIKey: ""},
			want:   false,
		},
		{
			name:   "config with APIKey returns true",
			config: &Config{APIKey: "sk-available"},
			want:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := NewClient(tt.config)
			if got := client.IsAvailable(); got != tt.want {
				t.Errorf("IsAvailable() = %v, want %v", got, tt.want)
			}
		})
	}
}

// =============================================================================
// ChatCompletion model default tests
// =============================================================================

func TestChatCompletion_ModelDefault(t *testing.T) {
	cfg := &Config{
		APIKey:  "sk-test",
		BaseURL: "http://127.0.0.1:19999", // nothing listening → fast connection refused
		Model:   "deepseek-v4-flash",
	}
	client := NewClient(cfg)
	if client == nil {
		t.Fatal("expected non-nil client")
	}

	req := &ChatRequest{
		Messages: []ChatMessage{{Role: "user", Content: "hello"}},
		// Model intentionally left empty — should be defaulted.
	}

	_, err := client.ChatCompletion(req)
	// An HTTP error is expected because there is no real server.
	if err == nil {
		t.Log("unexpected HTTP success (maybe port 19999 is occupied)")
	}

	if req.Model != cfg.Model {
		t.Errorf("req.Model = %q, want %q (should default to config.Model)", req.Model, cfg.Model)
	}
}

// =============================================================================
// EstimateCost tests
// =============================================================================

func TestEstimateCost(t *testing.T) {
	tests := []struct {
		name             string
		model            string
		promptTokens     int
		completionTokens int
		want             float64
	}{
		{
			name:             "1000 prompt + 500 completion = 0.00028",
			model:            "deepseek-chat",
			promptTokens:     1000,
			completionTokens: 500,
			want:             0.00028,
		},
		{
			name:         "1M prompt only = 0.14",
			model:        "deepseek-chat",
			promptTokens: 1_000_000,
			want:         0.14,
		},
		{
			name:             "1M completion only = 0.28",
			model:            "deepseek-chat",
			completionTokens: 1_000_000,
			want:             0.28,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := EstimateCost(tt.model, tt.promptTokens, tt.completionTokens)
			if math.Abs(got-tt.want) > 0.0000001 {
				t.Errorf("EstimateCost(%q, %d, %d) = %v, want %v",
					tt.model, tt.promptTokens, tt.completionTokens, got, tt.want)
			}
		})
	}
}
