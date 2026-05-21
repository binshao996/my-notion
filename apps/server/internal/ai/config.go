package ai

import "os"

type Config struct {
	APIKey  string
	BaseURL string
	Model   string
}

func LoadConfig() *Config {
	baseURL := os.Getenv("DEEPSEEK_BASE_URL")
	if baseURL == "" {
		baseURL = "https://api.deepseek.com"
	}
	model := os.Getenv("DEEPSEEK_MODEL")
	if model == "" {
		model = "deepseek-v4-flash"
	}
	return &Config{
		APIKey:  os.Getenv("DEEPSEEK_API_KEY"),
		BaseURL: baseURL,
		Model:   model,
	}
}
