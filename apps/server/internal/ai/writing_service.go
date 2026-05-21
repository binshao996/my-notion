package ai

import (
	"encoding/json"
	"fmt"
	"strings"
)

const writingSystemPrompt = `You are a writing assistant integrated into a Notion-like document editor.
You must respond with ONLY a valid JSON array of blocks. Each block has this structure:
{"type": "<type>", "content": "<text content>"}

Supported block types:
- paragraph: regular text
- heading1: main heading
- heading2: sub heading
- heading3: sub-sub heading
- bulleted_list_item: bullet point
- numbered_list_item: numbered item
- todo: task item (prefix with ☐)
- quote: blockquote text
- callout: callout box with icon prefix
- code: code block (language optional in content)
- divider: horizontal rule (content can be empty)

Rules:
1. Output ONLY the JSON array, no other text, no markdown fences, no explanations.
2. Every block must have "type" and "content" fields.
3. Use appropriate block types — not everything is a paragraph.
4. For lists, use bulleted_list_item or numbered_list_item.
5. Keep code in code blocks.
6. Use headings for section titles.

Example output:
[{"type":"heading1","content":"Introduction"},{"type":"paragraph","content":"This is the introduction paragraph."},{"type":"bulleted_list_item","content":"Key point one"},{"type":"bulleted_list_item","content":"Key point two"}]`

// WritingService handles AI writing operations (generate, rewrite, summarize, etc.).
type WritingService struct {
	gateway *Gateway
}

// NewWritingService creates a new WritingService.
func NewWritingService(gateway *Gateway) *WritingService {
	return &WritingService{gateway: gateway}
}

// Execute processes a WritingRequest and returns generated blocks.
func (s *WritingService) Execute(userID uint, req *WritingRequest) (*WritingResponse, error) {
	// Sanitize inputs
	if _, err := s.gateway.SanitizePrompt(req.Prompt); err != nil {
		return nil, fmt.Errorf("prompt rejected: %w", err)
	}
	if _, err := s.gateway.SanitizePrompt(req.Context); err != nil {
		return nil, fmt.Errorf("context rejected: %w", err)
	}

	// Rate limit
	if err := s.gateway.CheckLimit(userID); err != nil {
		return nil, err
	}

	// Build user prompt based on action
	userPrompt := buildUserPrompt(req)
	if userPrompt == "" {
		return nil, fmt.Errorf("action %q requires prompt or context", req.Action)
	}

	// Set temperature and max tokens
	temperature := 0.7
	if req.Action == "summarize" || req.Action == "translate" || req.Action == "proofread" ||
		req.Action == "extract_actions" || req.Action == "extract_decisions" || req.Action == "extract_entities" ||
		req.Action == "classify" || req.Action == "extract_structured" {
		temperature = 0.3
	}

	chatReq := &ChatRequest{
		Model: ModelForTask(req.Action),
		Messages: []ChatMessage{
			{Role: "system", Content: writingSystemPrompt},
			{Role: "user", Content: userPrompt},
		},
		MaxTokens:   2048,
		Temperature: temperature,
	}

	// Call AI
	chatResp, err := s.gateway.client.ChatCompletion(chatReq)
	if err != nil {
		return nil, fmt.Errorf("ai service error: %w", err)
	}

	if len(chatResp.Choices) == 0 {
		return nil, fmt.Errorf("ai returned no choices")
	}

	content := chatResp.Choices[0].Message.Content

	// Parse JSON blocks from the response
	blocks, err := parseBlocks(content)
	if err != nil {
		// Fallback: wrap raw content in a single paragraph block
		blocks = []AIGeneratedBlock{
			{Type: "paragraph", Content: content},
		}
	}

	// Log cost
	s.gateway.LogCost(
		userID,
		chatReq.Model,
		chatResp.Usage.PromptTokens,
		chatResp.Usage.CompletionTokens,
		"writing",
	)

	return &WritingResponse{
		Blocks: blocks,
		Usage:  chatResp.Usage,
	}, nil
}

// buildUserPrompt constructs the user message based on the writing action.
func buildUserPrompt(req *WritingRequest) string {
	switch req.Action {
	case "generate":
		prompt := req.Prompt
		context := req.Context
		if context != "" {
			return fmt.Sprintf("Write the following content: %s\n\nContext: %s", prompt, context)
		}
		return fmt.Sprintf("Write the following content: %s", prompt)

	case "rewrite":
		tone := req.Tone
		if tone == "" {
			tone = "professional"
		}
		context := req.Context
		prompt := req.Prompt
		if prompt != "" {
			return fmt.Sprintf("Rewrite the following text to be more %s. The original text:\n%s\n\nAdditional instruction: %s", tone, context, prompt)
		}
		return fmt.Sprintf("Rewrite the following text to be more %s. The original text:\n%s", tone, context)

	case "summarize":
		return fmt.Sprintf("Summarize the following text into 3-7 key bullet points:\n\n%s", req.Context)

	case "expand":
		context := req.Context
		prompt := req.Prompt
		if prompt != "" {
			return fmt.Sprintf("Expand the following text with more detail and examples:\n\n%s\n\nAdditional instruction: %s", context, prompt)
		}
		return fmt.Sprintf("Expand the following text with more detail and examples:\n\n%s", context)

	case "translate":
		lang := req.Lang
		if lang == "" {
			lang = "English"
		}
		return fmt.Sprintf("Translate the following text to %s:\n\n%s", lang, req.Context)

	case "proofread":
		return fmt.Sprintf("Proofread and fix grammar/spelling/punctuation in the following text. Return the corrected version. Do NOT change the meaning:\n\n%s", req.Context)

	case "extract_actions":
		return fmt.Sprintf("Extract all action items, todos, and tasks from the following text. For each action item, identify: the task, the owner (if mentioned), and the deadline (if mentioned). Format as structured blocks.\n\n%s", req.Context)

	case "extract_decisions":
		return fmt.Sprintf("Extract all decisions, conclusions, and key takeaways from the following text. For each, note the decision, rationale, and any trade-offs mentioned. Format as structured blocks.\n\n%s", req.Context)

	case "extract_entities":
		return fmt.Sprintf("Identify all entities mentioned in the following text that could be linked in a knowledge base. Include: people names, project names, document titles, dates, and key terms. Format as bulleted list items.\n\n%s", req.Context)

	case "classify":
		return fmt.Sprintf("Analyze the following text and suggest appropriate tags, categories, and labels. Consider: topic, content type, sentiment, urgency level, and any relevant domains. Output as bulleted list items with the category and suggested tag.\n\n%s", req.Context)

	case "extract_structured":
		return fmt.Sprintf("Extract structured information from the following text. Organize it into: Background/Context, Goals/Objectives, Key Requirements, Risks/Concerns, and Open Questions. Use heading2 for each section and paragraph for content. If a section has no information, skip it.\n\n%s", req.Context)

	default:
		// Treat unknown action as generate
		context := req.Context
		prompt := req.Prompt
		if context != "" {
			return fmt.Sprintf("Write the following content: %s\n\nContext: %s", prompt, context)
		}
		return fmt.Sprintf("Write the following content: %s", prompt)
	}
}

// parseBlocks extracts a JSON array of AIGeneratedBlock from an AI response string.
// Handles responses that may be wrapped in markdown code fences.
func parseBlocks(content string) ([]AIGeneratedBlock, error) {
	content = strings.TrimSpace(content)

	// Strip markdown code fences if present
	if strings.HasPrefix(content, "```") {
		// Find the first newline after opening fence
		idx := strings.Index(content, "\n")
		if idx != -1 {
			content = content[idx+1:]
		}
		// Remove closing fence
		if idx := strings.LastIndex(content, "```"); idx != -1 {
			content = content[:idx]
		}
		content = strings.TrimSpace(content)
	}

	var blocks []AIGeneratedBlock
	if err := json.Unmarshal([]byte(content), &blocks); err != nil {
		return nil, fmt.Errorf("failed to parse AI response as JSON blocks: %w", err)
	}

	// Validate that every block has a type
	for i := range blocks {
		if blocks[i].Type == "" {
			blocks[i].Type = "paragraph"
		}
	}

	return blocks, nil
}
