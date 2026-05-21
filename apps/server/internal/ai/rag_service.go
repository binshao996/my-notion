package ai

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"github.com/bin-ke/my-notion/internal/permission"
	"github.com/bin-ke/my-notion/internal/search"
	"gorm.io/gorm"
)

// RAGService is the Retrieval-Augmented Generation engine.
// It retrieves relevant content blocks, then generates an answer with citations.
type RAGService struct {
	client    *Client
	embedder  *Embedder
	db        *gorm.DB
	searchSvc *search.Service
	permSvc   *permission.Service
}

// SearchResult represents a retrieved content chunk with its relevance score.
type SearchResult struct {
	PageID    uint
	BlockID   uint
	PageTitle string
	Text      string
	Score     float64
}

// NewRAGService creates a new RAGService.
func NewRAGService(client *Client, db *gorm.DB, searchSvc *search.Service, permSvc *permission.Service) *RAGService {
	return &RAGService{
		client:    client,
		embedder:  NewEmbedder(),
		db:        db,
		searchSvc: searchSvc,
		permSvc:   permSvc,
	}
}

// Retrieve finds relevant content across all accessible pages/blocks.
// Steps:
//  1. Query OpenSearch for keyword matches (if searchSvc available)
//  2. Fetch blocks with text content from DB (limit 500 recent, joined with pages)
//  3. Tokenize query and each block's text, compute TF-IDF cosine similarity
//  4. Merge keyword + TF-IDF scores (weighted: 0.6 OpenSearch + 0.4 TF-IDF)
//  5. Deduplicate by block ID
//  6. Filter by permission — only blocks from pages the user can access
//  7. Sort by merged score descending, take top 10
func (s *RAGService) Retrieve(userID uint, workspaceID uint, query string) ([]SearchResult, error) {
	// 1. Query OpenSearch for keyword matches (if available)
	type keywordHit struct {
		PageID    uint
		Text      string
		Score     float64
	}
	keywordResults := make(map[uint]*keywordHit) // blockID -> hit
	if s.searchSvc != nil {
		searchResp, err := s.searchSvc.Search(workspaceID, query)
		if err == nil {
			for _, block := range searchResp.Blocks {
				// Use 1.0 as default relevance score; could be enhanced to
				// decode OpenSearch _score from the raw response.
				keywordResults[block.ID] = &keywordHit{
					PageID: block.PageID,
					Text:   block.Text,
					Score:  1.0,
				}
			}
		}
	}

	// 2. Get blocks with text content from DB
	type blockRow struct {
		ID        uint
		PageID    uint
		Props     string
		PageTitle string
	}
	var rows []blockRow
	q := s.db.Table("blocks").
		Select("blocks.id, blocks.page_id, blocks.props, pages.title AS page_title").
		Joins("JOIN pages ON pages.id = blocks.page_id")
	if workspaceID > 0 {
		q = q.Where("pages.workspace_id = ?", workspaceID)
	}
	if err := q.Order("blocks.updated_at DESC").Limit(500).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("query blocks: %w", err)
	}

	// 3. Tokenize query and compute TF-IDF cosine similarity for each block
	queryVec := s.embedder.Vectorize(query)

	type candidate struct {
		PageID    uint
		BlockID   uint
		PageTitle string
		Text      string
		Score     float64
	}
	var candidates []candidate

	for _, row := range rows {
		text := search.ExtractText(row.Props)
		if text == "" {
			continue
		}
		blockVec := s.embedder.Vectorize(text)
		tfidfScore := CosineSimilarity(queryVec, blockVec)
		if tfidfScore > 0 {
			candidates = append(candidates, candidate{
				PageID:    row.PageID,
				BlockID:   row.ID,
				PageTitle: row.PageTitle,
				Text:      text,
				Score:     tfidfScore,
			})
		}
	}

	// 4. Merge keyword and TF-IDF results with weighted scoring.
	// combinedScore = 0.6 * keywordScore + 0.4 * tfidfScore
	// Deduplicate by block ID, keeping the merged score.
	type merged struct {
		PageID    uint
		BlockID   uint
		PageTitle string
		Text      string
		Score     float64
	}
	seen := make(map[uint]*merged)

	// Add keyword-matched blocks first (they may get additional TF-IDF weight)
	for blockID, kh := range keywordResults {
		seen[blockID] = &merged{
			BlockID: blockID,
			PageID:  kh.PageID,
			Text:    kh.Text,
			Score:   0.6 * kh.Score,
		}
	}

	// Merge in TF-IDF candidates
	for _, c := range candidates {
		if existing, ok := seen[c.BlockID]; ok {
			// Block exists from keyword search: add TF-IDF contribution
			existing.Score += 0.4 * c.Score
			existing.PageID = c.PageID
			existing.PageTitle = c.PageTitle
			existing.Text = c.Text
		} else {
			seen[c.BlockID] = &merged{
				PageID:    c.PageID,
				BlockID:   c.BlockID,
				PageTitle: c.PageTitle,
				Text:      c.Text,
				Score:     0.4 * c.Score,
			}
		}
	}

	// Collect merged candidates from the seen map
	mergedCandidates := make([]merged, 0, len(seen))
	for _, m := range seen {
		mergedCandidates = append(mergedCandidates, *m)
	}

	// 5. Filter by permission
	var results []SearchResult
	for _, m := range mergedCandidates {
		if s.permSvc.CheckAccess(userID, m.PageID) != "" {
			results = append(results, SearchResult{
				PageID:    m.PageID,
				BlockID:   m.BlockID,
				PageTitle: m.PageTitle,
				Text:      m.Text,
				Score:     m.Score,
			})
		}
	}

	// 6. Sort by merged score descending, take top 10
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > 10 {
		results = results[:10]
	}

	return results, nil
}

// Ask performs the full RAG pipeline: retrieve -> build prompt -> generate answer -> return with citations.
func (s *RAGService) Ask(userID uint, req *QARequest) (*QAResponse, error) {
	if s.client == nil || !s.client.IsAvailable() {
		return nil, fmt.Errorf("ai client not available")
	}

	// 1. Retrieve relevant chunks
	results, err := s.Retrieve(userID, req.WorkspaceID, req.Question)
	if err != nil {
		return nil, fmt.Errorf("retrieve: %w", err)
	}

	// 2. Build system prompt with context
	systemPrompt := "You are a helpful assistant that answers questions based on the provided context. " +
		"Always cite your sources using [N] notation where N is the reference number. " +
		"If the context does not contain enough information, say so clearly."

	if len(results) > 0 {
		var contextParts []string
		for i, r := range results {
			contextParts = append(contextParts,
				fmt.Sprintf("Reference [%d]: Page '%s': %s", i+1, r.PageTitle, r.Text))
		}
		systemPrompt += "\n\nRelevant context:\n" + strings.Join(contextParts, "\n")
	}

	// 3. Generate answer via LLM
	chatReq := &ChatRequest{
		Model: ModelForTask("rag"),
		Messages: []ChatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: req.Question},
		},
		MaxTokens:   1024,
		Temperature: 0.3,
	}

	chatResp, err := s.client.ChatCompletion(chatReq)
	if err != nil {
		return nil, fmt.Errorf("chat completion: %w", err)
	}

	answer := ""
	if len(chatResp.Choices) > 0 {
		answer = chatResp.Choices[0].Message.Content
	}

	// 4. Parse citations: find [N] references and map back to page/block IDs
	citations := s.parseCitations(answer, results)

	return &QAResponse{
		Answer:    answer,
		Citations: citations,
		Usage:     chatResp.Usage,
	}, nil
}

// parseCitations extracts [N] references from the answer text and maps them to
// SearchResults, returning deduplicated Citation objects.
func (s *RAGService) parseCitations(answer string, results []SearchResult) []Citation {
	re := regexp.MustCompile(`\[(\d+)\]`)
	matches := re.FindAllStringSubmatch(answer, -1)
	if len(matches) == 0 {
		return nil
	}

	seen := make(map[uint]bool)
	var citations []Citation
	for _, m := range matches {
		var num int
		fmt.Sscanf(m[1], "%d", &num)
		idx := num - 1
		if idx < 0 || idx >= len(results) {
			continue
		}
		r := results[idx]
		if seen[r.PageID] {
			continue
		}
		seen[r.PageID] = true

		snippet := r.Text
		if len(snippet) > 200 {
			snippet = snippet[:200] + "..."
		}
		citations = append(citations, Citation{
			PageID:  r.PageID,
			BlockID: r.BlockID,
			Title:   r.PageTitle,
			Snippet: snippet,
		})
	}
	return citations
}
