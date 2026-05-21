package main

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/bin-ke/my-notion/internal/search"
	"github.com/bin-ke/my-notion/pkg/db"
	"github.com/bin-ke/my-notion/pkg/queue"
	myredis "github.com/bin-ke/my-notion/pkg/redis"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

func main() {
	// Connect to database
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "postgres://notion:notion_dev@localhost:5432/my_notion?sslmode=disable"
	}
	database, err := db.Connect(dsn)
	if err != nil {
		log.Fatalf("worker: failed to connect to database: %v", err)
	}

	// Connect to Redis (fatal for worker)
	redisAddr := os.Getenv("REDIS_URL")
	if redisAddr == "" {
		redisAddr = "redis://localhost:6379"
	}
	rdb, err := myredis.Connect(redisAddr)
	if err != nil {
		log.Fatalf("worker: failed to connect to redis: %v", err)
	}
	defer rdb.Close()

	// Create search service (no RedisClient so it indexes directly to OpenSearch)
	searchService, err := search.NewService()
	if err != nil {
		log.Printf("WARNING: worker: search service not available: %v", err)
		log.Println("worker: hanging (no search backend)")
		select {}
	}

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		sig := <-sigCh
		log.Printf("worker: received %v, shutting down...", sig)
		cancel()
	}()

	log.Println("worker: started, waiting for search indexing jobs...")

	for {
		select {
		case <-ctx.Done():
			log.Println("worker: stopped")
			return
		default:
		}

		job, err := queue.DequeueSearchIndex(rdb)
		if err != nil {
			if err == redis.Nil {
				// Timeout with no jobs available, continue polling
				continue
			}
			log.Printf("worker: dequeue error: %v", err)
			continue
		}

		log.Printf("worker: processing job type=%s id=%d", job.Type, job.ID)
		processJob(database, searchService, job)
		queue.AckSearchIndex(rdb, job)
	}
}

// processJob looks up the entity from the database and indexes it into OpenSearch.
func processJob(database *gorm.DB, svc *search.Service, job *queue.SearchJob) {
	switch job.Type {
	case "page":
		var page db.Page
		if err := database.First(&page, job.ID).Error; err != nil {
			log.Printf("worker: page %d not found: %v", job.ID, err)
			return
		}
		if err := svc.IndexPage(page.ID, page.WorkspaceID, page.Title); err != nil {
			log.Printf("worker: index page %d error: %v", job.ID, err)
		}

	case "block":
		var block db.Block
		if err := database.First(&block, job.ID).Error; err != nil {
			log.Printf("worker: block %d not found: %v", job.ID, err)
			return
		}
		var page db.Page
		if err := database.First(&page, block.PageID).Error; err != nil {
			log.Printf("worker: page for block %d not found: %v", job.ID, err)
			return
		}
		text := search.ExtractText(block.Props)
		if err := svc.IndexBlock(block.ID, block.PageID, page.WorkspaceID, block.Type, text); err != nil {
			log.Printf("worker: index block %d error: %v", job.ID, err)
		}

	case "record":
		var record db.Record
		if err := database.First(&record, job.ID).Error; err != nil {
			log.Printf("worker: record %d not found: %v", job.ID, err)
			return
		}
		var dbase db.Database
		if err := database.First(&dbase, record.DatabaseID).Error; err != nil {
			log.Printf("worker: database for record %d not found: %v", job.ID, err)
			return
		}

		// Get property values for this record
		var pvs []struct {
			PropertyName string
			Value        string
		}
		database.Table("property_values").
			Select("properties.name as property_name, property_values.value").
			Joins("JOIN properties ON properties.id = property_values.property_id").
			Where("property_values.record_id = ?", job.ID).
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
		if err := svc.IndexRecord(record.ID, record.DatabaseID, dbase.WorkspaceID, title, strings.Join(propTexts, " ")); err != nil {
			log.Printf("worker: index record %d error: %v", job.ID, err)
		}

	default:
		log.Printf("worker: unknown job type: %s", job.Type)
	}
}
