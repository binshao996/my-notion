package ai

import "os"

// Config holds all AI-related configuration (DeepSeek, Milvus, embeddings).
type Config struct {
	APIKey  string
	BaseURL string
	Model   string

	// Milvus vector database
	MilvusAddr string // e.g. "localhost:19530"

	// Embedding settings
	EmbeddingDim int // default 1536 (DeepSeek embedding dimension)
}

// LoadConfig reads configuration from environment variables with sensible defaults.
func LoadConfig() *Config {
	baseURL := os.Getenv("DEEPSEEK_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}
	model := os.Getenv("DEEPSEEK_MODEL")
	if model == "" {
		model = "deepseek-v4-flash"
	}

	milvusAddr := os.Getenv("MILVUS_ADDR")
	if milvusAddr == "" {
		milvusAddr = "localhost:19530"
	}

	embeddingDim := 1536
	if dim := os.Getenv("EMBEDDING_DIM"); dim != "" {
		if d, err := parseInt(dim); err == nil && d > 0 {
			embeddingDim = d
		}
	}

	return &Config{
		APIKey:       os.Getenv("DEEPSEEK_API_KEY"),
		BaseURL:      baseURL,
		Model:        model,
		MilvusAddr:   milvusAddr,
		EmbeddingDim: embeddingDim,
	}
}

func parseInt(s string) (int, error) {
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, os.ErrInvalid
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
