package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	databasepkg "github.com/bin-ke/my-notion/internal/database"
	"github.com/bin-ke/my-notion/pkg/db"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// AutofillService uses AI to fill database property values from record content.
type AutofillService struct {
	client    *Client
	db        *gorm.DB
	recordSvc *databasepkg.RecordService
	rdb       *redis.Client
	jobs      map[string]*AutofillJob
	mu        sync.Mutex
}

// NewAutofillService creates a new AutofillService.
func NewAutofillService(client *Client, db *gorm.DB, recordSvc *databasepkg.RecordService, rdb *redis.Client) *AutofillService {
	return &AutofillService{
		client:    client,
		db:        db,
		recordSvc: recordSvc,
		rdb:       rdb,
		jobs:      make(map[string]*AutofillJob),
	}
}

// FillProperty fills ONE property for ONE record using AI.
func (s *AutofillService) FillProperty(userID uint, req *AutofillRequest) error {
	if len(req.RecordIDs) == 0 {
		return fmt.Errorf("at least one record_id is required")
	}

	recordID := req.RecordIDs[0]

	// 1. Load the record (with property values and properties)
	_, values, properties, err := s.recordSvc.GetByID(recordID)
	if err != nil {
		return fmt.Errorf("load record: %w", err)
	}

	// 2. Load target property info
	var targetProp db.Property
	if err := s.db.First(&targetProp, req.PropertyID).Error; err != nil {
		return fmt.Errorf("load property: %w", err)
	}

	// 3. Get the record struct for the page reference
	record, _, _, _ := s.recordSvc.GetByID(recordID)
	if record == nil {
		return fmt.Errorf("record not found")
	}

	// 4. Extract content from the record's page blocks
	content, err := s.getRecordContent(record)
	if err != nil {
		return fmt.Errorf("get record content: %w", err)
	}

	// Include existing property values as additional context
	var existingVals []string
	for _, pv := range values {
		for _, prop := range properties {
			if prop.ID == pv.PropertyID && prop.ID != targetProp.ID {
				var v map[string]any
				if err := json.Unmarshal([]byte(pv.Value), &v); err != nil {
					continue
				}
				for _, val := range v {
					existingVals = append(existingVals, fmt.Sprintf("%s: %v", prop.Name, val))
				}
				break
			}
		}
	}

	if len(existingVals) > 0 {
		content = content + "\n\nExisting properties: " + strings.Join(existingVals, "; ")
	}

	// 5. Build prompt based on property type
	prompt := s.buildPrompt(targetProp.Name, targetProp.Type, targetProp.Config, content)
	if req.Instruction != "" {
		prompt = req.Instruction + "\n\n" + prompt
	}

	// 6. Call DeepSeek to generate the value
	aiResp, err := s.client.ChatCompletion(&ChatRequest{
		Model: ModelForTask("autofill"),
		Messages: []ChatMessage{
			{Role: "system", Content: "You are a data extraction assistant. Return only the requested value with no additional text, explanation, or formatting."},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.1,
		MaxTokens:   256,
	})
	if err != nil {
		return fmt.Errorf("ai call: %w", err)
	}

	generatedText := strings.TrimSpace(aiResp.Choices[0].Message.Content)

	// 7. Parse response into the correct property value format
	valueJSON, err := s.formatPropertyValue(targetProp.Type, generatedText)
	if err != nil {
		return fmt.Errorf("format value: %w", err)
	}

	// 8. Update the record
	updateValues := map[uint]string{
		targetProp.ID: valueJSON,
	}
	if err := s.recordSvc.Update(recordID, updateValues); err != nil {
		return fmt.Errorf("update record: %w", err)
	}

	return nil
}

// FillPropertyBatch fills a property for multiple records and tracks progress.
// Returns a job ID (UUID) that can be polled for status.
// Jobs are persisted to Redis (with 1h TTL) when available; falls back to in-memory.
func (s *AutofillService) FillPropertyBatch(userID uint, req *AutofillRequest) (string, error) {
	// Determine which records to process
	var recordIDs []uint
	if len(req.RecordIDs) > 0 {
		recordIDs = req.RecordIDs
	} else {
		// Get all records in the database
		records, err := s.recordSvc.ListByDatabase(req.DatabaseID)
		if err != nil {
			return "", fmt.Errorf("list records: %w", err)
		}
		for _, r := range records {
			recordIDs = append(recordIDs, r.ID)
		}
	}

	if len(recordIDs) == 0 {
		return "", fmt.Errorf("no records to process")
	}

	jobID := uuid.New().String()
	job := &AutofillJob{
		ID:         jobID,
		DatabaseID: req.DatabaseID,
		PropertyID: req.PropertyID,
		Total:      len(recordIDs),
		Completed:  0,
		Failed:     0,
		Status:     "running",
		CreatedAt:  time.Now(),
	}

	s.mu.Lock()
	s.jobs[jobID] = job
	s.mu.Unlock()

	// Persist initial job state to Redis
	s.saveJob(job)

	// Process records in a goroutine (sequential to avoid rate limits)
	go func() {
		for _, recordID := range recordIDs {
			singleReq := &AutofillRequest{
				DatabaseID:   req.DatabaseID,
				PropertyID:   req.PropertyID,
				SourcePropID: req.SourcePropID,
				RecordIDs:    []uint{recordID},
				Instruction:  req.Instruction,
			}

			if err := s.FillProperty(userID, singleReq); err != nil {
				s.mu.Lock()
				job.Failed++
				s.mu.Unlock()
			} else {
				s.mu.Lock()
				job.Completed++
				s.mu.Unlock()
			}

			// Persist progress after each record
			s.mu.Lock()
			progressCopy := *job
			s.mu.Unlock()
			s.saveJob(&progressCopy)
		}

		s.mu.Lock()
		if job.Failed == job.Total {
			job.Status = "failed"
		} else {
			job.Status = "completed"
		}
		s.mu.Unlock()

		// Persist final state
		s.mu.Lock()
		finalCopy := *job
		s.mu.Unlock()
		s.saveJob(&finalCopy)
	}()

	return jobID, nil
}

// GetJobStatus returns the current status of an autofill batch job.
// Reads from Redis first, falls back to in-memory map.
func (s *AutofillService) GetJobStatus(jobID string) (*AutofillJob, error) {
	// Try Redis first
	if job := s.loadJob(jobID); job != nil {
		return job, nil
	}

	// Fall back to in-memory
	s.mu.Lock()
	defer s.mu.Unlock()

	job, ok := s.jobs[jobID]
	if !ok {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}
	return job, nil
}

// saveJob persists job metadata to Redis. Falls back silently if Redis is unavailable.
func (s *AutofillService) saveJob(job *AutofillJob) {
	if s.rdb == nil {
		return // already stored in-memory
	}
	ctx := context.Background()
	key := "autofill:job:" + job.ID
	s.rdb.HSet(ctx, key,
		"id", job.ID,
		"database_id", job.DatabaseID,
		"property_id", job.PropertyID,
		"total", job.Total,
		"completed", job.Completed,
		"failed", job.Failed,
		"status", job.Status,
	)
	s.rdb.Expire(ctx, key, 1*time.Hour)
}

// loadJob reads job metadata from Redis. Returns nil if not found or Redis unavailable.
func (s *AutofillService) loadJob(jobID string) *AutofillJob {
	if s.rdb == nil {
		return nil
	}
	ctx := context.Background()
	key := "autofill:job:" + jobID
	vals, err := s.rdb.HGetAll(ctx, key).Result()
	if err != nil || len(vals) == 0 {
		return nil
	}

	job := &AutofillJob{
		ID:     vals["id"],
		Status: vals["status"],
	}
	if v, err := strconv.Atoi(vals["database_id"]); err == nil {
		job.DatabaseID = uint(v)
	}
	if v, err := strconv.Atoi(vals["property_id"]); err == nil {
		job.PropertyID = uint(v)
	}
	if v, err := strconv.Atoi(vals["total"]); err == nil {
		job.Total = v
	}
	if v, err := strconv.Atoi(vals["completed"]); err == nil {
		job.Completed = v
	}
	if v, err := strconv.Atoi(vals["failed"]); err == nil {
		job.Failed = v
	}
	return job
}

// getRecordContent extracts the text content from a record's page blocks.
func (s *AutofillService) getRecordContent(record *db.Record) (string, error) {
	var blocks []db.Block
	if err := s.db.Where("page_id = ?", record.PageID).Order("position ASC").Find(&blocks).Error; err != nil {
		return "", err
	}

	var texts []string
	for _, block := range blocks {
		var props map[string]any
		if err := json.Unmarshal([]byte(block.Props), &props); err != nil {
			continue
		}
		if text, ok := props["text"].(string); ok && text != "" {
			texts = append(texts, text)
		} else if title, ok := props["title"].(string); ok && title != "" {
			texts = append(texts, title)
		}
	}

	if len(texts) == 0 {
		return "(no content)", nil
	}
	return strings.Join(texts, "\n"), nil
}

// buildPrompt creates an AI prompt based on the property type.
func (s *AutofillService) buildPrompt(name, propType, config, content string) string {
	baseContent := fmt.Sprintf("Content:\n%s", content)

	switch propType {
	case "text", "title":
		return fmt.Sprintf("Based on the record content, generate a short value for the property \"%s\". Return ONLY the value.\n\n%s", name, baseContent)

	case "select":
		options := s.extractOptions(config)
		if len(options) > 0 {
			return fmt.Sprintf("Based on the record content, choose the best option from [%s] for \"%s\". Return ONLY the option name.\n\n%s",
				strings.Join(options, ", "), name, baseContent)
		}
		return fmt.Sprintf("Based on the record content, generate a value for \"%s\". Return ONLY the value.\n\n%s", name, baseContent)

	case "number":
		return fmt.Sprintf("Extract the value for \"%s\" as a number from this content. Return ONLY the number.\n\n%s", name, baseContent)

	case "status":
		options := s.extractOptions(config)
		if len(options) > 0 {
			return fmt.Sprintf("Based on the record content, select the appropriate status from [%s] for \"%s\". Return ONLY the status name.\n\n%s",
				strings.Join(options, ", "), name, baseContent)
		}
		return fmt.Sprintf("Based on the record content, suggest a status for \"%s\". Return ONLY the status name.\n\n%s", name, baseContent)

	case "date":
		return fmt.Sprintf("Extract a relevant date for \"%s\" from this content in YYYY-MM-DD format. Return ONLY the date.\n\n%s", name, baseContent)

	case "url":
		return fmt.Sprintf("Extract a URL for \"%s\" from this content. Return ONLY the URL.\n\n%s", name, baseContent)

	case "checkbox":
		return fmt.Sprintf("Based on the record content, answer YES or NO for \"%s\". Return ONLY YES or NO.\n\n%s", name, baseContent)

	case "email":
		return fmt.Sprintf("Extract an email address for \"%s\" from this content. Return ONLY the email address.\n\n%s", name, baseContent)

	case "phone":
		return fmt.Sprintf("Extract a phone number for \"%s\" from this content. Return ONLY the phone number.\n\n%s", name, baseContent)

	default:
		return fmt.Sprintf("Based on the record content, generate a value for the property \"%s\" of type \"%s\". Return ONLY the value.\n\n%s", name, propType, baseContent)
	}
}

// extractOptions extracts option names from a property's JSON config.
func (s *AutofillService) extractOptions(config string) []string {
	if config == "" || config == "{}" {
		return nil
	}

	// Try structured options format: options as objects with name field
	var cfg struct {
		Options []struct {
			Name  string `json:"name"`
			ID    string `json:"id"`
			Color string `json:"color"`
		} `json:"options"`
	}
	if err := json.Unmarshal([]byte(config), &cfg); err == nil && len(cfg.Options) > 0 {
		var names []string
		for _, opt := range cfg.Options {
			if opt.Name != "" {
				names = append(names, opt.Name)
			}
		}
		if len(names) > 0 {
			return names
		}
	}

	// Try simple string array format
	var simpleCfg struct {
		Options []string `json:"options"`
	}
	if err := json.Unmarshal([]byte(config), &simpleCfg); err == nil && len(simpleCfg.Options) > 0 {
		return simpleCfg.Options
	}

	// Try values array format (e.g. {"values": ["a", "b"]})
	var valuesCfg struct {
		Values []string `json:"values"`
	}
	if err := json.Unmarshal([]byte(config), &valuesCfg); err == nil && len(valuesCfg.Values) > 0 {
		return valuesCfg.Values
	}

	return nil
}

// formatPropertyValue converts the AI-generated text into the correct property value JSON format.
func (s *AutofillService) formatPropertyValue(propType, generatedValue string) (string, error) {
	switch propType {
	case "text", "title":
		return s.marshalValue(map[string]string{"text": generatedValue})

	case "select":
		return s.marshalValue(map[string]string{"selected": generatedValue})

	case "number":
		num, err := strconv.ParseFloat(strings.TrimSpace(generatedValue), 64)
		if err != nil {
			// If parsing fails, store the raw value as text fallback
			return s.marshalValue(map[string]any{"number": generatedValue})
		}
		return s.marshalValue(map[string]float64{"number": num})

	case "status":
		return s.marshalValue(map[string]string{"status": generatedValue})

	case "date":
		return s.marshalValue(map[string]string{"date": generatedValue})

	case "url":
		return s.marshalValue(map[string]string{"url": generatedValue})

	case "checkbox":
		upper := strings.ToUpper(strings.TrimSpace(generatedValue))
		checked := upper == "YES" || upper == "TRUE" || upper == "Y"
		return s.marshalValue(map[string]bool{"checked": checked})

	case "email":
		return s.marshalValue(map[string]string{"email": generatedValue})

	case "phone":
		return s.marshalValue(map[string]string{"phone": generatedValue})

	default:
		return s.marshalValue(map[string]string{"text": generatedValue})
	}
}

// marshalValue is a helper that JSON-encodes a value and returns it as a string.
func (s *AutofillService) marshalValue(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
