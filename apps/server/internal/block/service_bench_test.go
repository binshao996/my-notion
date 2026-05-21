package block

import (
	"fmt"
	"testing"

	"github.com/bin-ke/my-notion/pkg/db"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupBenchDB(b *testing.B) *gorm.DB {
	gormDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		b.Fatal(err)
	}
	if err := gormDB.AutoMigrate(&db.Workspace{}, &db.Page{}, &db.Block{}); err != nil {
		b.Fatal(err)
	}
	ws := db.Workspace{Name: "bench-ws"}
	if err := gormDB.Create(&ws).Error; err != nil {
		b.Fatal(err)
	}
	return gormDB
}

func setupBenchPage(b *testing.B, gormDB *gorm.DB) db.Page {
	page := db.Page{WorkspaceID: 1, Title: "bench-page", CreatedBy: 1}
	if err := gormDB.Create(&page).Error; err != nil {
		b.Fatal(err)
	}
	return page
}

func makeBlocks(n int) []db.Block {
	types := []string{"text", "heading", "code", "image", "checkbox"}
	blocks := make([]db.Block, n)
	for i := 0; i < n; i++ {
		blocks[i] = db.Block{
			Type:  types[i%len(types)],
			Props: fmt.Sprintf(`{"text":"block-%d","type":"%s"}`, i, types[i%len(types)]),
		}
	}
	return blocks
}

func BenchmarkBatchSave_1000Blocks(b *testing.B) {
	gormDB := setupBenchDB(b)
	page := setupBenchPage(b, gormDB)
	svc := NewService(gormDB)
	blocks := makeBlocks(1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.BatchSave(page.ID, blocks)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkGetByPage_1000Blocks(b *testing.B) {
	gormDB := setupBenchDB(b)
	page := setupBenchPage(b, gormDB)
	svc := NewService(gormDB)
	blocks := makeBlocks(1000)
	if _, err := svc.BatchSave(page.ID, blocks); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := svc.GetByPage(page.ID)
		if err != nil {
			b.Fatal(err)
		}
	}
}
