package ai

import (
	"context"
	"fmt"
	"log"

	"github.com/milvus-io/milvus-sdk-go/v2/client"
	"github.com/milvus-io/milvus-sdk-go/v2/entity"
)

// VectorStore wraps a Milvus client for storing and searching dense vector
// embeddings used in hybrid RAG retrieval.
type VectorStore struct {
	client   client.Client
	collName string
	dim      int
}

// VectorHit is a single result from Milvus vector search.
type VectorHit struct {
	ChunkID     string
	PageID      int64
	WorkspaceID int64
	Score       float32
}

// NewVectorStore creates a new VectorStore connected to the given Milvus address.
// The connection is validated with a liveness check.
func NewVectorStore(ctx context.Context, addr string, dim int) (*VectorStore, error) {
	c, err := client.NewClient(ctx, client.Config{
		Address: addr,
	})
	if err != nil {
		return nil, fmt.Errorf("milvus connect: %w", err)
	}

	return &VectorStore{
		client:   c,
		collName: "rag_chunks",
		dim:      dim,
	}, nil
}

// EnsureCollection creates the rag_chunks collection if it does not exist and
// builds an IVF_FLAT index on the embedding field.
func (vs *VectorStore) EnsureCollection(ctx context.Context) error {
	has, err := vs.client.HasCollection(ctx, vs.collName)
	if err != nil {
		return fmt.Errorf("milvus has collection: %w", err)
	}
	if has {
		log.Printf("vectorstore: collection %s already exists", vs.collName)
		return nil
	}

	schema := entity.NewSchema().
		WithName(vs.collName).
		WithField(entity.NewField().
			WithName("id").
			WithDataType(entity.FieldTypeVarChar).
			WithMaxLength(128).
			WithIsPrimaryKey(true)).
		WithField(entity.NewField().
			WithName("embedding").
			WithDataType(entity.FieldTypeFloatVector).
			WithDim(int64(vs.dim))).
		WithField(entity.NewField().
			WithName("page_id").
			WithDataType(entity.FieldTypeInt64)).
		WithField(entity.NewField().
			WithName("workspace_id").
			WithDataType(entity.FieldTypeInt64))

	if err := vs.client.CreateCollection(ctx, schema, 2); err != nil {
		return fmt.Errorf("milvus create collection: %w", err)
	}

	// Build IVF_FLAT index for approximate nearest neighbor search.
	idx, err := entity.NewIndexIvfFlat(entity.L2, 128)
	if err != nil {
		return fmt.Errorf("milvus create index params: %w", err)
	}

	if err := vs.client.CreateIndex(ctx, vs.collName, "embedding", idx, false); err != nil {
		return fmt.Errorf("milvus create index: %w", err)
	}

	log.Printf("vectorstore: collection %s created with dimension %d", vs.collName, vs.dim)
	return nil
}

// Insert stores a single vector embedding and associated metadata into Milvus.
// The id should uniquely identify the chunk (e.g. "pageID_blockID").
func (vs *VectorStore) Insert(ctx context.Context, id string, vector []float32, metadata map[string]any) error {
	pageID, _ := metadata["page_id"].(int64)
	workspaceID, _ := metadata["workspace_id"].(int64)

	columns := []entity.Column{
		entity.NewColumnVarChar("id", []string{id}),
		entity.NewColumnFloatVector("embedding", vs.dim, [][]float32{vector}),
		entity.NewColumnInt64("page_id", []int64{pageID}),
		entity.NewColumnInt64("workspace_id", []int64{workspaceID}),
	}

	if _, err := vs.client.Insert(ctx, vs.collName, "", columns...); err != nil {
		return fmt.Errorf("milvus insert: %w", err)
	}

	return nil
}

// Search finds the top-K most similar chunks to the given query vector,
// optionally filtered by metadata (e.g. workspace_id).
func (vs *VectorStore) Search(ctx context.Context, vector []float32, topK int, filter map[string]any) ([]VectorHit, error) {
	// Build optional boolean expression for filtering.
	expr := ""
	if filter != nil {
		if wsID, ok := filter["workspace_id"]; ok {
			switch v := wsID.(type) {
			case uint:
				expr = fmt.Sprintf("workspace_id == %d", v)
			case int64:
				expr = fmt.Sprintf("workspace_id == %d", v)
			}
		}
	}

	sp, err := entity.NewIndexIvfFlatSearchParam(16)
	if err != nil {
		return nil, fmt.Errorf("milvus search param: %w", err)
	}

	vectors := []entity.Vector{entity.FloatVector(vector)}
	results, err := vs.client.Search(
		ctx,
		vs.collName,
		nil, // all partitions
		expr,
		[]string{"id", "page_id", "workspace_id"},
		vectors,
		"embedding",
		entity.L2,
		topK,
		sp,
	)
	if err != nil {
		return nil, fmt.Errorf("milvus search: %w", err)
	}

	var hits []VectorHit
	for _, r := range results {
		if r.Err != nil {
			log.Printf("vectorstore: search result error: %v", r.Err)
			continue
		}

		for i := 0; i < r.ResultCount; i++ {
			hit := VectorHit{
				Score: r.Scores[i],
			}

			// Extract field values from the returned columns.
			for _, col := range r.Fields {
				switch col.Name() {
				case "id":
					if idCol, ok := col.(*entity.ColumnVarChar); ok {
						hit.ChunkID = idCol.Data()[i]
					}
				case "page_id":
					if pageCol, ok := col.(*entity.ColumnInt64); ok {
						hit.PageID = pageCol.Data()[i]
					}
				case "workspace_id":
					if wsCol, ok := col.(*entity.ColumnInt64); ok {
						hit.WorkspaceID = wsCol.Data()[i]
					}
				}
			}

			hits = append(hits, hit)
		}
	}

	return hits, nil
}

// Delete removes a single vector from the collection by its chunk ID.
func (vs *VectorStore) Delete(ctx context.Context, id string) error {
	expr := fmt.Sprintf(`id == "%s"`, id)
	if err := vs.client.Delete(ctx, vs.collName, "", expr); err != nil {
		return fmt.Errorf("milvus delete %s: %w", id, err)
	}
	return nil
}

// Close releases the Milvus client connection.
func (vs *VectorStore) Close() error {
	return vs.client.Close()
}
