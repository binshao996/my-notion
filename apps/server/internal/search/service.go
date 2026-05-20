package search

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/opensearch-project/opensearch-go/v2"
	"github.com/opensearch-project/opensearch-go/v2/opensearchapi"
	"gorm.io/gorm"
)

// Service wraps the OpenSearch client for indexing and searching.
type Service struct {
	client *opensearch.Client
}

// SearchResults holds grouped search results.
type SearchResults struct {
	Pages   []PageResult   `json:"pages"`
	Blocks  []BlockResult  `json:"blocks"`
	Records []RecordResult `json:"records"`
}

type PageResult struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	WorkspaceID uint   `json:"workspace_id"`
}

type BlockResult struct {
	ID          uint   `json:"id"`
	Text        string `json:"text"`
	PageID      uint   `json:"page_id"`
	WorkspaceID uint   `json:"workspace_id"`
	BlockType   string `json:"block_type"`
}

type RecordResult struct {
	ID          uint   `json:"id"`
	Title       string `json:"title"`
	DatabaseID  uint   `json:"database_id"`
	WorkspaceID uint   `json:"workspace_id"`
}

func NewService() (*Service, error) {
	url := os.Getenv("OPENSEARCH_URL")
	if url == "" {
		url = "http://localhost:9200"
	}

	client, err := opensearch.NewClient(opensearch.Config{
		Addresses: []string{url},
	})
	if err != nil {
		return nil, fmt.Errorf("opensearch client: %w", err)
	}

	// Ping to verify connectivity
	_, err = client.Ping()
	if err != nil {
		return nil, fmt.Errorf("opensearch ping: %w", err)
	}

	s := &Service{client: client}
	if err := s.EnsureIndices(); err != nil {
		return nil, fmt.Errorf("opensearch ensure indices: %w", err)
	}

	log.Printf("search: connected to OpenSearch at %s", url)
	return s, nil
}

// EnsureIndices creates the pages, blocks, and records indices if they don't exist.
func (s *Service) EnsureIndices() error {
	indices := []struct {
		name    string
		mapping string
	}{
		{
			name: "pages",
			mapping: `{
				"mappings": {
					"properties": {
						"title": {"type": "text"},
						"workspace_id": {"type": "integer"},
						"created_at": {"type": "date"}
					}
				}
			}`,
		},
		{
			name: "blocks",
			mapping: `{
				"mappings": {
					"properties": {
						"text": {"type": "text"},
						"page_id": {"type": "integer"},
						"workspace_id": {"type": "integer"},
						"block_type": {"type": "keyword"}
					}
				}
			}`,
		},
		{
			name: "records",
			mapping: `{
				"mappings": {
					"properties": {
						"title": {"type": "text"},
						"property_values": {"type": "text"},
						"database_id": {"type": "integer"},
						"workspace_id": {"type": "integer"}
					}
				}
			}`,
		},
	}

	ctx := context.Background()
	for _, idx := range indices {
		req := opensearchapi.IndicesCreateRequest{
			Index: idx.name,
			Body:  strings.NewReader(idx.mapping),
		}
		resp, err := req.Do(ctx, s.client)
		if err != nil {
			return fmt.Errorf("create index %s: %w", idx.name, err)
		}
		resp.Body.Close()
	}
	return nil
}

// IndexPage indexes or updates a page document.
func (s *Service) IndexPage(pageID, workspaceID uint, title string) error {
	body := map[string]any{
		"title":        title,
		"workspace_id": workspaceID,
	}
	return s.index("pages", pageID, body)
}

// DeletePage removes a page from the index.
func (s *Service) DeletePage(pageID uint) error {
	return s.delete("pages", pageID)
}

// IndexBlock indexes or updates a block document.
func (s *Service) IndexBlock(blockID, pageID, workspaceID uint, blockType, text string) error {
	body := map[string]any{
		"text":         text,
		"page_id":      pageID,
		"workspace_id": workspaceID,
		"block_type":   blockType,
	}
	return s.index("blocks", blockID, body)
}

// DeleteBlock removes a block from the index.
func (s *Service) DeleteBlock(blockID uint) error {
	return s.delete("blocks", blockID)
}

// IndexRecord indexes or updates a database record document.
func (s *Service) IndexRecord(recordID, databaseID, workspaceID uint, title, propertyText string) error {
	body := map[string]any{
		"title":           title,
		"property_values": propertyText,
		"database_id":     databaseID,
		"workspace_id":    workspaceID,
	}
	return s.index("records", recordID, body)
}

// DeleteRecord removes a database record from the index.
func (s *Service) DeleteRecord(recordID uint) error {
	return s.delete("records", recordID)
}

func (s *Service) index(index string, id uint, body map[string]any) error {
	data, err := json.Marshal(body)
	if err != nil {
		return err
	}

	docID := strconv.FormatUint(uint64(id), 10)
	req := opensearchapi.IndexRequest{
		Index:      index,
		DocumentID: docID,
		Body:       bytes.NewReader(data),
		Refresh:    "true",
	}
	resp, err := req.Do(context.Background(), s.client)
	if err != nil {
		return fmt.Errorf("index %s/%s: %w", index, docID, err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return fmt.Errorf("index %s/%s: %s", index, docID, resp.String())
	}
	return nil
}

func (s *Service) delete(index string, id uint) error {
	docID := strconv.FormatUint(uint64(id), 10)
	req := opensearchapi.DeleteRequest{
		Index:      index,
		DocumentID: docID,
	}
	resp, err := req.Do(context.Background(), s.client)
	if err != nil {
		return fmt.Errorf("delete %s/%s: %w", index, docID, err)
	}
	defer resp.Body.Close()
	return nil
}

// Search performs a multi-index search filtered by workspace.
func (s *Service) Search(workspaceID uint, query string) (*SearchResults, error) {
	if strings.TrimSpace(query) == "" {
		return &SearchResults{}, nil
	}

	// Build multi-search body
	searchBody := map[string]any{
		"query": map[string]any{
			"bool": map[string]any{
				"must": []map[string]any{
					{
						"multi_match": map[string]any{
							"query":  query,
							"fields": []string{"title", "text", "property_values"},
						},
					},
				},
				"filter": []map[string]any{
					{
						"term": map[string]any{
							"workspace_id": workspaceID,
						},
					},
				},
			},
		},
		"size": 20,
	}

	bodyBytes, _ := json.Marshal(searchBody)

	req := opensearchapi.SearchRequest{
		Index: []string{"pages", "blocks", "records"},
		Body:  bytes.NewReader(bodyBytes),
	}
	resp, err := req.Do(context.Background(), s.client)
	if err != nil {
		return nil, fmt.Errorf("search: %w", err)
	}
	defer resp.Body.Close()

	if resp.IsError() {
		return nil, fmt.Errorf("search error: %s", resp.String())
	}

	var result struct {
		Hits struct {
			Hits []struct {
				Index  string          `json:"_index"`
				Source json.RawMessage `json:"_source"`
			} `json:"hits"`
		} `json:"hits"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("search decode: %w", err)
	}

	out := &SearchResults{}
	for _, hit := range result.Hits.Hits {
		switch hit.Index {
		case "pages":
			var p PageResult
			if err := json.Unmarshal(hit.Source, &p); err == nil {
				out.Pages = append(out.Pages, p)
			}
		case "blocks":
			var b BlockResult
			if err := json.Unmarshal(hit.Source, &b); err == nil {
				out.Blocks = append(out.Blocks, b)
			}
		case "records":
			var r RecordResult
			if err := json.Unmarshal(hit.Source, &r); err == nil {
				out.Records = append(out.Records, r)
			}
		}
	}
	return out, nil
}

// ReindexAll reindexes all pages, blocks, and records from the database.
func (s *Service) ReindexAll(db *gorm.DB) error {
	log.Println("search: starting reindex of all content...")

	// Reindex pages
	var pages []struct {
		ID          uint
		Title       string
		WorkspaceID uint
	}
	if err := db.Table("pages").Select("id, title, workspace_id").Where("archived = false").Find(&pages).Error; err != nil {
		return fmt.Errorf("reindex pages: %w", err)
	}
	for _, p := range pages {
		if err := s.IndexPage(p.ID, p.WorkspaceID, p.Title); err != nil {
			log.Printf("search: failed to index page %d: %v", p.ID, err)
		}
	}

	// Reindex blocks
	var blocks []struct {
		ID          uint
		Type        string
		Props       string
		PageID      uint
		WorkspaceID uint
	}
	if err := db.Table("blocks").
		Select("blocks.id, blocks.type, blocks.props, blocks.page_id, pages.workspace_id").
		Joins("JOIN pages ON pages.id = blocks.page_id").
		Find(&blocks).Error; err != nil {
		return fmt.Errorf("reindex blocks: %w", err)
	}
	for _, b := range blocks {
		text := ExtractText(b.Props)
		if text != "" {
			if err := s.IndexBlock(b.ID, b.PageID, b.WorkspaceID, b.Type, text); err != nil {
				log.Printf("search: failed to index block %d: %v", b.ID, err)
			}
		}
	}

	// Reindex records
	var records []struct {
		ID          uint
		DatabaseID  uint
		WorkspaceID uint
	}
	if err := db.Table("records").
		Select("records.id, records.database_id, databases.workspace_id").
		Joins("JOIN databases ON databases.id = records.database_id").
		Find(&records).Error; err != nil {
		return fmt.Errorf("reindex records: %w", err)
	}
	for _, r := range records {
		// Get property values for this record
		var pvs []struct {
			PropertyName string
			Value        string
		}
		db.Table("property_values").
			Select("properties.name as property_name, property_values.value").
			Joins("JOIN properties ON properties.id = property_values.property_id").
			Where("property_values.record_id = ?", r.ID).
			Find(&pvs)

		title := ""
		var propTexts []string
		for _, pv := range pvs {
			var v map[string]any
			if err := json.Unmarshal([]byte(pv.Value), &v); err != nil {
				continue
			}
			if t, ok := v["text"].(string); ok {
				if pv.PropertyName == "title" || title == "" {
					title = t
				}
				propTexts = append(propTexts, t)
			}
			if t, ok := v["title"].(string); ok {
				if title == "" {
					title = t
				}
				propTexts = append(propTexts, t)
			}
		}
		if err := s.IndexRecord(r.ID, r.DatabaseID, r.WorkspaceID, title, strings.Join(propTexts, " ")); err != nil {
			log.Printf("search: failed to index record %d: %v", r.ID, err)
		}
	}

	log.Printf("search: reindex complete (%d pages, %d blocks, %d records)", len(pages), len(blocks), len(records))
	return nil
}

// ExtractText pulls readable text from a JSONB block props string.
func ExtractText(propsJSON string) string {
	var props map[string]any
	if err := json.Unmarshal([]byte(propsJSON), &props); err != nil {
		return ""
	}
	if t, ok := props["text"]; ok {
		switch v := t.(type) {
		case string:
			return v
		}
	}
	if t, ok := props["title"]; ok {
		switch v := t.(type) {
		case string:
			return v
		}
	}
	return ""
}
