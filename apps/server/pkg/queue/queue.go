package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// JobQueue is a general-purpose job queue interface.
// Implementations can be backed by Redis, in-memory channels, or other brokers.
type JobQueue interface {
	Enqueue(jobType string, payload []byte) error
}

const (
	searchQueue      = "search:queue"
	searchProcessing = "search:processing"
)

// SearchJob represents a search indexing job in the Redis queue.
type SearchJob struct {
	Type string `json:"type"`
	ID   uint   `json:"id"`
}

// EnqueueSearchIndex pushes a search indexing job onto the Redis queue.
func EnqueueSearchIndex(client *redis.Client, jobType string, id uint) error {
	if client == nil {
		return nil
	}
	job := SearchJob{Type: jobType, ID: id}
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal search job: %w", err)
	}
	return client.RPush(context.Background(), searchQueue, data).Err()
}

// DequeueSearchIndex atomically moves a job from the queue to a processing list
// using BLMOVE (blocking left-move). It blocks for up to 5 seconds waiting for a job.
// Returns redis.Nil when the timeout expires with no available job.
func DequeueSearchIndex(client *redis.Client) (*SearchJob, error) {
	result, err := client.BLMove(
		context.Background(),
		searchQueue,
		searchProcessing,
		"LEFT",
		"LEFT",
		5*time.Second,
	).Result()
	if err != nil {
		return nil, err
	}
	var job SearchJob
	if err := json.Unmarshal([]byte(result), &job); err != nil {
		return nil, fmt.Errorf("unmarshal search job: %w", err)
	}
	return &job, nil
}

// AckSearchIndex removes a successfully processed job from the processing list.
func AckSearchIndex(client *redis.Client, job *SearchJob) error {
	if client == nil {
		return nil
	}
	data, err := json.Marshal(job)
	if err != nil {
		return fmt.Errorf("marshal search job for ack: %w", err)
	}
	return client.LRem(context.Background(), searchProcessing, 1, string(data)).Err()
}
