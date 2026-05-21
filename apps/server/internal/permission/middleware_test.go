package permission

import "testing"

func TestParseUint(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    uint
		wantErr bool
	}{
		{"zero", "0", 0, false},
		{"positive", "42", 42, false},
		{"max page id", "9999999", 9999999, false},
		{"empty", "", 0, false},
		{"negative", "-1", 0, true},
		{"alpha", "abc", 0, true},
		{"mixed", "12x34", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseUint(tt.input)
			if tt.wantErr && err == nil {
				t.Errorf("parseUint(%q) expected error", tt.input)
			}
			if !tt.wantErr && err != nil {
				t.Errorf("parseUint(%q) unexpected error: %v", tt.input, err)
			}
			if got != tt.want {
				t.Errorf("parseUint(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}
