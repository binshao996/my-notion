package search

import (
	"encoding/json"
	"testing"
)

// =============================================================================
// ExtractText tests
// =============================================================================

func TestExtractText_EmptyJSON(t *testing.T) {
	result := ExtractText("{}")
	if result != "" {
		t.Errorf("expected empty string from empty JSON, got %q", result)
	}
}

func TestExtractText_TextOnly(t *testing.T) {
	result := ExtractText(`{"text": "hello world"}`)
	if result != "hello world" {
		t.Errorf("expected %q, got %q", "hello world", result)
	}
}

func TestExtractText_TitleOnly(t *testing.T) {
	result := ExtractText(`{"title": "My Page"}`)
	if result != "My Page" {
		t.Errorf("expected %q, got %q", "My Page", result)
	}
}

func TestExtractText_TextAndTitle_TextHasPriority(t *testing.T) {
	result := ExtractText(`{"text": "body text", "title": "My Page"}`)
	if result != "body text" {
		t.Errorf("expected text to take priority, got %q", result)
	}
}

func TestExtractText_NestedTextObject(t *testing.T) {
	// Tiptap rich text stores "text" as a nested object, not a string.
	// Only string type is extracted; objects fall through.
	json := `{"text": {"type": "doc", "content": [{"text": "hello"}]}}`
	result := ExtractText(json)
	if result != "" {
		t.Errorf("expected empty string for nested text object, got %q", result)
	}
}

func TestExtractText_MalformedJSON(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"empty string", ""},
		{"plain text", "not json at all"},
		{"unclosed object", `{"text": "hello"`},
		{"trailing comma", `{"text": "hello",}`},
		{"single quote (invalid json)", `{'text': 'hello'}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractText(tt.input)
			if result != "" {
				t.Errorf("expected empty string for malformed JSON %q, got %q", tt.input, result)
			}
		})
	}
}

func TestExtractText_SpecialCharacters(t *testing.T) {
	result := ExtractText(`{"text": "hello \"world\""}`)
	if result != `hello "world"` {
		t.Errorf("expected %q, got %q", `hello "world"`, result)
	}
}

func TestExtractText_NumberValueInTextField(t *testing.T) {
	// The code only extracts string-typed "text". Numbers should be ignored.
	result := ExtractText(`{"text": 42}`)
	if result != "" {
		t.Errorf("expected empty string for numeric text field, got %q", result)
	}
}

func TestExtractText_EmptyTextString(t *testing.T) {
	result := ExtractText(`{"text": ""}`)
	if result != "" {
		t.Errorf("expected empty string from empty text, got %q", result)
	}
}

func TestExtractText_BooleanValueInTextField(t *testing.T) {
	result := ExtractText(`{"text": true}`)
	if result != "" {
		t.Errorf("expected empty string for boolean text field, got %q", result)
	}
}

func TestExtractText_ArrayValueInTextField(t *testing.T) {
	result := ExtractText(`{"text": ["line1", "line2"]}`)
	if result != "" {
		t.Errorf("expected empty string for array text field, got %q", result)
	}
}

func TestExtractText_NullValueInTextField(t *testing.T) {
	result := ExtractText(`{"text": null}`)
	if result != "" {
		t.Errorf("expected empty string for null text field, got %q", result)
	}
}

func TestExtractText_TextNestedButTitleString(t *testing.T) {
	// text is an object (skipped), title is a string (returned)
	result := ExtractText(`{"text": {"foo": "bar"}, "title": "Page Title"}`)
	if result != "Page Title" {
		t.Errorf("expected title %q when text is non-string, got %q", "Page Title", result)
	}
}

func TestExtractText_NumberValueInTitleField(t *testing.T) {
	result := ExtractText(`{"title": 123}`)
	if result != "" {
		t.Errorf("expected empty string for numeric title field, got %q", result)
	}
}

func TestExtractText_UnicodeText(t *testing.T) {
	result := ExtractText(`{"text": "こんにちは世界"}`)
	if result != "こんにちは世界" {
		t.Errorf("expected unicode text, got %q", result)
	}
}

func TestExtractText_EmojiText(t *testing.T) {
	result := ExtractText(`{"text": "🎉 Hello 🚀"}`)
	if result != "🎉 Hello 🚀" {
		t.Errorf("expected emoji text, got %q", result)
	}
}

func TestExtractText_WhitespaceOnly(t *testing.T) {
	result := ExtractText(`{"text": "   "}`)
	// Whitespace is still a valid string; the function does not trim.
	if result != "   " {
		t.Errorf("expected whitespace string preserved, got %q", result)
	}
}

func TestExtractText_NewlinesInText(t *testing.T) {
	result := ExtractText(`{"text": "line1\nline2"}`)
	if result != "line1\nline2" {
		t.Errorf("expected multiline text, got %q", result)
	}
}

func TestExtractText_TextFieldIsFloat(t *testing.T) {
	// float64 from json.Unmarshal
	result := ExtractText(`{"text": 3.14}`)
	if result != "" {
		t.Errorf("expected empty string for float text field, got %q", result)
	}
}

func TestExtractText_TitleFieldIsFloat(t *testing.T) {
	result := ExtractText(`{"title": 2.718}`)
	if result != "" {
		t.Errorf("expected empty string for float title field, got %q", result)
	}
}

func TestExtractText_OnlyOtherFields(t *testing.T) {
	// Neither "text" nor "title" present; other fields are ignored.
	result := ExtractText(`{"note": "some note", "desc": "some desc"}`)
	if result != "" {
		t.Errorf("expected empty string when text/title absent, got %q", result)
	}
}

// =============================================================================
// SearchResults / struct tests
// =============================================================================

func TestSearchResults_ZeroValue(t *testing.T) {
	var sr SearchResults
	if sr.Pages != nil {
		t.Errorf("expected nil Pages slice on zero value, got %v", sr.Pages)
	}
	if sr.Blocks != nil {
		t.Errorf("expected nil Blocks slice on zero value, got %v", sr.Blocks)
	}
	if sr.Records != nil {
		t.Errorf("expected nil Records slice on zero value, got %v", sr.Records)
	}
}

func TestSearchResults_JSONRoundTrip(t *testing.T) {
	original := SearchResults{
		Pages: []PageResult{
			{ID: 1, Title: "Page One", WorkspaceID: 10},
			{ID: 2, Title: "Page Two", WorkspaceID: 10},
		},
		Blocks: []BlockResult{
			{ID: 100, Text: "hello world", PageID: 1, WorkspaceID: 10, BlockType: "paragraph"},
		},
		Records: []RecordResult{
			{ID: 1000, Title: "Record A", DatabaseID: 50, WorkspaceID: 10},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal SearchResults: %v", err)
	}

	var decoded SearchResults
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal SearchResults: %v", err)
	}

	if len(decoded.Pages) != 2 {
		t.Errorf("expected 2 pages, got %d", len(decoded.Pages))
	}
	if decoded.Pages[0].Title != "Page One" {
		t.Errorf("expected 'Page One', got %q", decoded.Pages[0].Title)
	}
	if len(decoded.Blocks) != 1 {
		t.Errorf("expected 1 block, got %d", len(decoded.Blocks))
	}
	if decoded.Blocks[0].BlockType != "paragraph" {
		t.Errorf("expected 'paragraph', got %q", decoded.Blocks[0].BlockType)
	}
	if len(decoded.Records) != 1 {
		t.Errorf("expected 1 record, got %d", len(decoded.Records))
	}
	if decoded.Records[0].DatabaseID != 50 {
		t.Errorf("expected DatabaseID 50, got %d", decoded.Records[0].DatabaseID)
	}
}

func TestSearchResults_EmptySlicesMarshalAsJSONArray(t *testing.T) {
	sr := SearchResults{
		Pages:   []PageResult{},
		Blocks:  []BlockResult{},
		Records: []RecordResult{},
	}

	data, err := json.Marshal(sr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Empty slices should serialize as [] not null
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal to raw: %v", err)
	}

	if string(raw["pages"]) != "[]" {
		t.Errorf("expected pages to marshal as [], got %s", raw["pages"])
	}
	if string(raw["blocks"]) != "[]" {
		t.Errorf("expected blocks to marshal as [], got %s", raw["blocks"])
	}
	if string(raw["records"]) != "[]" {
		t.Errorf("expected records to marshal as [], got %s", raw["records"])
	}
}

func TestSearchResults_NilSlicesMarshalAsNull(t *testing.T) {
	var sr SearchResults

	data, err := json.Marshal(sr)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal to raw: %v", err)
	}

	if string(raw["pages"]) != "null" {
		t.Errorf("expected nil pages to marshal as null, got %s", raw["pages"])
	}
}

// =============================================================================
// PageResult tests
// =============================================================================

func TestPageResult_JSONRoundTrip(t *testing.T) {
	original := PageResult{ID: 42, Title: "Test Page", WorkspaceID: 7}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal PageResult: %v", err)
	}

	var decoded PageResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal PageResult: %v", err)
	}

	if decoded.ID != 42 {
		t.Errorf("expected ID 42, got %d", decoded.ID)
	}
	if decoded.Title != "Test Page" {
		t.Errorf("expected title 'Test Page', got %q", decoded.Title)
	}
	if decoded.WorkspaceID != 7 {
		t.Errorf("expected WorkspaceID 7, got %d", decoded.WorkspaceID)
	}
}

func TestPageResult_JSONFieldNames(t *testing.T) {
	original := PageResult{ID: 1, Title: "p", WorkspaceID: 2}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}

	// Verify expected JSON keys
	if _, ok := raw["id"]; !ok {
		t.Error("expected 'id' field in JSON")
	}
	if _, ok := raw["title"]; !ok {
		t.Error("expected 'title' field in JSON")
	}
	if _, ok := raw["workspace_id"]; !ok {
		t.Error("expected 'workspace_id' field in JSON")
	}
}

// =============================================================================
// BlockResult tests
// =============================================================================

func TestBlockResult_JSONRoundTrip(t *testing.T) {
	original := BlockResult{
		ID:          100,
		Text:        "Hello, world!",
		PageID:      42,
		WorkspaceID: 7,
		BlockType:   "paragraph",
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal BlockResult: %v", err)
	}

	var decoded BlockResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal BlockResult: %v", err)
	}

	if decoded.ID != 100 {
		t.Errorf("expected ID 100, got %d", decoded.ID)
	}
	if decoded.Text != "Hello, world!" {
		t.Errorf("expected text 'Hello, world!', got %q", decoded.Text)
	}
	if decoded.PageID != 42 {
		t.Errorf("expected PageID 42, got %d", decoded.PageID)
	}
	if decoded.BlockType != "paragraph" {
		t.Errorf("expected BlockType 'paragraph', got %q", decoded.BlockType)
	}
}

func TestBlockResult_JSONFieldNames(t *testing.T) {
	original := BlockResult{ID: 1, Text: "x", PageID: 2, WorkspaceID: 3, BlockType: "heading"}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}

	if _, ok := raw["block_type"]; !ok {
		t.Error("expected 'block_type' field in JSON")
	}
}

// =============================================================================
// RecordResult tests
// =============================================================================

func TestRecordResult_JSONRoundTrip(t *testing.T) {
	original := RecordResult{
		ID:          200,
		Title:       "Sample Record",
		DatabaseID:  10,
		WorkspaceID: 7,
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal RecordResult: %v", err)
	}

	var decoded RecordResult
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal RecordResult: %v", err)
	}

	if decoded.ID != 200 {
		t.Errorf("expected ID 200, got %d", decoded.ID)
	}
	if decoded.Title != "Sample Record" {
		t.Errorf("expected title 'Sample Record', got %q", decoded.Title)
	}
	if decoded.DatabaseID != 10 {
		t.Errorf("expected DatabaseID 10, got %d", decoded.DatabaseID)
	}
}

func TestRecordResult_JSONFieldNames(t *testing.T) {
	original := RecordResult{ID: 1, Title: "r", DatabaseID: 2, WorkspaceID: 3}
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}

	if _, ok := raw["database_id"]; !ok {
		t.Error("expected 'database_id' field in JSON")
	}
}

// =============================================================================
// ExtractText with real-world Tiptap JSON examples
// =============================================================================

func TestExtractText_TiptapParagraph(t *testing.T) {
	// Real Tiptap paragraph block props
	propsJSON := `{
		"text": {
			"type": "doc",
			"content": [
				{
					"type": "paragraph",
					"content": [
						{
							"type": "text",
							"text": "Hello, this is a paragraph."
						}
					]
				}
			]
		}
	}`

	result := ExtractText(propsJSON)
	// Since "text" is an object, not a string, the function returns "".
	// This is a known limitation — the code only extracts string-typed text.
	if result != "" {
		t.Errorf("expected empty string for Tiptap paragraph (object text), got %q", result)
	}
}

func TestExtractText_TiptapWithTitle(t *testing.T) {
	// A page-like block that has both the rich text content and a title
	propsJSON := `{
		"text": {
			"type": "doc",
			"content": [{"type": "paragraph", "content": [{"type": "text", "text": "body"}]}]
		},
		"title": "My Document Title"
	}`

	result := ExtractText(propsJSON)
	// text is an object (skipped), so title is returned
	if result != "My Document Title" {
		t.Errorf("expected 'My Document Title' (title fallback), got %q", result)
	}
}

// =============================================================================
// SearchJob queue struct JSON round-trip (from pkg/queue)
// =============================================================================

func TestSearchJob_JSONRoundTrip(t *testing.T) {
	// Since SearchJob is in pkg/queue, we test the equivalent concept here
	type searchJob struct {
		Type string `json:"type"`
		ID   uint   `json:"id"`
	}

	original := searchJob{Type: "page", ID: 42}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded searchJob
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Type != "page" {
		t.Errorf("expected type 'page', got %q", decoded.Type)
	}
	if decoded.ID != 42 {
		t.Errorf("expected ID 42, got %d", decoded.ID)
	}

	// Verify expected JSON shape
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		t.Fatalf("unmarshal raw: %v", err)
	}
	if string(raw["type"]) != `"page"` {
		t.Errorf("expected type JSON %q, got %s", `"page"`, raw["type"])
	}
	// uint marshals as a JSON number
	if string(raw["id"]) != "42" {
		t.Errorf("expected id JSON %q, got %s", "42", raw["id"])
	}
}

func TestSearchJob_ZeroValue(t *testing.T) {
	type searchJob struct {
		Type string `json:"type"`
		ID   uint   `json:"id"`
	}

	job := searchJob{}
	data, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded searchJob
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Type != "" {
		t.Errorf("expected empty type on zero value, got %q", decoded.Type)
	}
	if decoded.ID != 0 {
		t.Errorf("expected ID 0 on zero value, got %d", decoded.ID)
	}
}
