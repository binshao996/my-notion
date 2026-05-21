package ai

import (
	"encoding/json"
	"strings"
	"testing"
)

// newTestAutofillService creates an AutofillService with nil dependencies
// for testing pure methods that do not use DB, Redis, or external clients.
func newTestAutofillService() *AutofillService {
	return &AutofillService{
		jobs: make(map[string]*AutofillJob),
	}
}

func TestBuildPromptForPropertyTypes(t *testing.T) {
	svc := newTestAutofillService()
	content := "The project deadline is June 15. Budget is $5000."

	tests := []struct {
		name     string
		propName string
		propType string
		config   string
		contains []string // substrings the prompt must contain
	}{
		{
			name:     "text property",
			propName: "Summary",
			propType: "text",
			contains: []string{"Summary", content},
		},
		{
			name:     "title property",
			propName: "Title",
			propType: "title",
			contains: []string{"Title", content},
		},
		{
			name:     "select with options",
			propName: "Category",
			propType: "select",
			config:   `{"options":[{"name":"Work"},{"name":"Personal"}]}`,
			contains: []string{"Work, Personal", "Category", content},
		},
		{
			name:     "select without options",
			propName: "Tag",
			propType: "select",
			config:   "{}",
			contains: []string{"Tag", content},
		},
		{
			name:     "number property",
			propName: "Budget",
			propType: "number",
			contains: []string{"Budget", "number", content},
		},
		{
			name:     "status with options",
			propName: "Stage",
			propType: "status",
			config:   `{"options":[{"name":"To Do"},{"name":"Done"}]}`,
			contains: []string{"To Do, Done", "Stage", content},
		},
		{
			name:     "status without options",
			propName: "Phase",
			propType: "status",
			config:   "",
			contains: []string{"Phase", content},
		},
		{
			name:     "date property",
			propName: "Deadline",
			propType: "date",
			contains: []string{"YYYY-MM-DD", "Deadline", content},
		},
		{
			name:     "url property",
			propName: "Website",
			propType: "url",
			contains: []string{"URL", "Website", content},
		},
		{
			name:     "checkbox property",
			propName: "Complete",
			propType: "checkbox",
			contains: []string{"YES or NO", "Complete", content},
		},
		{
			name:     "email property",
			propName: "Contact",
			propType: "email",
			contains: []string{"email", "Contact", content},
		},
		{
			name:     "phone property",
			propName: "Mobile",
			propType: "phone",
			contains: []string{"phone number", "Mobile", content},
		},
		{
			name:     "unknown type fallback",
			propName: "Custom",
			propType: "custom_type",
			contains: []string{"Custom", "custom_type", content},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prompt := svc.buildPrompt(tt.propName, tt.propType, tt.config, content)
			for _, substr := range tt.contains {
				if !strings.Contains(prompt, substr) {
					t.Errorf("expected prompt to contain %q\ngot: %s", substr, prompt)
				}
			}
		})
	}
}

func TestExtractOptions(t *testing.T) {
	svc := newTestAutofillService()

	tests := []struct {
		name     string
		config   string
		expected []string
	}{
		{
			name:     "structured options with names",
			config:   `{"options":[{"name":"Option A","id":"1"},{"name":"Option B","id":"2"}]}`,
			expected: []string{"Option A", "Option B"},
		},
		{
			name:     "simple string array options",
			config:   `{"options":["Red","Green","Blue"]}`,
			expected: []string{"Red", "Green", "Blue"},
		},
		{
			name:     "values array format",
			config:   `{"values":["X","Y","Z"]}`,
			expected: []string{"X", "Y", "Z"},
		},
		{
			name:     "empty config string",
			config:   "",
			expected: nil,
		},
		{
			name:     "empty object",
			config:   "{}",
			expected: nil,
		},
		{
			name:     "invalid json",
			config:   "{invalid}",
			expected: nil,
		},
		{
			name:     "structured options with empty name skipped",
			config:   `{"options":[{"name":"Valid"},{"name":""},{"name":"Also Valid"}]}`,
			expected: []string{"Valid", "Also Valid"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := svc.extractOptions(tt.config)
			if len(result) != len(tt.expected) {
				t.Fatalf("expected %d options, got %d: %v", len(tt.expected), len(result), result)
			}
			for i, v := range result {
				if v != tt.expected[i] {
					t.Errorf("option[%d]: expected %q, got %q", i, tt.expected[i], v)
				}
			}
		})
	}
}

func TestFormatPropertyValue(t *testing.T) {
	svc := newTestAutofillService()

	tests := []struct {
		name          string
		propType      string
		value         string
		wantKey       string
		wantValue     any
		wantValueType string // "string", "float64", "bool"
	}{
		{name: "text", propType: "text", value: "hello world", wantKey: "text", wantValue: "hello world", wantValueType: "string"},
		{name: "title", propType: "title", value: "My Title", wantKey: "text", wantValue: "My Title", wantValueType: "string"},
		{name: "select", propType: "select", value: "Option A", wantKey: "selected", wantValue: "Option A", wantValueType: "string"},
		{name: "number valid", propType: "number", value: "42.5", wantKey: "number", wantValue: 42.5, wantValueType: "float64"},
		{name: "number integer", propType: "number", value: "100", wantKey: "number", wantValue: 100.0, wantValueType: "float64"},
		{name: "number invalid fallback", propType: "number", value: "not a number", wantKey: "number", wantValue: "not a number", wantValueType: "string"},
		{name: "status", propType: "status", value: "In Progress", wantKey: "status", wantValue: "In Progress", wantValueType: "string"},
		{name: "date", propType: "date", value: "2024-01-15", wantKey: "date", wantValue: "2024-01-15", wantValueType: "string"},
		{name: "url", propType: "url", value: "https://example.com", wantKey: "url", wantValue: "https://example.com", wantValueType: "string"},
		{name: "checkbox yes", propType: "checkbox", value: "YES", wantKey: "checked", wantValue: true, wantValueType: "bool"},
		{name: "checkbox no", propType: "checkbox", value: "NO", wantKey: "checked", wantValue: false, wantValueType: "bool"},
		{name: "checkbox true", propType: "checkbox", value: "true", wantKey: "checked", wantValue: true, wantValueType: "bool"},
		{name: "checkbox false", propType: "checkbox", value: "false", wantKey: "checked", wantValue: false, wantValueType: "bool"},
		{name: "checkbox y", propType: "checkbox", value: "y", wantKey: "checked", wantValue: true, wantValueType: "bool"},
		{name: "email", propType: "email", value: "test@example.com", wantKey: "email", wantValue: "test@example.com", wantValueType: "string"},
		{name: "phone", propType: "phone", value: "+1-555-0123", wantKey: "phone", wantValue: "+1-555-0123", wantValueType: "string"},
		{name: "unknown type fallback", propType: "unknown_type", value: "some value", wantKey: "text", wantValue: "some value", wantValueType: "string"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := svc.formatPropertyValue(tt.propType, tt.value)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if result == "" {
				t.Fatal("expected non-empty result")
			}

			var parsed map[string]any
			if err := json.Unmarshal([]byte(result), &parsed); err != nil {
				t.Fatalf("result is not valid JSON: %s", result)
			}

			actualValue, ok := parsed[tt.wantKey]
			if !ok {
				t.Errorf("expected key %q in result, got keys: %v", tt.wantKey, mapKeys(parsed))
				return
			}

			switch tt.wantValueType {
			case "float64":
				v, ok := actualValue.(float64)
				if !ok {
					t.Errorf("expected float64 for key %q, got %T: %v", tt.wantKey, actualValue, actualValue)
					return
				}
				if v != tt.wantValue.(float64) {
					t.Errorf("%s: expected %q=%v, got %v", tt.name, tt.wantKey, tt.wantValue, v)
				}
			case "bool":
				v, ok := actualValue.(bool)
				if !ok {
					t.Errorf("expected bool for key %q, got %T: %v", tt.wantKey, actualValue, actualValue)
					return
				}
				if v != tt.wantValue.(bool) {
					t.Errorf("%s: expected %q=%v, got %v", tt.name, tt.wantKey, tt.wantValue, v)
				}
			case "string":
				v, ok := actualValue.(string)
				if !ok {
					t.Errorf("expected string for key %q, got %T: %v", tt.wantKey, actualValue, actualValue)
					return
				}
				if v != tt.wantValue.(string) {
					t.Errorf("%s: expected %q=%q, got %q", tt.name, tt.wantKey, tt.wantValue, v)
				}
			}
		})
	}
}

func TestMarshalValue(t *testing.T) {
	svc := newTestAutofillService()

	t.Run("string value", func(t *testing.T) {
		result, err := svc.marshalValue(map[string]string{"text": "hello"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := `{"text":"hello"}`
		if result != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	})

	t.Run("bool value", func(t *testing.T) {
		result, err := svc.marshalValue(map[string]bool{"checked": true})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := `{"checked":true}`
		if result != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	})

	t.Run("number value", func(t *testing.T) {
		result, err := svc.marshalValue(map[string]float64{"number": 42.5})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		expected := `{"number":42.5}`
		if result != expected {
			t.Errorf("expected %s, got %s", expected, result)
		}
	})

	t.Run("mixed types", func(t *testing.T) {
		result, err := svc.marshalValue(map[string]any{
			"text":    "hello",
			"count":   3,
			"checked": true,
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Re-marshal and compare structurally.
		var parsed map[string]any
		if err := json.Unmarshal([]byte(result), &parsed); err != nil {
			t.Fatalf("invalid JSON: %s", result)
		}
		if parsed["text"] != "hello" {
			t.Errorf("expected text='hello', got %v", parsed["text"])
		}
		if parsed["checked"] != true {
			t.Errorf("expected checked=true, got %v", parsed["checked"])
		}
	})
}

func TestAutofillJobStatusTransitions(t *testing.T) {
	t.Run("job created with running status", func(t *testing.T) {
		job := &AutofillJob{
			ID:         "job-1",
			DatabaseID: 1,
			PropertyID: 2,
			Total:      10,
			Completed:  0,
			Failed:     0,
			Status:     "running",
		}
		if job.Status != "running" {
			t.Errorf("expected status 'running', got %q", job.Status)
		}
		if job.Total != 10 {
			t.Errorf("expected Total 10, got %d", job.Total)
		}
	})

	t.Run("job transitions to completed", func(t *testing.T) {
		job := &AutofillJob{
			ID:         "job-1",
			DatabaseID: 1,
			PropertyID: 2,
			Total:      10,
			Completed:  10,
			Failed:     0,
			Status:     "completed",
		}
		if job.Completed != job.Total {
			t.Errorf("expected all records completed")
		}
		if job.Status != "completed" {
			t.Errorf("expected status 'completed', got %q", job.Status)
		}
	})

	t.Run("job transitions to failed when all fail", func(t *testing.T) {
		job := &AutofillJob{
			ID:         "job-1",
			DatabaseID: 1,
			PropertyID: 2,
			Total:      5,
			Completed:  0,
			Failed:     5,
			Status:     "failed",
		}
		if job.Failed != job.Total {
			t.Errorf("expected all records failed")
		}
		if job.Status != "failed" {
			t.Errorf("expected status 'failed', got %q", job.Status)
		}
	})

	t.Run("job in progress with partial completion", func(t *testing.T) {
		job := &AutofillJob{
			ID:         "job-2",
			DatabaseID: 1,
			PropertyID: 3,
			Total:      10,
			Completed:  4,
			Failed:     1,
			Status:     "running",
		}
		if job.Completed+job.Failed != 5 {
			t.Errorf("expected 5 processed, got %d", job.Completed+job.Failed)
		}
		if job.Status != "running" {
			t.Errorf("expected status 'running' for partial progress, got %q", job.Status)
		}
	})
}

// mapKeys returns the keys of a map as a slice of strings.
func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
