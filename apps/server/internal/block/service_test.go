package block

import (
	"encoding/json"
	"testing"

	"github.com/bin-ke/my-notion/pkg/db"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	err = gormDB.AutoMigrate(&db.Workspace{}, &db.Page{}, &db.Block{})
	require.NoError(t, err)
	return gormDB
}

// setupPage creates a workspace and a page, returns the page.
func setupPage(t *testing.T, gormDB *gorm.DB) db.Page {
	ws := db.Workspace{Name: "test-workspace"}
	require.NoError(t, gormDB.Create(&ws).Error)

	page := db.Page{WorkspaceID: ws.ID, Title: "test page", CreatedBy: 1}
	require.NoError(t, gormDB.Create(&page).Error)
	return page
}

// ---------------------------------------------------------------------------
// NewService
// ---------------------------------------------------------------------------

func TestNewService(t *testing.T) {
	gormDB := setupTestDB(t)
	svc := NewService(gormDB)

	require.NotNil(t, svc)
	assert.Equal(t, gormDB, svc.DB)
	assert.Nil(t, svc.SearchService, "SearchService should be nil when not explicitly set")
}

// ---------------------------------------------------------------------------
// GetByPage
// ---------------------------------------------------------------------------

func TestGetByPage_EmptyDB_ReturnsEmptySlice(t *testing.T) {
	gormDB := setupTestDB(t)
	page := setupPage(t, gormDB)
	svc := NewService(gormDB)

	blocks, err := svc.GetByPage(page.ID)

	require.NoError(t, err)
	assert.Empty(t, blocks)
	assert.NotNil(t, blocks, "should return empty slice, not nil")
}

func TestGetByPage_NonExistentPage_ReturnsEmptySlice(t *testing.T) {
	gormDB := setupTestDB(t)
	svc := NewService(gormDB)

	blocks, err := svc.GetByPage(99999)

	require.NoError(t, err)
	assert.Empty(t, blocks)
}

func TestGetByPage_ReturnsBlocksInPositionOrder(t *testing.T) {
	gormDB := setupTestDB(t)
	page := setupPage(t, gormDB)
	svc := NewService(gormDB)

	blocks := []db.Block{
		{Type: "text", Props: `{"text":"first"}`},
		{Type: "heading", Props: `{"text":"second"}`},
		{Type: "text", Props: `{"text":"third"}`},
	}
	saved, err := svc.BatchSave(page.ID, blocks)
	require.NoError(t, err)
	require.Len(t, saved, 3)

	result, err := svc.GetByPage(page.ID)
	require.NoError(t, err)
	require.Len(t, result, 3)

	for i := 1; i < len(result); i++ {
		assert.True(t, result[i-1].Position < result[i].Position,
			"blocks should be in ascending position order, got %q before %q",
			result[i-1].Position, result[i].Position)
	}
}

// ---------------------------------------------------------------------------
// BatchSave -- basic save
// ---------------------------------------------------------------------------

func TestBatchSave_SavesBlocksAndAssignsIDs(t *testing.T) {
	gormDB := setupTestDB(t)
	page := setupPage(t, gormDB)
	svc := NewService(gormDB)

	blocks := []db.Block{
		{Type: "text", Props: `{"text":"hello"}`},
		{Type: "heading", Props: `{"text":"important heading"}`},
	}

	saved, err := svc.BatchSave(page.ID, blocks)

	require.NoError(t, err)
	require.Len(t, saved, 2)

	for i, b := range saved {
		assert.NotZero(t, b.ID, "saved block %d should have a non-zero auto-increment ID", i)
	}

	// Verify persistence: blocks exist in the DB
	var all []db.Block
	err = gormDB.Where("page_id = ?", page.ID).Find(&all).Error
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestBatchSave_AssignsCorrectPageID(t *testing.T) {
	gormDB := setupTestDB(t)
	page := setupPage(t, gormDB)
	svc := NewService(gormDB)

	blocks := []db.Block{
		{Type: "text", Props: `{"text":"belongs to page"}`},
	}

	saved, err := svc.BatchSave(page.ID, blocks)

	require.NoError(t, err)
	require.Len(t, saved, 1)
	assert.Equal(t, page.ID, saved[0].PageID)

	// Double-check in DB
	var b db.Block
	err = gormDB.First(&b, saved[0].ID).Error
	require.NoError(t, err)
	assert.Equal(t, page.ID, b.PageID)
}

func TestBatchSave_ResetsExistingBlockIDs(t *testing.T) {
	// If the caller passes blocks with non-zero IDs, they should be ignored
	// (set to 0) so GORM creates new records instead of updating.
	gormDB := setupTestDB(t)
	page := setupPage(t, gormDB)
	svc := NewService(gormDB)

	blocks := []db.Block{
		{ID: 42, Type: "text", Props: `{"text":"with-explicit-id"}`},
	}

	saved, err := svc.BatchSave(page.ID, blocks)

	require.NoError(t, err)
	require.Len(t, saved, 1)
	assert.NotEqual(t, uint(42), saved[0].ID, "explicit ID should be discarded and a new one assigned")
}

// ---------------------------------------------------------------------------
// BatchSave -- position assignment
// ---------------------------------------------------------------------------

func TestBatchSave_AssignsNonEmptyPositions(t *testing.T) {
	gormDB := setupTestDB(t)
	page := setupPage(t, gormDB)
	svc := NewService(gormDB)

	blocks := []db.Block{
		{Type: "text", Props: `{"text":"a"}`},
		{Type: "text", Props: `{"text":"b"}`},
		{Type: "text", Props: `{"text":"c"}`},
		{Type: "text", Props: `{"text":"d"}`},
		{Type: "text", Props: `{"text":"e"}`},
	}

	saved, err := svc.BatchSave(page.ID, blocks)

	require.NoError(t, err)
	require.Len(t, saved, 5)

	for i, b := range saved {
		assert.NotEmpty(t, b.Position, "saved block %d should have a non-empty position string", i)
	}
}

func TestBatchSave_AssignsUniquePositions(t *testing.T) {
	gormDB := setupTestDB(t)
	page := setupPage(t, gormDB)
	svc := NewService(gormDB)

	blocks := []db.Block{
		{Type: "text", Props: `{}`},
		{Type: "text", Props: `{}`},
		{Type: "text", Props: `{}`},
	}

	saved, err := svc.BatchSave(page.ID, blocks)

	require.NoError(t, err)
	require.Len(t, saved, 3)

	seen := make(map[string]bool)
	for _, b := range saved {
		require.False(t, seen[b.Position], "duplicate position found: %q", b.Position)
		seen[b.Position] = true
	}
}

func TestBatchSave_PositionsAreLexicographicForOrdering(t *testing.T) {
	// position.GenerateN is designed so that positions sort
	// lexicographically in the order they are generated.
	gormDB := setupTestDB(t)
	page := setupPage(t, gormDB)
	svc := NewService(gormDB)

	blocks := make([]db.Block, 10)
	for i := range blocks {
		blocks[i] = db.Block{Type: "text", Props: `{}`}
	}

	saved, err := svc.BatchSave(page.ID, blocks)

	require.NoError(t, err)
	require.Len(t, saved, 10)

	result, err := svc.GetByPage(page.ID)
	require.NoError(t, err)
	require.Len(t, result, 10)

	for i := 1; i < len(result); i++ {
		assert.True(t, result[i-1].Position < result[i].Position,
			"blocks should be in ascending lexicographic order, but %q >= %q",
			result[i-1].Position, result[i].Position)
	}
}

// ---------------------------------------------------------------------------
// BatchSave -- deletion / replacement
// ---------------------------------------------------------------------------

func TestBatchSave_EmptyBlocksList_Noop(t *testing.T) {
	// Saving an empty blocks list to a page that has no blocks is a no-op.
	gormDB := setupTestDB(t)
	page := setupPage(t, gormDB)
	svc := NewService(gormDB)

	saved, err := svc.BatchSave(page.ID, []db.Block{})

	require.NoError(t, err)
	assert.Empty(t, saved)

	var count int64
	gormDB.Model(&db.Block{}).Where("page_id = ?", page.ID).Count(&count)
	assert.Equal(t, int64(0), count)
}

func TestBatchSave_ReplacesOldBlocks_OldIDsRemoved(t *testing.T) {
	gormDB := setupTestDB(t)
	page := setupPage(t, gormDB)
	svc := NewService(gormDB)

	// Save an initial set
	first := []db.Block{
		{Type: "text", Props: `{"text":"old"}`},
	}
	old, err := svc.BatchSave(page.ID, first)
	require.NoError(t, err)
	require.Len(t, old, 1)
	oldID := old[0].ID

	// Save a new set on the same page
	second := []db.Block{
		{Type: "heading", Props: `{"text":"new1"}`},
		{Type: "text", Props: `{"text":"new2"}`},
	}
	newBlocks, err := svc.BatchSave(page.ID, second)
	require.NoError(t, err)
	require.Len(t, newBlocks, 2)

	// The old block should no longer exist
	var oldBlock db.Block
	err = gormDB.First(&oldBlock, oldID).Error
	assert.Error(t, err, "old block ID=%d should have been deleted", oldID)

	// Both new blocks should exist
	for _, b := range newBlocks {
		var found db.Block
		err := gormDB.First(&found, b.ID).Error
		assert.NoError(t, err, "new block ID=%d should be present", b.ID)
	}
}

func TestBatchSave_ReplacesOldBlocks_OnlyNewRemain(t *testing.T) {
	gormDB := setupTestDB(t)
	page := setupPage(t, gormDB)
	svc := NewService(gormDB)

	// Save 3 initial blocks
	initial := []db.Block{
		{Type: "text", Props: `{"text":"a"}`},
		{Type: "text", Props: `{"text":"b"}`},
		{Type: "text", Props: `{"text":"c"}`},
	}
	_, err := svc.BatchSave(page.ID, initial)
	require.NoError(t, err)

	// Replace with 2 different blocks
	replacement := []db.Block{
		{Type: "heading", Props: `{"text":"x"}`},
		{Type: "heading", Props: `{"text":"y"}`},
	}
	saved, err := svc.BatchSave(page.ID, replacement)
	require.NoError(t, err)
	require.Len(t, saved, 2)

	// Only 2 blocks should remain for this page
	var all []db.Block
	err = gormDB.Where("page_id = ?", page.ID).Find(&all).Error
	require.NoError(t, err)
	assert.Len(t, all, 2)
}

func TestBatchSave_EmptyBlocksAfterHavingBlocks_ClearsPage(t *testing.T) {
	gormDB := setupTestDB(t)
	page := setupPage(t, gormDB)
	svc := NewService(gormDB)

	// Save blocks first
	initial := []db.Block{
		{Type: "text", Props: `{"text":"one"}`},
		{Type: "text", Props: `{"text":"two"}`},
	}
	old, err := svc.BatchSave(page.ID, initial)
	require.NoError(t, err)
	require.Len(t, old, 2)
	oldIDs := []uint{old[0].ID, old[1].ID}

	// Now save empty to clear the page
	cleared, err := svc.BatchSave(page.ID, []db.Block{})
	require.NoError(t, err)
	assert.Empty(t, cleared)

	// Page should have no blocks
	var count int64
	gormDB.Model(&db.Block{}).Where("page_id = ?", page.ID).Count(&count)
	assert.Equal(t, int64(0), count, "page should have zero blocks after clearing")

	// Old block IDs should not exist in DB anymore
	for _, id := range oldIDs {
		var b db.Block
		err := gormDB.First(&b, id).Error
		assert.Error(t, err, "old block ID=%d should be gone", id)
	}
}

func TestBatchSave_DoesNotAffectOtherPages(t *testing.T) {
	// Blocks from other pages must not be touched when saving a different page.
	gormDB := setupTestDB(t)
	page1 := setupPage(t, gormDB)
	page2 := setupPage(t, gormDB)
	svc := NewService(gormDB)

	// Save blocks for both pages
	b1, err := svc.BatchSave(page1.ID, []db.Block{
		{Type: "text", Props: `{"text":"p1"}`},
	})
	require.NoError(t, err)
	b2, err := svc.BatchSave(page2.ID, []db.Block{
		{Type: "text", Props: `{"text":"p2"}`},
	})
	require.NoError(t, err)

	// Replace page1 blocks
	_, err = svc.BatchSave(page1.ID, []db.Block{
		{Type: "heading", Props: `{"text":"p1-new"}`},
	})
	require.NoError(t, err)

	// page1 old block should be gone
	var p1OldBlock db.Block
	err = gormDB.First(&p1OldBlock, b1[0].ID).Error
	assert.Error(t, err, "page1 old block should have been replaced (deleted)")

	// page2 block must still exist
	var p2Block db.Block
	err = gormDB.First(&p2Block, b2[0].ID).Error
	require.NoError(t, err, "blocks from other pages must not be affected")
	assert.Equal(t, page2.ID, p2Block.PageID)
}

func TestBatchSave_BlocksWithDifferentTypes(t *testing.T) {
	gormDB := setupTestDB(t)
	page := setupPage(t, gormDB)
	svc := NewService(gormDB)

	blocks := []db.Block{
		{Type: "text", Props: `{"text":"plain"}`},
		{Type: "heading", Props: `{"text":"title"}`},
		{Type: "code", Props: `{"text":"fmt.Println()"}`},
		{Type: "image", Props: `{"src":"https://example.com/img.png"}`},
		{Type: "checkbox", Props: `{"checked":true}`},
	}

	saved, err := svc.BatchSave(page.ID, blocks)

	require.NoError(t, err)
	require.Len(t, saved, 5)

	expectedTypes := []string{"text", "heading", "code", "image", "checkbox"}
	for i, b := range saved {
		assert.Equal(t, expectedTypes[i], b.Type, "block %d type mismatch", i)
		assert.NotZero(t, b.ID)
		assert.NotEmpty(t, b.Position)
	}
}

// ---------------------------------------------------------------------------
// ApplyOps (stub)
// ---------------------------------------------------------------------------

func TestApplyOps_ReturnsNil(t *testing.T) {
	svc := &Service{}

	err := svc.ApplyOps(1, nil)
	assert.NoError(t, err)

	err = svc.ApplyOps(1, []OpRequest{})
	assert.NoError(t, err)

	err = svc.ApplyOps(1, []OpRequest{
		{Type: "insert", Position: "a0", Props: json.RawMessage(`{"text":"new block"}`)},
		{Type: "delete", BlockID: 5},
	})
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// BatchSave -- SearchService is nil (no indexing)
// ---------------------------------------------------------------------------

func TestBatchSave_WorksWithNilSearchService(t *testing.T) {
	gormDB := setupTestDB(t)
	page := setupPage(t, gormDB)
	svc := NewService(gormDB)

	assert.Nil(t, svc.SearchService, "SearchService should be nil when not explicitly set")

	blocks := []db.Block{
		{Type: "text", Props: `{"text":"no search service"}`},
	}

	saved, err := svc.BatchSave(page.ID, blocks)

	require.NoError(t, err)
	require.Len(t, saved, 1)
	assert.NotZero(t, saved[0].ID)
}
