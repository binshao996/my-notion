package ai

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/bin-ke/my-notion/internal/permission"
	"github.com/bin-ke/my-notion/internal/search"
	"gorm.io/gorm"
)

// RAGService is the Retrieval-Augmented Generation engine.
// It retrieves relevant content blocks, then generates an answer with citations.
type RAGService struct {
	client      *Client
	embedder    Embedder           // primary embedder (DeepSeek if available)
	keywordEmb  *KeywordVectorizer // always-available TF-IDF fallback
	db          *gorm.DB
	searchSvc   *search.Service
	permSvc     *permission.Service
	vectorStore *VectorStore // Milvus store (nil if unavailable)
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
		client:     client,
		keywordEmb: NewKeywordVectorizer(),
		db:         db,
		searchSvc:  searchSvc,
		permSvc:    permSvc,
	}
}

// SetVectorStore configures the Milvus vector store for hybrid search.
func (s *RAGService) SetVectorStore(vs *VectorStore, emb Embedder) {
	s.vectorStore = vs
	s.embedder = emb
}

// Retrieve finds relevant content across all accessible pages/blocks.
// Steps:
//  1. If Milvus is available: generate dense query embedding, search Milvus
//  2. Query OpenSearch for keyword matches (if searchSvc available)
//  3. If Milvus unavailable: fetch blocks from DB, compute TF-IDF cosine similarity
//  4. Merge keyword + semantic/TF-IDF scores (weighted: 0.6 keyword + 0.4 vector)
//  5. Deduplicate by block ID
//  6. Filter by permission — only blocks from pages the user can access
//  7. Sort by merged score descending, take top 10
func (s *RAGService) Retrieve(userID uint, workspaceID uint, query string) ([]SearchResult, error) {
	type merged struct {
		PageID    uint
		BlockID   uint
		PageTitle string
		Text      string
		Score     float64
	}

	seen := make(map[uint]*merged)

	// 1. Semantic search via Milvus (if available)
	if s.vectorStore != nil && s.embedder != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		queryVec, err := s.embedder.Embed(ctx, query)
		if err != nil {
			log.Printf("rag: embedding failed, falling back to keyword: %v", err)
		} else {
			filter := map[string]any{"workspace_id": workspaceID}
			hits, err := s.vectorStore.Search(ctx, queryVec, 20, filter)
			if err != nil {
				log.Printf("rag: milvus search failed: %v", err)
			} else {
				// L2 distance: lower is better. Convert to similarity score (1 / (1 + distance)).
				for _, hit := range hits {
					pageID := uint(hit.PageID)
					blockID := s.parseBlockID(hit.ChunkID)
					if blockID == 0 {
						continue
					}
					// Normalize L2 distance to a [0,1] similarity score.
					score := 0.4 * (1.0 / (1.0 + float64(hit.Score)))
					seen[blockID] = &merged{
						PageID:  pageID,
						BlockID: blockID,
						Score:   score,
					}
				}
			}
		}
	}

	// 2. Keyword search via OpenSearch (if available)
	if s.searchSvc != nil {
		searchResp, err := s.searchSvc.Search(workspaceID, query)
		if err == nil {
			for _, block := range searchResp.Blocks {
				if existing, ok := seen[block.ID]; ok {
					existing.Score += 0.6 * 1.0
					existing.PageID = block.PageID
					existing.Text = block.Text
				} else {
					seen[block.ID] = &merged{
						PageID:  block.PageID,
						BlockID: block.ID,
						Text:    block.Text,
						Score:   0.6 * 1.0,
					}
				}
			}
		}
	}

	// 3. Fallback: TF-IDF from DB blocks (only if no Milvus results to augment)
	if s.vectorStore == nil {
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

		queryVec := s.keywordEmb.Vectorize(query)

		for _, row := range rows {
			text := search.ExtractText(row.Props)
			if text == "" {
				continue
			}
			blockVec := s.keywordEmb.Vectorize(text)
			tfidfScore := CosineSimilarity(queryVec, blockVec)
			if tfidfScore > 0 {
				if existing, ok := seen[row.ID]; ok {
					existing.Score += 0.4 * tfidfScore
					existing.PageID = row.PageID
					existing.PageTitle = row.PageTitle
					existing.Text = text
				} else {
					seen[row.ID] = &merged{
						PageID:    row.PageID,
						BlockID:   row.ID,
						PageTitle: row.PageTitle,
						Text:      text,
						Score:     0.4 * tfidfScore,
					}
				}
			}
		}
	}

	// 4. Filter by permission
	var results []SearchResult
	for _, m := range seen {
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

	// 5. Sort by merged score descending, take top 10
	sort.Slice(results, func(i, j int) bool {
		return results[i].Score > results[j].Score
	})
	if len(results) > 10 {
		results = results[:10]
	}

	return results, nil
}

// parseBlockID extracts the block ID from a Milvus chunk ID of the form "pageID_blockID".
func (s *RAGService) parseBlockID(chunkID string) uint {
	parts := strings.Split(chunkID, "_")
	if len(parts) < 2 {
		return 0
	}
	id, err := strconv.ParseUint(parts[1], 10, 64)
	if err != nil {
		return 0
	}
	return uint(id)
}

// IndexBlock generates an embedding for the block's text and stores it in Milvus.
// The chunk ID is formatted as "pageID_blockID".
func (s *RAGService) IndexBlock(ctx context.Context, blockID, pageID, workspaceID uint, text string) error {
	if s.vectorStore == nil || s.embedder == nil {
		return fmt.Errorf("vector store not configured")
	}

	vec, err := s.embedder.Embed(ctx, text)
	if err != nil {
		return fmt.Errorf("embed block %d: %w", blockID, err)
	}

	chunkID := fmt.Sprintf("%d_%d", pageID, blockID)
	meta := map[string]any{
		"page_id":      int64(pageID),
		"workspace_id": int64(workspaceID),
	}

	return s.vectorStore.Insert(ctx, chunkID, vec, meta)
}

// DeleteBlock removes a block's embedding from Milvus.
func (s *RAGService) DeleteBlock(ctx context.Context, blockID, pageID uint) error {
	if s.vectorStore == nil {
		return nil
	}
	chunkID := fmt.Sprintf("%d_%d", pageID, blockID)
	return s.vectorStore.Delete(ctx, chunkID)
}

// IndexAll generates embeddings for all text blocks in the database and
// stores them in Milvus. Call this once after initial Milvus setup or for
// reindexing.
func (s *RAGService) IndexAll(ctx context.Context) error {
	if s.vectorStore == nil || s.embedder == nil {
		return fmt.Errorf("vector store not configured")
	}

	log.Println("rag: starting full reindex into Milvus...")

	type blockRow struct {
		ID          uint
		PageID      uint
		Props       string
		WorkspaceID uint
	}
	var rows []blockRow
	if err := s.db.Table("blocks").
		Select("blocks.id, blocks.page_id, blocks.props, pages.workspace_id").
		Joins("JOIN pages ON pages.id = blocks.page_id").
		Find(&rows).Error; err != nil {
		return fmt.Errorf("query blocks for reindex: %w", err)
	}

	indexed := 0
	failed := 0
	for _, row := range rows {
		text := search.ExtractText(row.Props)
		if text == "" {
			continue
		}

		vec, err := s.embedder.Embed(ctx, text)
		if err != nil {
			log.Printf("rag: failed to embed block %d: %v", row.ID, err)
			failed++
			continue
		}

		chunkID := fmt.Sprintf("%d_%d", row.PageID, row.ID)
		meta := map[string]any{
			"page_id":      int64(row.PageID),
			"workspace_id": int64(row.WorkspaceID),
		}

		if err := s.vectorStore.Insert(ctx, chunkID, vec, meta); err != nil {
			log.Printf("rag: failed to insert block %d into milvus: %v", row.ID, err)
			failed++
			continue
		}
		indexed++
	}

	log.Printf("rag: reindex complete — indexed %d, failed %d", indexed, failed)
	return nil
}

// ensureMilvusIndices creates the Milvus collection and index if not already present.
func (s *RAGService) ensureMilvusIndices(ctx context.Context) error {
	if s.vectorStore == nil {
		return nil
	}
	return s.vectorStore.EnsureCollection(ctx)
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
