package ai

import (
	"math"
	"strings"
	"unicode"
)

// Embedder is a simple keyword-based text vectorizer for MVP.
// Uses TF-like scoring — no external embedding API needed.
type Embedder struct {
	stopWords map[string]bool
}

// NewEmbedder creates an Embedder initialized with common English stop words.
func NewEmbedder() *Embedder {
	stopWords := map[string]bool{
		"a": true, "an": true, "the": true, "is": true, "are": true, "was": true,
		"were": true, "be": true, "been": true, "being": true, "have": true,
		"has": true, "had": true, "do": true, "does": true, "did": true,
		"will": true, "would": true, "shall": true, "should": true, "can": true,
		"could": true, "may": true, "might": true, "must": true, "of": true,
		"in": true, "to": true, "for": true, "with": true, "on": true,
		"at": true, "from": true, "by": true, "about": true, "as": true,
		"into": true, "through": true, "during": true, "before": true, "after": true,
		"above": true, "below": true, "between": true, "and": true, "but": true,
		"or": true, "nor": true, "not": true, "so": true, "yet": true,
		"both": true, "either": true, "neither": true, "each": true, "every": true,
		"all": true, "any": true, "few": true, "more": true, "most": true,
		"other": true, "some": true, "such": true, "no": true, "only": true,
		"own": true, "same": true, "this": true, "that": true, "these": true,
		"those": true, "it": true, "its": true, "he": true, "she": true,
		"they": true, "them": true, "we": true, "us": true, "i": true,
		"me": true, "my": true, "you": true, "your": true,
	}
	return &Embedder{stopWords: stopWords}
}

// Tokenize lowercases the text, removes punctuation/whitespace, drops stop
// words and single-character tokens, and returns the remaining tokens.
func (e *Embedder) Tokenize(text string) []string {
	text = strings.ToLower(text)
	var tokens []string
	var current strings.Builder
	for _, r := range text {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			current.WriteRune(r)
		} else {
			if current.Len() > 0 {
				token := current.String()
				if !e.stopWords[token] && len(token) > 1 {
					tokens = append(tokens, token)
				}
				current.Reset()
			}
		}
	}
	if current.Len() > 0 {
		token := current.String()
		if !e.stopWords[token] && len(token) > 1 {
			tokens = append(tokens, token)
		}
	}
	return tokens
}

// Vectorize converts text to a sparse TF vector (token -> term frequency).
func (e *Embedder) Vectorize(text string) map[string]float64 {
	tokens := e.Tokenize(text)
	vec := make(map[string]float64, len(tokens))
	if len(tokens) == 0 {
		return vec
	}
	for _, token := range tokens {
		vec[token] += 1.0
	}
	// Normalize by total token count (term frequency)
	total := float64(len(tokens))
	for token := range vec {
		vec[token] /= total
	}
	return vec
}

// CosineSimilarity returns the cosine similarity between two sparse vectors.
func CosineSimilarity(a, b map[string]float64) float64 {
	if len(a) == 0 || len(b) == 0 {
		return 0
	}
	var dotProduct float64
	for k, v := range a {
		if w, ok := b[k]; ok {
			dotProduct += v * w
		}
	}
	var normA, normB float64
	for _, v := range a {
		normA += v * v
	}
	for _, v := range b {
		normB += v * v
	}
	if normA == 0 || normB == 0 {
		return 0
	}
	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
