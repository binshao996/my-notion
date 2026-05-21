package ai

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Client is the DeepSeek API client (OpenAI-compatible).
type Client struct {
	config     *Config
	httpClient *http.Client
}

func NewClient(config *Config) *Client {
	if config == nil || config.APIKey == "" {
		return nil
	}
	return &Client{
		config: config,
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (c *Client) IsAvailable() bool {
	return c != nil && c.config.APIKey != ""
}

// ChatCompletion sends a chat completion request to DeepSeek.
func (c *Client) ChatCompletion(req *ChatRequest) (*ChatResponse, error) {
	if !c.IsAvailable() {
		return nil, fmt.Errorf("ai client not available: DEEPSEEK_API_KEY not set")
	}
	if req.Model == "" {
		req.Model = c.config.Model
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.config.BaseURL + "/v1/chat/completions"
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("deepseek api error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var chatResp ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("decode response: %w", err)
	}
	return &chatResp, nil
}

// ChatCompletionStream sends a streaming request and calls onDelta for each chunk.
// Returns the full accumulated response.
func (c *Client) ChatCompletionStream(req *ChatRequest, onDelta func(delta string) error) (*ChatResponse, error) {
	if !c.IsAvailable() {
		return nil, fmt.Errorf("ai client not available")
	}

	req.Stream = true
	if req.Model == "" {
		req.Model = c.config.Model
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := c.config.BaseURL + "/v1/chat/completions"
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	var fullContent string
	decoder := json.NewDecoder(resp.Body)
	for {
		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
			} `json:"choices"`
		}
		if err := decoder.Decode(&chunk); err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("decode stream chunk: %w", err)
		}
		if len(chunk.Choices) > 0 && chunk.Choices[0].Delta.Content != "" {
			fullContent += chunk.Choices[0].Delta.Content
			if onDelta != nil {
				if err := onDelta(chunk.Choices[0].Delta.Content); err != nil {
					return nil, err
				}
			}
		}
	}

	return &ChatResponse{
		Choices: []ChatChoice{{Message: ChatMessage{Role: "assistant", Content: fullContent}}},
	}, nil
}

// EstimateCost returns an estimated cost in USD for the given token usage.
// DeepSeek pricing: deepseek-chat ~$0.14/1M input, $0.28/1M output.
func EstimateCost(model string, promptTokens, completionTokens int) float64 {
	inputCost := float64(promptTokens) / 1_000_000 * 0.14
	outputCost := float64(completionTokens) / 1_000_000 * 0.28
	return inputCost + outputCost
}

// CreateEmbedding calls the OpenAI-compatible embeddings endpoint (POST /v1/embeddings).
// Returns a float32 vector of length config.EmbeddingDim.
func (c *Client) CreateEmbedding(req *EmbeddingRequest) (*EmbeddingResponse, error) {
	if !c.IsAvailable() {
		return nil, fmt.Errorf("ai client not available: DEEPSEEK_API_KEY not set")
	}
	if req.Model == "" {
		req.Model = c.config.Model
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal embedding request: %w", err)
	}

	url := c.config.BaseURL + "/v1/embeddings"
	httpReq, err := http.NewRequest("POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create embedding request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+c.config.APIKey)

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("embedding http request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("embedding api error %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var embResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embResp); err != nil {
		return nil, fmt.Errorf("decode embedding response: %w", err)
	}
	return &embResp, nil
}
