package ai

import "time"

// ---------- OpenAI-compatible chat types ----------

type ChatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ChatRequest struct {
	Model       string        `json:"model"`
	Messages    []ChatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens,omitempty"`
	Temperature float64       `json:"temperature,omitempty"`
	Stream      bool          `json:"stream,omitempty"`
}

type ChatChoice struct {
	Index   int         `json:"index"`
	Message ChatMessage `json:"message"`
	Delta   ChatMessage `json:"delta,omitempty"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type ChatResponse struct {
	ID      string       `json:"id"`
	Choices []ChatChoice `json:"choices"`
	Usage   Usage        `json:"usage,omitempty"`
}

// ---------- AI writing ----------

type WritingRequest struct {
	Action  string `json:"action"`  // generate, rewrite, summarize, expand, translate, proofread
	Context string `json:"context"` // selected text or page content
	Prompt  string `json:"prompt"`  // user instruction (for generate/rewrite)
	Tone    string `json:"tone,omitempty"`
	Lang    string `json:"lang,omitempty"` // target language for translate
}

type WritingResponse struct {
	Blocks []AIGeneratedBlock `json:"blocks"`
	Usage  Usage              `json:"usage"`
}

type AIGeneratedBlock struct {
	Type    string `json:"type"`    // paragraph, heading1, heading2, heading3, bulleted_list_item, etc.
	Content string `json:"content"` // text content
}

// ---------- RAG / Q&A ----------

type QARequest struct {
	Question    string `json:"question"`
	WorkspaceID uint   `json:"workspace_id"`
}

type QAResponse struct {
	Answer    string      `json:"answer"`
	Citations []Citation  `json:"citations"`
	Usage     Usage       `json:"usage"`
}

type Citation struct {
	PageID    uint   `json:"page_id"`
	BlockID   uint   `json:"block_id,omitempty"`
	Title     string `json:"title"`
	Snippet   string `json:"snippet"`
}

// ---------- Database autofill ----------

type AutofillRequest struct {
	DatabaseID   uint   `json:"database_id"`
	PropertyID   uint   `json:"property_id"`   // target property to fill
	SourcePropID uint   `json:"source_prop_id"` // input property (usually title or text)
	RecordIDs    []uint `json:"record_ids,omitempty"` // empty = all records
	Instruction  string `json:"instruction,omitempty"` // e.g. "summarize in one sentence"
}

type AutofillJob struct {
	ID          string    `json:"id"`
	DatabaseID  uint      `json:"database_id"`
	PropertyID  uint      `json:"property_id"`
	Total       int       `json:"total"`
	Completed   int       `json:"completed"`
	Failed      int       `json:"failed"`
	Status      string    `json:"status"` // pending, running, completed, failed
	CreatedAt   time.Time `json:"created_at"`
}

// ---------- Cost tracking ----------

type CostLog struct {
	ID        uint      `json:"id"`
	UserID    uint      `json:"user_id"`
	Model     string    `json:"model"`
	Tokens    int       `json:"tokens"`
	Cost      float64   `json:"cost"`
	Feature   string    `json:"feature"` // writing, rag, autofill
	CreatedAt time.Time `json:"created_at"`
}
