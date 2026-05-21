package ai

import (
	"fmt"
	"math"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// parseCitations tests
// ---------------------------------------------------------------------------

func TestParseCitations_SingleCitation(t *testing.T) {
	svc := &RAGService{}
	results := []SearchResult{
		{PageID: 10, BlockID: 100, PageTitle: "Page A", Text: "Content of page A"},
	}

	citations := svc.parseCitations("The answer is [1] according to the docs.", results)

	if len(citations) != 1 {
		t.Fatalf("expected 1 citation, got %d", len(citations))
	}
	c := citations[0]
	if c.PageID != 10 {
		t.Errorf("expected PageID 10, got %d", c.PageID)
	}
	if c.BlockID != 100 {
		t.Errorf("expected BlockID 100, got %d", c.BlockID)
	}
	if c.Title != "Page A" {
		t.Errorf("expected Title 'Page A', got %q", c.Title)
	}
}

func TestParseCitations_MultipleCitations(t *testing.T) {
	svc := &RAGService{}
	results := []SearchResult{
		{PageID: 1, BlockID: 11, PageTitle: "First", Text: "First page text"},
		{PageID: 2, BlockID: 22, PageTitle: "Second", Text: "Second page text"},
		{PageID: 3, BlockID: 33, PageTitle: "Third", Text: "Third page text"},
	}

	citations := svc.parseCitations("See [1], [2], and [3] for details.", results)

	if len(citations) != 3 {
		t.Fatalf("expected 3 citations, got %d", len(citations))
	}
	for i, expectedID := range []uint{1, 2, 3} {
		if citations[i].PageID != expectedID {
			t.Errorf("citation[%d] expected PageID %d, got %d", i, expectedID, citations[i].PageID)
		}
	}
}

func TestParseCitations_OutOfRangeCitation(t *testing.T) {
	svc := &RAGService{}
	results := []SearchResult{
		{PageID: 1, BlockID: 11, PageTitle: "Page 1", Text: "Text 1"},
		{PageID: 2, BlockID: 22, PageTitle: "Page 2", Text: "Text 2"},
		{PageID: 3, BlockID: 33, PageTitle: "Page 3", Text: "Text 3"},
	}

	// [99] is out of range; it should be silently ignored. [2] is valid.
	citations := svc.parseCitations("Ref [99] is invalid, but [2] is fine.", results)

	if len(citations) != 1 {
		t.Fatalf("expected 1 citation (out-of-range ignored), got %d", len(citations))
	}
	if citations[0].PageID != 2 {
		t.Errorf("expected PageID 2, got %d", citations[0].PageID)
	}
}

func TestParseCitations_OutOfRangeZero(t *testing.T) {
	svc := &RAGService{}
	results := []SearchResult{
		{PageID: 1, BlockID: 11, PageTitle: "Only", Text: "Only page"},
	}

	// [0] is out of range (index -1)
	citations := svc.parseCitations("Look at [0].", results)

	if len(citations) != 0 {
		t.Fatalf("expected 0 citations for [0], got %d", len(citations))
	}
}

func TestParseCitations_DuplicatePageIDs(t *testing.T) {
	svc := &RAGService{}
	results := []SearchResult{
		{PageID: 1, BlockID: 11, PageTitle: "Page One", Text: "First block from page 1"},
		{PageID: 1, BlockID: 12, PageTitle: "Page One", Text: "Second block from page 1"},
		{PageID: 2, BlockID: 21, PageTitle: "Page Two", Text: "Block from page 2"},
	}

	// Both [1] and [2] point to the same PageID (1), only first should appear.
	citations := svc.parseCitations("Citing [1] and also [2] and [3].", results)

	if len(citations) != 2 {
		t.Fatalf("expected 2 citations (deduplicated by page ID), got %d", len(citations))
	}
	if citations[0].PageID != 1 {
		t.Errorf("first citation expected PageID 1, got %d", citations[0].PageID)
	}
	if citations[1].PageID != 2 {
		t.Errorf("second citation expected PageID 2, got %d", citations[1].PageID)
	}
}

func TestParseCitations_SnippetTruncation(t *testing.T) {
	svc := &RAGService{}
	longText := strings.Repeat("x", 250)
	results := []SearchResult{
		{PageID: 5, BlockID: 55, PageTitle: "Long", Text: longText},
	}

	citations := svc.parseCitations("See [1].", results)

	if len(citations) != 1 {
		t.Fatalf("expected 1 citation, got %d", len(citations))
	}
	expectedSnippet := strings.Repeat("x", 200) + "..."
	if citations[0].Snippet != expectedSnippet {
		t.Errorf("expected snippet of length %d ending with '...', got length %d (%q)",
			len(expectedSnippet), len(citations[0].Snippet), citations[0].Snippet)
	}
}

func TestParseCitations_SnippetExactly200Chars(t *testing.T) {
	svc := &RAGService{}
	exactText := strings.Repeat("y", 200)
	results := []SearchResult{
		{PageID: 5, BlockID: 55, PageTitle: "Exact", Text: exactText},
	}

	citations := svc.parseCitations("See [1].", results)

	if len(citations) != 1 {
		t.Fatalf("expected 1 citation, got %d", len(citations))
	}
	// Exactly 200 chars — no truncation
	if citations[0].Snippet != exactText {
		t.Errorf("expected snippet to match exact 200-char text, got %q", citations[0].Snippet)
	}
	if strings.HasSuffix(citations[0].Snippet, "...") {
		t.Error("200-char snippet should NOT be truncated")
	}
}

func TestParseCitations_ShortSnippetNoTruncation(t *testing.T) {
	svc := &RAGService{}
	shortText := "Short text"
	results := []SearchResult{
		{PageID: 1, BlockID: 11, PageTitle: "Short", Text: shortText},
	}

	citations := svc.parseCitations("See [1].", results)

	if len(citations) != 1 {
		t.Fatalf("expected 1 citation, got %d", len(citations))
	}
	if citations[0].Snippet != shortText {
		t.Errorf("expected snippet %q, got %q", shortText, citations[0].Snippet)
	}
}

func TestParseCitations_NoCitations(t *testing.T) {
	svc := &RAGService{}
	results := []SearchResult{
		{PageID: 1, BlockID: 11, PageTitle: "Page", Text: "Some text"},
	}

	citations := svc.parseCitations("This answer has no citation markers.", results)

	if citations != nil {
		t.Fatalf("expected nil for no citations, got %v", citations)
	}
}

func TestParseCitations_NonNumericBrackets(t *testing.T) {
	svc := &RAGService{}
	results := []SearchResult{
		{PageID: 1, BlockID: 11, PageTitle: "Page", Text: "Some text"},
	}

	// [note] is not a numeric citation and should be ignored by the regex.
	citations := svc.parseCitations("See [note] for more details.", results)

	if citations != nil {
		t.Fatalf("expected nil for non-numeric brackets, got %v", citations)
	}
}

func TestParseCitations_MixedValidAndInvalid(t *testing.T) {
	svc := &RAGService{}
	results := []SearchResult{
		{PageID: 1, BlockID: 11, PageTitle: "First", Text: "First text"},
		{PageID: 2, BlockID: 22, PageTitle: "Second", Text: "Second text"},
	}

	// [note] ignored, [99] out of range, [1] and [2] valid.
	citations := svc.parseCitations("Text with [note], [99], [1], and [2].", results)

	if len(citations) != 2 {
		t.Fatalf("expected 2 citations, got %d", len(citations))
	}
	if citations[0].PageID != 1 {
		t.Errorf("expected first citation PageID 1, got %d", citations[0].PageID)
	}
	if citations[1].PageID != 2 {
		t.Errorf("expected second citation PageID 2, got %d", citations[1].PageID)
	}
}

func TestParseCitations_EmptyResults(t *testing.T) {
	svc := &RAGService{}
	var results []SearchResult

	citations := svc.parseCitations("See [1] and [2].", results)

	if citations != nil {
		t.Fatalf("expected nil when results are empty, got %v", citations)
	}
}

func TestParseCitations_EmptyAnswer(t *testing.T) {
	svc := &RAGService{}
	results := []SearchResult{
		{PageID: 1, BlockID: 11, PageTitle: "Page", Text: "Text"},
	}

	citations := svc.parseCitations("", results)

	if citations != nil {
		t.Fatalf("expected nil for empty answer, got %v", citations)
	}
}

func TestParseCitations_MultiDigitCitation(t *testing.T) {
	svc := &RAGService{}
	results := make([]SearchResult, 12)
	for i := range results {
		results[i] = SearchResult{
			PageID: uint(i + 1), BlockID: uint((i + 1) * 10),
			PageTitle: fmt.Sprintf("Page %d", i+1),
			Text:      fmt.Sprintf("Text %d", i+1),
		}
	}

	citations := svc.parseCitations("See [10] and [12].", results)

	if len(citations) != 2 {
		t.Fatalf("expected 2 citations, got %d", len(citations))
	}
	if citations[0].PageID != 10 {
		t.Errorf("expected first citation PageID 10, got %d", citations[0].PageID)
	}
	if citations[1].PageID != 12 {
		t.Errorf("expected second citation PageID 12, got %d", citations[1].PageID)
	}
}

func TestParseCitations_CitationWithoutSpace(t *testing.T) {
	svc := &RAGService{}
	results := []SearchResult{
		{PageID: 1, BlockID: 11, PageTitle: "Page", Text: "Some text"},
	}

	// Adjacent text without spaces around the citation bracket.
	citations := svc.parseCitations("important[1]point", results)

	if len(citations) != 1 {
		t.Fatalf("expected 1 citation for adjacent text, got %d", len(citations))
	}
	if citations[0].PageID != 1 {
		t.Errorf("expected PageID 1, got %d", citations[0].PageID)
	}
}

// ---------------------------------------------------------------------------
// Embedder Tokenize tests
// ---------------------------------------------------------------------------

func TestEmbedder_Tokenize_Empty(t *testing.T) {
	e := NewEmbedder()
	tokens := e.Tokenize("")

	if len(tokens) != 0 {
		t.Fatalf("expected 0 tokens for empty string, got %d", len(tokens))
	}
}

func TestEmbedder_Tokenize_SimpleText(t *testing.T) {
	e := NewEmbedder()
	tokens := e.Tokenize("hello world")

	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d: %v", len(tokens), tokens)
	}
	if tokens[0] != "hello" {
		t.Errorf("expected 'hello', got %q", tokens[0])
	}
	if tokens[1] != "world" {
		t.Errorf("expected 'world', got %q", tokens[1])
	}
}

func TestEmbedder_Tokenize_StopWordsExcluded(t *testing.T) {
	e := NewEmbedder()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "article the",
			input:    "the quick brown fox",
			expected: []string{"quick", "brown", "fox"},
		},
		{
			name:     "articles and prepositions",
			input:    "a story about the king and the queen",
			expected: []string{"story", "king", "queen"},
		},
		{
			name:     "verb to be",
			input:    "this is a test of the system",
			expected: []string{"test", "system"},
		},
		{
			name:     "pronouns",
			input:    "i think you are right and they are wrong",
			expected: []string{"think", "right", "wrong"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := e.Tokenize(tt.input)
			if len(tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d: %v", len(tt.expected), len(tokens), tokens)
			}
			for i, tok := range tt.expected {
				if tokens[i] != tok {
					t.Errorf("token[%d]: expected %q, got %q", i, tok, tokens[i])
				}
			}
		})
	}
}

func TestEmbedder_Tokenize_PunctuationRemoved(t *testing.T) {
	e := NewEmbedder()

	tests := []struct {
		name     string
		input    string
		expected []string
	}{
		{
			name:     "commas and periods",
			input:    "hello, world. how do things work?",
			expected: []string{"hello", "world", "how", "things", "work"},
		},
		{
			name:     "brackets and quotes",
			input:    `the "quick" (brown) fox`,
			expected: []string{"quick", "brown", "fox"},
		},
		{
			name:     "digits preserved",
			input:    "error code 404 not found",
			expected: []string{"error", "code", "404", "found"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens := e.Tokenize(tt.input)
			if len(tokens) != len(tt.expected) {
				t.Fatalf("expected %d tokens, got %d: %v", len(tt.expected), len(tokens), tokens)
			}
			for i, tok := range tt.expected {
				if tokens[i] != tok {
					t.Errorf("token[%d]: expected %q, got %q", i, tok, tokens[i])
				}
			}
		})
	}
}

func TestEmbedder_Tokenize_SingleCharacterTokensExcluded(t *testing.T) {
	e := NewEmbedder()

	// "a" is a stop word, "x" is a single char
	tokens := e.Tokenize("x is a test")
	if len(tokens) != 1 {
		t.Fatalf("expected 1 token ('test'), got %d: %v", len(tokens), tokens)
	}
	if tokens[0] != "test" {
		t.Errorf("expected 'test', got %q", tokens[0])
	}
}

func TestEmbedder_Tokenize_CaseInsensitive(t *testing.T) {
	e := NewEmbedder()
	tokens := e.Tokenize("HELLO The World")

	// "The" is a stop word when lowercased to "the"
	if len(tokens) != 2 {
		t.Fatalf("expected 2 tokens, got %d: %v", len(tokens), tokens)
	}
	if tokens[0] != "hello" {
		t.Errorf("expected 'hello', got %q", tokens[0])
	}
	if tokens[1] != "world" {
		t.Errorf("expected 'world', got %q", tokens[1])
	}
}

// ---------------------------------------------------------------------------
// Embedder Vectorize tests
// ---------------------------------------------------------------------------

func TestVectorize_EmptyText(t *testing.T) {
	e := NewEmbedder()
	vec := e.Vectorize("")

	if len(vec) != 0 {
		t.Fatalf("expected empty vector, got %d entries", len(vec))
	}
}

func TestVectorize_NonEmpty(t *testing.T) {
	e := NewEmbedder()
	vec := e.Vectorize("hello world")

	if len(vec) == 0 {
		t.Fatal("expected non-empty vector for 'hello world'")
	}
	if _, ok := vec["hello"]; !ok {
		t.Error("expected 'hello' in vector")
	}
	if _, ok := vec["world"]; !ok {
		t.Error("expected 'world' in vector")
	}
}

func TestVectorize_StopWordsExcluded(t *testing.T) {
	e := NewEmbedder()
	vec := e.Vectorize("the quick brown fox")

	stopWords := []string{"the", "a", "is", "of", "in", "to"}
	for _, sw := range stopWords {
		if freq, ok := vec[sw]; ok {
			t.Errorf("stop word %q should not appear in vector, got freq=%v", sw, freq)
		}
	}

	// Content words should be present
	for _, word := range []string{"quick", "brown", "fox"} {
		if _, ok := vec[word]; !ok {
			t.Errorf("content word %q should appear in vector", word)
		}
	}
}

func TestVectorize_Normalization(t *testing.T) {
	e := NewEmbedder()

	tests := []struct {
		name     string
		text     string
		wantKeys []string
	}{
		{
			name:     "single token",
			text:     "hello",
			wantKeys: []string{"hello"},
		},
		{
			name:     "two unique tokens",
			text:     "hello world",
			wantKeys: []string{"hello", "world"},
		},
		{
			name:     "repeated token",
			text:     "hello hello hello",
			wantKeys: []string{"hello"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			vec := e.Vectorize(tt.text)

			if len(vec) != len(tt.wantKeys) {
				t.Fatalf("expected %d keys, got %d", len(tt.wantKeys), len(vec))
			}
			for _, k := range tt.wantKeys {
				if _, ok := vec[k]; !ok {
					t.Errorf("expected key %q in vector", k)
				}
			}

			// Sum of frequencies should be 1.0 (normalized)
			var sum float64
			for _, v := range vec {
				sum += v
			}
			if math.Abs(sum-1.0) > 1e-9 {
				t.Errorf("expected normalized sum 1.0, got %v", sum)
			}
		})
	}
}

func TestVectorize_RepeatedTokens(t *testing.T) {
	e := NewEmbedder()
	// "hello" appears 3 times, "world" appears 1 time
	vec := e.Vectorize("hello hello hello world")

	if len(vec) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(vec))
	}
	if math.Abs(vec["hello"]-0.75) > 1e-9 {
		t.Errorf("expected 'hello' freq 0.75, got %v", vec["hello"])
	}
	if math.Abs(vec["world"]-0.25) > 1e-9 {
		t.Errorf("expected 'world' freq 0.25, got %v", vec["world"])
	}
}

// ---------------------------------------------------------------------------
// CosineSimilarity tests
// ---------------------------------------------------------------------------

func TestCosineSimilarity_IdenticalTexts(t *testing.T) {
	e := NewEmbedder()
	v1 := e.Vectorize("the quick brown fox jumps over the lazy dog")
	v2 := e.Vectorize("the quick brown fox jumps over the lazy dog")

	sim := CosineSimilarity(v1, v2)

	if math.Abs(sim-1.0) > 1e-9 {
		t.Errorf("expected ~1.0 for identical texts, got %v", sim)
	}
}

func TestCosineSimilarity_DifferentTexts(t *testing.T) {
	e := NewEmbedder()

	tests := []struct {
		name    string
		textA   string
		textB   string
		maxSim  float64
	}{
		{
			name:    "completely unrelated",
			textA:   "machine learning algorithms",
			textB:   "basketball game today",
			maxSim:  0.5,
		},
		{
			name:    "somewhat related",
			textA:   "python programming language",
			textB:   "java programming language",
			maxSim:  0.99, // may be high due to shared terms
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			v1 := e.Vectorize(tt.textA)
			v2 := e.Vectorize(tt.textB)
			sim := CosineSimilarity(v1, v2)

			if sim > tt.maxSim {
				t.Errorf("expected similarity <= %v for %q vs %q, got %v",
					tt.maxSim, tt.textA, tt.textB, sim)
			}
		})
	}
}

func TestCosineSimilarity_CompletelyDifferent_LowSimilarity(t *testing.T) {
	e := NewEmbedder()
	// Ensure truly no overlap in content words
	v1 := e.Vectorize("artificial intelligence neural networks")
	v2 := e.Vectorize("banana apple orange fruit")

	sim := CosineSimilarity(v1, v2)

	if sim >= 0.5 {
		t.Errorf("expected similarity < 0.5 for unrelated texts, got %v", sim)
	}
	if sim != 0 {
		t.Errorf("expected 0 similarity for texts with no token overlap, got %v", sim)
	}
}

func TestCosineSimilarity_EmptyVectors(t *testing.T) {
	tests := []struct {
		name string
		a    map[string]float64
		b    map[string]float64
	}{
		{"both empty", map[string]float64{}, map[string]float64{}},
		{"first empty", map[string]float64{}, map[string]float64{"hello": 1.0}},
		{"second empty", map[string]float64{"hello": 1.0}, map[string]float64{}},
		{"both nil", nil, nil},
		{"first nil", nil, map[string]float64{"hello": 1.0}},
		{"second nil", map[string]float64{"hello": 1.0}, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sim := CosineSimilarity(tt.a, tt.b)
			if sim != 0 {
				t.Errorf("expected 0 for empty/nil vector(s), got %v", sim)
			}
		})
	}
}

func TestCosineSimilarity_SelfSimilarity(t *testing.T) {
	e := NewEmbedder()
	texts := []string{
		"hello world",
		"machine learning is fun",
		"distributed systems design patterns",
	}

	for _, text := range texts {
		t.Run(text, func(t *testing.T) {
			v := e.Vectorize(text)
			sim := CosineSimilarity(v, v)

			if math.Abs(sim-1.0) > 1e-9 {
				t.Errorf("expected self-similarity 1.0 for %q, got %v", text, sim)
			}
		})
	}
}

func TestCosineSimilarity_PartialOverlap(t *testing.T) {
	e := NewEmbedder()
	// "learning" overlaps, "football" / "machine" / "python" do not
	v1 := e.Vectorize("machine learning python")
	v2 := e.Vectorize("deep learning football")

	sim := CosineSimilarity(v1, v2)

	// Should be > 0 due to "learning" but < 1.0
	if sim <= 0 {
		t.Errorf("expected similarity > 0 due to overlapping token 'learning', got %v", sim)
	}
	if sim >= 1.0 {
		t.Errorf("expected similarity < 1.0, got %v", sim)
	}
}

func TestCosineSimilarity_OrthogonalVectors(t *testing.T) {
	// Two vectors with no shared keys
	a := map[string]float64{"apple": 0.5, "banana": 0.5}
	b := map[string]float64{"car": 0.6, "dog": 0.4}

	sim := CosineSimilarity(a, b)

	if sim != 0 {
		t.Errorf("expected 0 for orthogonal vectors, got %v", sim)
	}
}

// ---------------------------------------------------------------------------
// Integration: Vectorize + CosineSimilarity round-trip
// ---------------------------------------------------------------------------

func TestEmbedder_VectorizeAndSimilarity(t *testing.T) {
	e := NewEmbedder()

	tests := []struct {
		name     string
		query    string
		doc      string
		minSim   float64
		maxSim   float64
	}{
		{
			name:   "exact match minus stop words",
			query:  "what is machine learning",
			doc:    "machine learning is a subset of artificial intelligence",
			minSim: 0.3,
			maxSim: 1.0,
		},
		{
			name:   "semantically related but different words",
			query:  "how to train a neural network",
			doc:    "deep learning uses neural networks for training models",
			minSim: 0.1,
			maxSim: 1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			qv := e.Vectorize(tt.query)
			dv := e.Vectorize(tt.doc)
			sim := CosineSimilarity(qv, dv)

			if sim < tt.minSim {
				t.Errorf("similarity %v below min %v", sim, tt.minSim)
			}
			if sim > tt.maxSim {
				t.Errorf("similarity %v above max %v", sim, tt.maxSim)
			}
		})
	}
}
